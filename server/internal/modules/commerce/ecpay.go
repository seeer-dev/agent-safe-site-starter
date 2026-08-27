package commerce

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	ecpayStageEndpoint      = "https://payment-stage.ecpay.com.tw/Cashier/AioCheckOut/V5"
	ecpayProductionEndpoint = "https://payment.ecpay.com.tw/Cashier/AioCheckOut/V5"
	ecpayReturnPath         = "/api/payments/ecpay/return"
	ecpayBrowserReturnPath  = "/api/payments/ecpay/browser-return"
)

var (
	ErrECPayUnavailable        = errors.New("ecpay payment is not configured")
	ErrECPayInvalidConfig      = errors.New("invalid ecpay configuration")
	ErrECPayInvalidCallback    = errors.New("invalid ecpay callback")
	ErrECPayAmountMismatch     = errors.New("ecpay callback amount mismatch")
	ErrECPayCallbackConflict   = errors.New("ecpay callback conflicts with durable payment result")
	ErrECPayAlreadyPaid        = errors.New("order is already paid")
	ErrECPayWrongPaymentMethod = errors.New("order payment method is not ecpay")
)

type ECPayConfig struct {
	Environment      string
	MerchantID       string
	HashKey          string
	HashIV           string
	Endpoint         string
	ReturnURL        string
	BrowserReturnURL string
	SiteReturnURL    string
}

func NewECPayConfig(environment, apiOrigin, siteOrigin, merchantID, hashKey, hashIV string) (ECPayConfig, error) {
	apiOrigin, err := normalizedHTTPSOrigin(apiOrigin)
	if err != nil {
		return ECPayConfig{}, ErrECPayInvalidConfig
	}
	siteOrigin, err = normalizedHTTPSOrigin(siteOrigin)
	if err != nil {
		return ECPayConfig{}, ErrECPayInvalidConfig
	}
	cfg := ECPayConfig{
		Environment:      environment,
		MerchantID:       merchantID,
		HashKey:          hashKey,
		HashIV:           hashIV,
		ReturnURL:        apiOrigin + ecpayReturnPath,
		BrowserReturnURL: apiOrigin + ecpayBrowserReturnPath,
		SiteReturnURL:    siteOrigin + "/?payment=returned",
	}
	switch environment {
	case "stage":
		cfg.Endpoint = ecpayStageEndpoint
	case "production":
		cfg.Endpoint = ecpayProductionEndpoint
	default:
		return ECPayConfig{}, ErrECPayInvalidConfig
	}
	if strings.TrimSpace(merchantID) == "" || strings.TrimSpace(hashKey) == "" || strings.TrimSpace(hashIV) == "" {
		return ECPayConfig{}, ErrECPayInvalidConfig
	}
	return cfg, nil
}

func normalizedHTTPSOrigin(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", ErrECPayInvalidConfig
	}
	return "https://" + u.Host, nil
}

type ECPayLaunchForm struct {
	Action string            `json:"action"`
	Fields map[string]string `json:"fields"`
}

type ECPayCallbackResult struct {
	MerchantTradeNo string
	Amount          int
	MerchantID      string
	RtnCode         string
	TradeNo         string
}

func buildECPayLaunchForm(cfg ECPayConfig, merchantTradeNo string, amount int, itemName string, tradeDate time.Time) (ECPayLaunchForm, error) {
	if merchantTradeNo == "" || amount <= 0 {
		return ECPayLaunchForm{}, ErrECPayInvalidConfig
	}
	if len([]rune(itemName)) > 400 {
		itemName = string([]rune(itemName)[:400])
	}
	fields := url.Values{
		"MerchantID":        {cfg.MerchantID},
		"MerchantTradeNo":   {merchantTradeNo},
		"MerchantTradeDate": {tradeDate.In(time.FixedZone("Asia/Taipei", 8*60*60)).Format("2006/01/02 15:04:05")},
		"PaymentType":       {"aio"},
		"TotalAmount":       {strconv.Itoa(amount)},
		"TradeDesc":         {"Starter order payment"},
		"ItemName":          {itemName},
		"ReturnURL":         {cfg.ReturnURL},
		"OrderResultURL":    {cfg.BrowserReturnURL},
		"ChoosePayment":     {"Credit"},
		"EncryptType":       {"1"},
	}
	fields.Set("CheckMacValue", ecpayCheckMacValue(fields, cfg.HashKey, cfg.HashIV))
	public := make(map[string]string, len(fields))
	for k := range fields {
		public[k] = fields.Get(k)
	}
	return ECPayLaunchForm{Action: cfg.Endpoint, Fields: public}, nil
}

func verifyECPayCallback(cfg ECPayConfig, form url.Values) (ECPayCallbackResult, error) {
	required := []string{"CheckMacValue", "MerchantID", "MerchantTradeNo", "TotalAmount", "RtnCode", "TradeNo"}
	if len(form) == 0 {
		return ECPayCallbackResult{}, ErrECPayInvalidCallback
	}
	for key, values := range form {
		if len(values) != 1 {
			return ECPayCallbackResult{}, fmt.Errorf("%w: duplicate field %s", ErrECPayInvalidCallback, key)
		}
	}
	for _, key := range required {
		if strings.TrimSpace(form.Get(key)) == "" {
			return ECPayCallbackResult{}, fmt.Errorf("%w: missing %s", ErrECPayInvalidCallback, key)
		}
	}
	actual := form.Get("CheckMacValue")
	unsigned := make(url.Values, len(form)-1)
	for key, values := range form {
		if key != "CheckMacValue" {
			unsigned[key] = append([]string(nil), values...)
		}
	}
	expected := ecpayCheckMacValue(unsigned, cfg.HashKey, cfg.HashIV)
	if !hmac.Equal([]byte(expected), []byte(actual)) {
		return ECPayCallbackResult{}, ErrECPayInvalidCallback
	}
	if !hmac.Equal([]byte(cfg.MerchantID), []byte(form.Get("MerchantID"))) {
		return ECPayCallbackResult{}, ErrECPayInvalidCallback
	}
	amount, err := strconv.Atoi(form.Get("TotalAmount"))
	if err != nil || amount <= 0 {
		return ECPayCallbackResult{}, ErrECPayInvalidCallback
	}
	return ECPayCallbackResult{
		MerchantTradeNo: form.Get("MerchantTradeNo"),
		Amount:          amount,
		MerchantID:      form.Get("MerchantID"),
		RtnCode:         form.Get("RtnCode"),
		TradeNo:         form.Get("TradeNo"),
	}, nil
}

func ecpayCallbackFingerprint(form url.Values) string {
	keys := make([]string, 0, len(form))
	for key := range form {
		if key != "CheckMacValue" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(form.Get(key))
		b.WriteByte('&')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func ecpayCheckMacValue(fields url.Values, hashKey, hashIV string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if key != "CheckMacValue" {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		li, lj := strings.ToLower(keys[i]), strings.ToLower(keys[j])
		if li == lj {
			return keys[i] < keys[j]
		}
		return li < lj
	})
	var b strings.Builder
	b.WriteString("HashKey=")
	b.WriteString(hashKey)
	for _, key := range keys {
		b.WriteByte('&')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(fields.Get(key))
	}
	b.WriteString("&HashIV=")
	b.WriteString(hashIV)
	encoded := strings.ToLower(url.QueryEscape(b.String()))
	replacements := map[string]string{"%2d": "-", "%5f": "_", "%2e": ".", "%21": "!", "%2a": "*", "%28": "(", "%29": ")", "~": "%7e"}
	for from, to := range replacements {
		encoded = strings.ReplaceAll(encoded, from, to)
	}
	sum := sha256.Sum256([]byte(encoded))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

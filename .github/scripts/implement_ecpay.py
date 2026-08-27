from pathlib import Path

ROOT = Path('.')

def replace(path, old, new):
    p = ROOT / path
    text = p.read_text()
    if new in text:
        return
    if old not in text:
        raise SystemExit(f'anchor not found in {path}: {old[:80]!r}')
    p.write_text(text.replace(old, new, 1))

def write(path, content):
    p = ROOT / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content)

# Config: explicit all-or-none ECPay runtime configuration.
replace('server/internal/config/config.go', '\tCFPagesProject string\n', '\tCFPagesProject string\n\n\tECPayEnvironment string\n\tECPayMerchantID  string\n\tECPayHashKey     string\n\tECPayHashIV      string\n')
replace('server/internal/config/config.go', '\t\tCFPagesProject: os.Getenv("CF_PAGES_PROJECT"),\n', '\t\tCFPagesProject: os.Getenv("CF_PAGES_PROJECT"),\n\n\t\tECPayEnvironment: strings.ToLower(strings.TrimSpace(os.Getenv("ECPAY_ENVIRONMENT"))),\n\t\tECPayMerchantID:  strings.TrimSpace(os.Getenv("ECPAY_MERCHANT_ID")),\n\t\tECPayHashKey:     strings.TrimSpace(os.Getenv("ECPAY_HASH_KEY")),\n\t\tECPayHashIV:      strings.TrimSpace(os.Getenv("ECPAY_HASH_IV")),\n')
replace('server/internal/config/config.go', '\tdefault:\n\t\treturn fmt.Errorf("AUTH_MODE must be dev or supabase, got %q", c.AuthMode)\n\t}\n\treturn nil\n}\n\nfunc (c Config) R2Enabled() bool {', '\tdefault:\n\t\treturn fmt.Errorf("AUTH_MODE must be dev or supabase, got %q", c.AuthMode)\n\t}\n\n\tecpayValues := []string{c.ECPayEnvironment, c.ECPayMerchantID, c.ECPayHashKey, c.ECPayHashIV}\n\tconfigured := 0\n\tfor _, value := range ecpayValues {\n\t\tif strings.TrimSpace(value) != "" {\n\t\t\tconfigured++\n\t\t}\n\t}\n\tif configured != 0 && configured != len(ecpayValues) {\n\t\treturn fmt.Errorf("ECPAY_ENVIRONMENT, ECPAY_MERCHANT_ID, ECPAY_HASH_KEY, and ECPAY_HASH_IV must be configured together")\n\t}\n\tif configured == len(ecpayValues) && c.ECPayEnvironment != "stage" && c.ECPayEnvironment != "production" {\n\t\treturn fmt.Errorf("ECPAY_ENVIRONMENT must be stage or production, got %q", c.ECPayEnvironment)\n\t}\n\treturn nil\n}\n\nfunc (c Config) ECPayEnabled() bool {\n\treturn c.ECPayEnvironment != "" && c.ECPayMerchantID != "" && c.ECPayHashKey != "" && c.ECPayHashIV != ""\n}\n\nfunc (c Config) R2Enabled() bool {')

# Service owns optional concrete ECPay config; no provider registry/framework.
replace('server/internal/modules/commerce/service.go', '\tpublicBaseURL string        // R2 public base URL for deriving image URLs\n}', '\tpublicBaseURL string        // R2 public base URL for deriving image URLs\n\tecpay         *ECPayConfig  // nil when ECPay is not configured\n}')

# Persistence contract: ECPay attempt lifecycle remains commerce-owned.
replace('server/internal/modules/commerce/store.go', '\tCountOrders(ctx context.Context) (int, error)\n\n\t// Promos', '\tCountOrders(ctx context.Context) (int, error)\n\n\t// ECPay payment attempts\n\tEnsureECPayAttempt(ctx context.Context, attempt ECPayPaymentAttempt) (ECPayPaymentAttempt, error)\n\tGetECPayAttemptByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (ECPayPaymentAttempt, error)\n\tClaimECPayCallback(ctx context.Context, merchantTradeNo, callbackFingerprint, providerTradeNo, rtnCode, status string, captured bool, updatedUnix int64) (bool, error)\n\n\t// Promos')

write('server/internal/modules/commerce/ecpay.go', r'''package commerce

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
    ecpayReturnPath          = "/api/payments/ecpay/return"
    ecpayBrowserReturnPath   = "/api/payments/ecpay/browser-return"
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
        Environment: environment,
        MerchantID: merchantID,
        HashKey: hashKey,
        HashIV: hashIV,
        ReturnURL: apiOrigin + ecpayReturnPath,
        BrowserReturnURL: apiOrigin + ecpayBrowserReturnPath,
        SiteReturnURL: siteOrigin + "/?payment=returned",
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
        Amount: amount,
        MerchantID: form.Get("MerchantID"),
        RtnCode: form.Get("RtnCode"),
        TradeNo: form.Get("TradeNo"),
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
        if li == lj { return keys[i] < keys[j] }
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
    replacements := map[string]string{"%2d":"-", "%5f":"_", "%2e":".", "%21":"!", "%2a":"*", "%28":"(", "%29":")", "~":"%7e"}
    for from, to := range replacements { encoded = strings.ReplaceAll(encoded, from, to) }
    sum := sha256.Sum256([]byte(encoded))
    return strings.ToUpper(hex.EncodeToString(sum[:]))
}
''')

write('server/internal/modules/commerce/ecpay_payment.go', r'''package commerce

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "net/url"
    "strings"
    "time"
)

type ECPayPaymentAttempt struct {
    ID                  string
    OrderID             string
    MerchantTradeNo     string
    Amount              int
    Currency            string
    Status              string
    ProviderTradeNo     string
    RtnCode             string
    CallbackFingerprint string
    CreatedUnix         int64
    UpdatedUnix         int64
}

func (s Service) WithECPay(cfg ECPayConfig) Service {
    s.ecpay = &cfg
    return s
}

func (s Service) PrepareECPayPayment(ctx context.Context, orderID, accessToken string) (ECPayLaunchForm, error) {
    if s.ecpay == nil {
        return ECPayLaunchForm{}, ErrECPayUnavailable
    }
    order, err := s.store.GetOrderByAccessToken(ctx, strings.TrimSpace(orderID), strings.TrimSpace(accessToken))
    if err != nil {
        return ECPayLaunchForm{}, err
    }
    if order.PaymentStatus == "paid" {
        return ECPayLaunchForm{}, ErrECPayAlreadyPaid
    }
    methods, err := s.store.ListPaymentMethods(ctx)
    if err != nil {
        return ECPayLaunchForm{}, fmt.Errorf("load payment methods: %w", err)
    }
    valid := false
    for _, method := range methods {
        if (method.ID == order.PaymentMethod || method.Method == order.PaymentMethod) && method.Enabled && method.ReadinessStatus == "ready" && strings.EqualFold(method.Method, "ecpay") {
            valid = true
            break
        }
    }
    if !valid {
        return ECPayLaunchForm{}, ErrECPayWrongPaymentMethod
    }
    now := time.Now().Unix()
    tradeNo := merchantTradeNoForOrder(order.ID)
    attempt, err := s.store.EnsureECPayAttempt(ctx, ECPayPaymentAttempt{
        ID: tradeNo,
        OrderID: order.ID,
        MerchantTradeNo: tradeNo,
        Amount: order.Total,
        Currency: "TWD",
        Status: "prepared",
        CreatedUnix: now,
        UpdatedUnix: now,
    })
    if err != nil {
        return ECPayLaunchForm{}, err
    }
    if attempt.OrderID != order.ID || attempt.Amount != order.Total || attempt.Currency != "TWD" {
        return ECPayLaunchForm{}, ErrECPayCallbackConflict
    }
    return buildECPayLaunchForm(*s.ecpay, attempt.MerchantTradeNo, attempt.Amount, "Order "+order.ID, time.Now())
}

func (s Service) ReceiveECPayCallback(ctx context.Context, form url.Values) (string, error) {
    if s.ecpay == nil {
        return "", ErrECPayUnavailable
    }
    result, err := verifyECPayCallback(*s.ecpay, form)
    if err != nil {
        return "", err
    }
    attempt, err := s.store.GetECPayAttemptByMerchantTradeNo(ctx, result.MerchantTradeNo)
    if err != nil {
        return "", err
    }
    if result.Amount != attempt.Amount || attempt.Currency != "TWD" {
        return "", ErrECPayAmountMismatch
    }
    status := "failed"
    captured := result.RtnCode == "1"
    if captured {
        status = "captured"
    }
    _, err = s.store.ClaimECPayCallback(ctx, result.MerchantTradeNo, ecpayCallbackFingerprint(form), result.TradeNo, result.RtnCode, status, captured, time.Now().Unix())
    if err != nil {
        return "", err
    }
    return "1|OK", nil
}

func (s Service) ECPayBrowserReturn(form url.Values) (string, error) {
    if s.ecpay == nil {
        return "", ErrECPayUnavailable
    }
    if _, err := verifyECPayCallback(*s.ecpay, form); err != nil {
        return "", err
    }
    return s.ecpay.SiteReturnURL, nil
}

func merchantTradeNoForOrder(orderID string) string {
    sum := sha256.Sum256([]byte(orderID))
    return "S" + strings.ToUpper(hex.EncodeToString(sum[:]))[:19]
}
''')

write('server/internal/modules/commerce/store_ecpay.go', r'''package commerce

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "github.com/example/ai-site-starter/server/internal/platform/database"
)

func scanECPayAttempt(row interface{ Scan(...any) error }) (ECPayPaymentAttempt, error) {
    var a ECPayPaymentAttempt
    err := row.Scan(&a.ID, &a.OrderID, &a.MerchantTradeNo, &a.Amount, &a.Currency, &a.Status, &a.ProviderTradeNo, &a.RtnCode, &a.CallbackFingerprint, &a.CreatedUnix, &a.UpdatedUnix)
    if errors.Is(err, sql.ErrNoRows) { return ECPayPaymentAttempt{}, ErrNotFound }
    return a, err
}

func (s SQLStore) EnsureECPayAttempt(ctx context.Context, attempt ECPayPaymentAttempt) (ECPayPaymentAttempt, error) {
    query := database.Bind(s.dialect, `INSERT INTO ecpay_payment_attempts (id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix) VALUES (?, ?, ?, ?, ?, ?, '', '', '', ?, ?)`)
    _, err := s.db.ExecContext(ctx, query, attempt.ID, attempt.OrderID, attempt.MerchantTradeNo, attempt.Amount, attempt.Currency, attempt.Status, attempt.CreatedUnix, attempt.UpdatedUnix)
    if err == nil { return attempt, nil }
    if !database.IsUniqueViolation(err) { return ECPayPaymentAttempt{}, fmt.Errorf("insert ecpay attempt: %w", err) }
    query = database.Bind(s.dialect, `SELECT id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix FROM ecpay_payment_attempts WHERE order_id = ? LIMIT 1`)
    return scanECPayAttempt(s.db.QueryRowContext(ctx, query, attempt.OrderID))
}

func (s SQLStore) GetECPayAttemptByMerchantTradeNo(ctx context.Context, merchantTradeNo string) (ECPayPaymentAttempt, error) {
    query := database.Bind(s.dialect, `SELECT id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix FROM ecpay_payment_attempts WHERE merchant_trade_no = ? LIMIT 1`)
    return scanECPayAttempt(s.db.QueryRowContext(ctx, query, merchantTradeNo))
}

func (s SQLStore) ClaimECPayCallback(ctx context.Context, merchantTradeNo, callbackFingerprint, providerTradeNo, rtnCode, status string, captured bool, updatedUnix int64) (bool, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return false, err }
    defer func() { _ = tx.Rollback() }()

    selectQuery := database.Bind(s.dialect, `SELECT id, order_id, merchant_trade_no, amount, currency, status, provider_trade_no, rtn_code, callback_fingerprint, created_unix, updated_unix FROM ecpay_payment_attempts WHERE merchant_trade_no = ? LIMIT 1`)
    current, err := scanECPayAttempt(tx.QueryRowContext(ctx, selectQuery, merchantTradeNo))
    if err != nil { return false, err }
    if current.CallbackFingerprint != "" {
        if current.CallbackFingerprint == callbackFingerprint && current.ProviderTradeNo == providerTradeNo && current.RtnCode == rtnCode && current.Status == status {
            if err := tx.Commit(); err != nil { return false, err }
            return true, nil
        }
        return false, ErrECPayCallbackConflict
    }

    updateAttempt := database.Bind(s.dialect, `UPDATE ecpay_payment_attempts SET status = ?, provider_trade_no = ?, rtn_code = ?, callback_fingerprint = ?, updated_unix = ? WHERE merchant_trade_no = ? AND callback_fingerprint = ''`)
    res, err := tx.ExecContext(ctx, updateAttempt, status, providerTradeNo, rtnCode, callbackFingerprint, updatedUnix, merchantTradeNo)
    if err != nil { return false, err }
    affected, err := res.RowsAffected()
    if err != nil { return false, err }
    if affected == 0 {
        current, err = scanECPayAttempt(tx.QueryRowContext(ctx, selectQuery, merchantTradeNo))
        if err != nil { return false, err }
        if current.CallbackFingerprint == callbackFingerprint && current.ProviderTradeNo == providerTradeNo && current.RtnCode == rtnCode && current.Status == status {
            if err := tx.Commit(); err != nil { return false, err }
            return true, nil
        }
        return false, ErrECPayCallbackConflict
    }

    if captured {
        updateOrder := database.Bind(s.dialect, `UPDATE orders SET payment_status = 'paid', payment_intent_id = ?, version = version + 1, updated_unix = ? WHERE id = ? AND payment_status <> 'paid'`)
        orderRes, err := tx.ExecContext(ctx, updateOrder, merchantTradeNo, updatedUnix, current.OrderID)
        if err != nil { return false, err }
        orderAffected, err := orderRes.RowsAffected()
        if err != nil { return false, err }
        if orderAffected == 1 {
            var seq int
            seqQuery := database.Bind(s.dialect, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM order_events WHERE order_id = ?`)
            if err := tx.QueryRowContext(ctx, seqQuery, current.OrderID).Scan(&seq); err != nil { return false, err }
            eventQuery := database.Bind(s.dialect, `INSERT INTO order_events (id, order_id, event_type, sequence, actor_user_id, from_status, to_status, reason, created_unix) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
            eventID := current.OrderID + ":ecpay:" + callbackFingerprint[:12]
            if _, err := tx.ExecContext(ctx, eventQuery, eventID, current.OrderID, "payment_status", seq, "ecpay", "unpaid", "paid", "verified ecpay callback", updatedUnix); err != nil { return false, err }
        }
    }
    if err := tx.Commit(); err != nil { return false, err }
    return false, nil
}
''')

write('server/internal/modules/commerce/ecpay_http.go', r'''package commerce

import (
    "errors"
    "net/http"
    "strings"

    "github.com/example/ai-site-starter/server/internal/httpx"
)

func (h Handler) PrepareECPayPayment(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimSpace(r.PathValue("id"))
    token := strings.TrimSpace(r.Header.Get("X-Order-Access-Token"))
    if id == "" || token == "" {
        httpx.Error(w, http.StatusBadRequest, "order id and X-Order-Access-Token are required")
        return
    }
    launch, err := h.service.PrepareECPayPayment(r.Context(), id, token)
    if err != nil {
        switch {
        case errors.Is(err, ErrNotFound):
            httpx.Error(w, http.StatusNotFound, "order not found")
        case errors.Is(err, ErrECPayUnavailable):
            httpx.Error(w, http.StatusServiceUnavailable, err.Error())
        case errors.Is(err, ErrECPayAlreadyPaid), errors.Is(err, ErrECPayCallbackConflict):
            httpx.Error(w, http.StatusConflict, err.Error())
        case errors.Is(err, ErrECPayWrongPaymentMethod):
            httpx.Error(w, http.StatusBadRequest, err.Error())
        default:
            httpx.Error(w, http.StatusInternalServerError, "failed to prepare ecpay payment")
        }
        return
    }
    httpx.JSON(w, http.StatusOK, launch)
}

func (h Handler) ReceiveECPayReturn(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
    if err := r.ParseForm(); err != nil {
        httpx.Error(w, http.StatusBadRequest, "invalid ecpay form")
        return
    }
    ack, err := h.service.ReceiveECPayCallback(r.Context(), r.PostForm)
    if err != nil {
        switch {
        case errors.Is(err, ErrECPayInvalidCallback), errors.Is(err, ErrECPayAmountMismatch):
            httpx.Error(w, http.StatusBadRequest, "invalid ecpay callback")
        case errors.Is(err, ErrECPayCallbackConflict):
            httpx.Error(w, http.StatusConflict, err.Error())
        case errors.Is(err, ErrNotFound):
            httpx.Error(w, http.StatusNotFound, "payment attempt not found")
        case errors.Is(err, ErrECPayUnavailable):
            httpx.Error(w, http.StatusServiceUnavailable, err.Error())
        default:
            httpx.Error(w, http.StatusInternalServerError, "failed to process ecpay callback")
        }
        return
    }
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(ack))
}

func (h Handler) ECPayBrowserReturn(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
    if err := r.ParseForm(); err != nil {
        httpx.Error(w, http.StatusBadRequest, "invalid ecpay form")
        return
    }
    target, err := h.service.ECPayBrowserReturn(r.PostForm)
    if err != nil {
        httpx.Error(w, http.StatusBadRequest, "invalid ecpay browser return")
        return
    }
    http.Redirect(w, r, target, http.StatusSeeOther)
}
''')

# Bootstrap concrete config + routes.
replace('server/internal/bootstrap/app.go', '\tcommerceService := commerce.NewService(commerceStore).\n\t\tWithMediaVerifier(mediaVerifierAdapter{registry: mediaRegistry}).\n\t\tWithPublicBaseURL(cfg.R2PublicBaseURL)\n\tcommerceHandler := commerce.NewHandler(commerceService, authenticator)\n', '\tcommerceService := commerce.NewService(commerceStore).\n\t\tWithMediaVerifier(mediaVerifierAdapter{registry: mediaRegistry}).\n\t\tWithPublicBaseURL(cfg.R2PublicBaseURL)\n\tif cfg.ECPayEnabled() {\n\t\tecpayConfig, err := commerce.NewECPayConfig(cfg.ECPayEnvironment, cfg.PublicAPIBase, cfg.PublicSiteURL, cfg.ECPayMerchantID, cfg.ECPayHashKey, cfg.ECPayHashIV)\n\t\tif err != nil {\n\t\t\treturn nil, fmt.Errorf("configure ecpay: %w", err)\n\t\t}\n\t\tcommerceService = commerceService.WithECPay(ecpayConfig)\n\t}\n\tcommerceHandler := commerce.NewHandler(commerceService, authenticator)\n')
replace('server/internal/bootstrap/app.go', '\tmux.HandleFunc("GET /api/orders/mine/{id}", commerceHandler.GetMyOrder)\n', '\tmux.HandleFunc("GET /api/orders/mine/{id}", commerceHandler.GetMyOrder)\n\tmux.HandleFunc("POST /api/orders/{id}/payments/ecpay", commerceHandler.PrepareECPayPayment)\n\tmux.HandleFunc("POST /api/payments/ecpay/return", commerceHandler.ReceiveECPayReturn)\n\tmux.HandleFunc("POST /api/payments/ecpay/browser-return", commerceHandler.ECPayBrowserReturn)\n')

migration = '''CREATE TABLE IF NOT EXISTS ecpay_payment_attempts (\n  id TEXT PRIMARY KEY,\n  order_id TEXT NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,\n  merchant_trade_no TEXT NOT NULL UNIQUE,\n  amount INTEGER NOT NULL CHECK (amount > 0),\n  currency TEXT NOT NULL DEFAULT 'TWD' CHECK (currency = 'TWD'),\n  status TEXT NOT NULL DEFAULT 'prepared',\n  provider_trade_no TEXT NOT NULL DEFAULT '',\n  rtn_code TEXT NOT NULL DEFAULT '',\n  callback_fingerprint TEXT NOT NULL DEFAULT '',\n  created_unix INTEGER NOT NULL,\n  updated_unix INTEGER NOT NULL\n);\n\nCREATE INDEX IF NOT EXISTS idx_ecpay_payment_attempts_status ON ecpay_payment_attempts (status);\n'''
write('db/migrations/sqlite/017_ecpay_payment_attempts.sql', migration)
write('db/migrations/postgres/017_ecpay_payment_attempts.sql', migration)

# Frontend API: launch form helper.
replace('site/themes/minimal-cart/shared/lib/api.ts', '// Look up a guest order by ID + opaque access token.', '''export interface ECPayLaunchForm {\n  action: string\n  fields: Record<string, string>\n}\n\nexport async function prepareECPayPayment(orderId: string, accessToken: string): Promise<ECPayLaunchForm> {\n  const res = await fetch(apiUrl(`/api/orders/${encodeURIComponent(orderId)}/payments/ecpay`), {\n    method: 'POST',\n    headers: { 'X-Order-Access-Token': accessToken },\n  })\n  if (!res.ok) {\n    throw new ApiRequestError(res.status, await readErrorMessage(res, 'prepareECPayPayment'))\n  }\n  const data = await res.json()\n  if (!data || typeof data.action !== 'string' || !data.action || !data.fields || typeof data.fields !== 'object') {\n    throw new Error('綠界付款初始化回應格式不完整')\n  }\n  return data as ECPayLaunchForm\n}\n\nexport function submitHostedPayment(form: ECPayLaunchForm): void {\n  const element = document.createElement('form')\n  element.method = 'POST'\n  element.action = form.action\n  element.style.display = 'none'\n  for (const [name, value] of Object.entries(form.fields)) {\n    const input = document.createElement('input')\n    input.type = 'hidden'\n    input.name = name\n    input.value = value\n    element.appendChild(input)\n  }\n  document.body.appendChild(element)\n  element.submit()\n}\n\n// Look up a guest order by ID + opaque access token.''')

# Checkout: for ECPay, launch hosted payment only after durable order creation.
replace('site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue', '  createOrder, createOrderForMember, fetchQuote, fetchShippingMethods, fetchPaymentMethods,\n', '  createOrder, createOrderForMember, fetchQuote, fetchShippingMethods, fetchPaymentMethods,\n  prepareECPayPayment, submitHostedPayment,\n')
replace('site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue', '    placedOrder.value = order\n', '''    if (selectedPaymentMethod.value?.method.toLowerCase() === 'ecpay') {\n      if (!o.access_token || typeof o.access_token !== 'string') {\n        throw new Error('綠界付款需要訂單存取憑證，但伺服器未回傳')\n      }\n      const launch = await prepareECPayPayment(o.id, o.access_token)\n      // The order is already durable at this point. The browser only\n      // transports the signed public form to ECPay; payment truth comes\n      // back later through the verified server-to-server ReturnURL.\n      submitHostedPayment(launch)\n      return\n    }\n\n    placedOrder.value = order\n''')

# Environment examples.
for envfile in ['.env.example', '.env.development.example', '.env.production.example']:
    p = ROOT / envfile
    text = p.read_text()
    marker = 'ECPAY_ENVIRONMENT='
    if marker not in text:
        text += '''\n# ECPay AIO v5 (leave all blank to disable)\nECPAY_ENVIRONMENT=\nECPAY_MERCHANT_ID=\nECPAY_HASH_KEY=\nECPAY_HASH_IV=\n'''
        p.write_text(text)

# Controlled change docs.
write('specs/changes/commerce-ecpay-payment-flow/spec.md', '''# Commerce ECPay Payment Flow\n\n## Goal\n\nConnect durable starter orders to ECPay AIO v5 credit-card checkout without making browser navigation authoritative for payment state.\n\n## Requirements\n\n### REQ-001 Server-owned launch\nThe server MUST derive the ECPay endpoint and callback URLs from finite environment configuration, keep HashKey/HashIV server-only, and sign the hosted form.\n\n### REQ-002 Durable payment truth\nEach order MUST have at most one starter-owned ECPay payment attempt. The provider ReturnURL MUST verify CheckMacValue, MerchantID, merchant trade identity, and amount against durable state before payment_status can become paid.\n\n### REQ-003 Replay and conflict safety\nAn identical verified callback MAY be acknowledged repeatedly with one durable effect. A conflicting terminal callback MUST fail closed and MUST NOT overwrite the durable result.\n\n### REQ-004 Browser return is not payment truth\nOrderResultURL/browser return MUST verify the provider form and redirect only. It MUST NOT transition payment state.\n\n### REQ-005 Storefront handoff\nAfter a durable order is created with an enabled ready ECPay method, the minimal-cart storefront MUST request the signed launch form from the server and POST it to ECPay.\n''')
write('specs/changes/commerce-ecpay-payment-flow/plan.md', '''# Plan\n\n1. Add explicit all-or-none ECPay runtime configuration.\n2. Add SQLite/PostgreSQL parity migration for durable ECPay payment attempts.\n3. Add concrete commerce-owned ECPay signer, launch, callback verification, replay handling, and atomic paid transition.\n4. Add HTTP launch/ReturnURL/browser-return routes.\n5. Wire minimal-cart to launch ECPay only after CreateOrder succeeds.\n6. Run format, migration parity, commerce tests, config/bootstrap tests, and storefront build/contracts.\n''')
write('specs/changes/commerce-ecpay-payment-flow/evidence.md', '''# Evidence\n\n| Requirement | Proof |\n|---|---|\n| REQ-001 | `server/internal/modules/commerce/ecpay.go`, runtime wiring in `server/internal/bootstrap/app.go` |\n| REQ-002 | `db/migrations/*/017_ecpay_payment_attempts.sql`, `store_ecpay.go`, `ecpay_payment.go` |\n| REQ-003 | callback fingerprint CAS in `store_ecpay.go` |\n| REQ-004 | `Handler.ECPayBrowserReturn` delegates to verification-only `Service.ECPayBrowserReturn` |\n| REQ-005 | `shared/lib/api.ts` + `CheckoutDialog.vue` hosted form handoff |\n''')
write('specs/changes/commerce-ecpay-payment-flow/control.json', '''{\n  "change_id": "commerce-ecpay-payment-flow",\n  "revision": 1,\n  "status": "Accepted",\n  "decision_authority": "Repository owner",\n  "approval_basis": "Owner approved adapting the proven ECPay payment protocol into the independent starter project on 2026-08-27.",\n  "baseline": "8968f4943b6697a70d981ed3e5338d4584518b6f",\n  "applies_to": [\n    "server/internal/config/config.go",\n    "server/internal/modules/commerce/service.go",\n    "server/internal/modules/commerce/store.go",\n    "server/internal/modules/commerce/ecpay.go",\n    "server/internal/modules/commerce/ecpay_payment.go",\n    "server/internal/modules/commerce/store_ecpay.go",\n    "server/internal/modules/commerce/ecpay_http.go",\n    "server/internal/bootstrap/app.go",\n    "db/migrations/sqlite/017_ecpay_payment_attempts.sql",\n    "db/migrations/postgres/017_ecpay_payment_attempts.sql",\n    "site/themes/minimal-cart/shared/lib/api.ts",\n    "site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue",\n    ".env.example",\n    ".env.development.example",\n    ".env.production.example"\n  ],\n  "requirements": [\n    {"id":"REQ-001","proof":"ECPay launch configuration and signing are server-owned and secret-backed."},\n    {"id":"REQ-002","proof":"Verified ReturnURL callbacks reconcile against a durable payment attempt and order amount before paid transition."},\n    {"id":"REQ-003","proof":"Callback fingerprint compare-and-set makes identical replay one-effect and rejects conflicts."},\n    {"id":"REQ-004","proof":"Browser return verifies and redirects without payment mutation."},\n    {"id":"REQ-005","proof":"Minimal-cart launches the server-signed hosted payment form only after durable order creation."}\n  ]\n}\n''')

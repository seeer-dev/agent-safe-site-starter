package commerce

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
		ID:              tradeNo,
		OrderID:         order.ID,
		MerchantTradeNo: tradeNo,
		Amount:          order.Total,
		Currency:        "TWD",
		Status:          "prepared",
		CreatedUnix:     now,
		UpdatedUnix:     now,
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

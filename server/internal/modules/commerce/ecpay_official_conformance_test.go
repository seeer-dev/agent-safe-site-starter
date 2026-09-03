package commerce

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

func TestECPayOfficialConfigRejectsUnsupportedPublicOrigins(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{name: "explicit nonstandard port", origin: "https://api.example.com:8443"},
		{name: "direct IP", origin: "https://203.0.113.10"},
		{name: "unicode hostname", origin: "https://中文.tw"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewECPayConfig("stage", tt.origin, "https://shop.example.com", "merchant", "key", "iv")
			if !errors.Is(err, ErrECPayInvalidConfig) {
				t.Fatalf("origin %q error = %v, want ErrECPayInvalidConfig", tt.origin, err)
			}
		})
	}
}

func TestECPayOfficialConfigAcceptsStandardHTTPSOrigins(t *testing.T) {
	tests := []struct {
		name       string
		origin     string
		returnBase string
	}{
		{name: "implicit 443", origin: "https://api.example.com", returnBase: "https://api.example.com"},
		{name: "explicit 443", origin: "https://api.example.com:443", returnBase: "https://api.example.com:443"},
		{name: "punycode", origin: "https://xn--fiq228c.tw", returnBase: "https://xn--fiq228c.tw"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewECPayConfig("stage", tt.origin, "https://shop.example.com", "merchant", "key", "iv")
			if err != nil {
				t.Fatalf("NewECPayConfig: %v", err)
			}
			if cfg.ReturnURL != tt.returnBase+ecpayReturnPath {
				t.Fatalf("ReturnURL = %q", cfg.ReturnURL)
			}
		})
	}
}

func TestECPayOfficialCallbackAllowsMissingSimulatePaid(t *testing.T) {
	cfg := testECPayConfig(t)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {"S1234567890123456789"},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-real"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(form, cfg.HashKey, cfg.HashIV))
	result, err := verifyECPayCallback(cfg, form)
	if err != nil {
		t.Fatalf("verifyECPayCallback: %v", err)
	}
	if result.SimulatePaid != "" {
		t.Fatalf("SimulatePaid = %q, want empty", result.SimulatePaid)
	}
}

func TestECPayOfficialCallbackRejectsUnknownSimulatePaidValue(t *testing.T) {
	cfg := testECPayConfig(t)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {"S1234567890123456789"},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-real"},
		"SimulatePaid":    {"2"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(form, cfg.HashKey, cfg.HashIV))
	if _, err := verifyECPayCallback(cfg, form); !errors.Is(err, ErrECPayInvalidCallback) {
		t.Fatalf("error = %v, want ErrECPayInvalidCallback", err)
	}
}

func TestECPayOfficialNonSuccessDoesNotConsumeLaterSuccessClaim(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{attempt: ECPayPaymentAttempt{
		OrderID:         "ORD-1",
		MerchantTradeNo: "S1234567890123456789",
		Amount:          999,
		Currency:        "TWD",
	}}
	service := NewService(store).WithECPay(cfg)
	pending := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TradeAmt":        {"999"},
		"RtnCode":         {"10300066"},
		"TradeNo":         {"provider-pending"},
	}
	pending.Set("CheckMacValue", ecpayCheckMacValue(pending, cfg.HashKey, cfg.HashIV))
	ack, err := service.ReceiveECPayCallback(context.Background(), pending)
	if err != nil {
		t.Fatalf("pending callback: %v", err)
	}
	if ack != "1|OK" || store.claimCalls != 0 || store.captured {
		t.Fatalf("pending ack=%q claimCalls=%d captured=%v", ack, store.claimCalls, store.captured)
	}

	real := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-real"},
	}
	real.Set("CheckMacValue", ecpayCheckMacValue(real, cfg.HashKey, cfg.HashIV))
	ack, err = service.ReceiveECPayCallback(context.Background(), real)
	if err != nil {
		t.Fatalf("real callback after pending: %v", err)
	}
	if ack != "1|OK" || store.claimCalls != 1 || store.claimStatus != "captured" || !store.captured {
		t.Fatalf("real ack=%q claimCalls=%d status=%q captured=%v", ack, store.claimCalls, store.claimStatus, store.captured)
	}
}

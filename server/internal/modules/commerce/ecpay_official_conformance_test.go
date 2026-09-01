package commerce

import (
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
		{name: "explicit standard port", origin: "https://api.example.com:443"},
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

func TestECPayOfficialConfigAcceptsPunycodeHTTPSOriginWithoutExplicitPort(t *testing.T) {
	cfg, err := NewECPayConfig("stage", "https://xn--fiq228c.tw", "https://shop.example.com", "merchant", "key", "iv")
	if err != nil {
		t.Fatalf("NewECPayConfig: %v", err)
	}
	if cfg.ReturnURL != "https://xn--fiq228c.tw"+ecpayReturnPath {
		t.Fatalf("ReturnURL = %q", cfg.ReturnURL)
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

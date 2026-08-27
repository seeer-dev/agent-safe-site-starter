package commerce

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
)

func testECPayConfig(t *testing.T) ECPayConfig {
	t.Helper()
	cfg, err := NewECPayConfig("stage", "https://api.example.com", "https://shop.example.com", "3002607", "pwFHCqoQZGmho4w6", "EkRm7iFT261dpevs")
	if err != nil {
		t.Fatalf("NewECPayConfig: %v", err)
	}
	return cfg
}

func TestNewECPayConfigDerivesFiniteEndpoints(t *testing.T) {
	cfg := testECPayConfig(t)
	if cfg.Endpoint != ecpayStageEndpoint {
		t.Fatalf("endpoint = %q", cfg.Endpoint)
	}
	if cfg.ReturnURL != "https://api.example.com"+ecpayReturnPath {
		t.Fatalf("ReturnURL = %q", cfg.ReturnURL)
	}
	if cfg.BrowserReturnURL != "https://api.example.com"+ecpayBrowserReturnPath {
		t.Fatalf("BrowserReturnURL = %q", cfg.BrowserReturnURL)
	}
	if cfg.SiteReturnURL != "https://shop.example.com/?payment=returned" {
		t.Fatalf("SiteReturnURL = %q", cfg.SiteReturnURL)
	}
}

func TestNewECPayConfigRejectsNonHTTPSOrigin(t *testing.T) {
	_, err := NewECPayConfig("stage", "http://api.example.com", "https://shop.example.com", "merchant", "key", "iv")
	if !errors.Is(err, ErrECPayInvalidConfig) {
		t.Fatalf("error = %v, want ErrECPayInvalidConfig", err)
	}
}

func TestECPayLaunchFormSignsAndCallbackVerificationRejectsTamper(t *testing.T) {
	cfg := testECPayConfig(t)
	launch, err := buildECPayLaunchForm(cfg, "S1234567890123456789", 1234, "Order test", time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildECPayLaunchForm: %v", err)
	}
	if launch.Fields["CheckMacValue"] == "" {
		t.Fatal("CheckMacValue must be present")
	}
	if _, ok := launch.Fields["HashKey"]; ok {
		t.Fatal("HashKey leaked into public launch fields")
	}
	if _, ok := launch.Fields["HashIV"]; ok {
		t.Fatal("HashIV leaked into public launch fields")
	}

	form := url.Values{}
	for key, value := range launch.Fields {
		form.Set(key, value)
	}
	form.Set("RtnCode", "1")
	form.Set("TradeNo", "240827000000001")
	form.Set("CheckMacValue", ecpayCheckMacValue(withoutCheckMac(form), cfg.HashKey, cfg.HashIV))
	result, err := verifyECPayCallback(cfg, form)
	if err != nil {
		t.Fatalf("verifyECPayCallback: %v", err)
	}
	if result.Amount != 1234 || result.RtnCode != "1" {
		t.Fatalf("result = %#v", result)
	}

	form.Set("TotalAmount", "1")
	if _, err := verifyECPayCallback(cfg, form); !errors.Is(err, ErrECPayInvalidCallback) {
		t.Fatalf("tampered callback error = %v", err)
	}
}

func withoutCheckMac(form url.Values) url.Values {
	out := url.Values{}
	for key, values := range form {
		if key == "CheckMacValue" {
			continue
		}
		out[key] = append([]string(nil), values...)
	}
	return out
}

func TestMerchantTradeNoIsStableAndProviderSafe(t *testing.T) {
	a := merchantTradeNoForOrder("ORD-very-long-order-id-1234567890")
	b := merchantTradeNoForOrder("ORD-very-long-order-id-1234567890")
	if a != b {
		t.Fatalf("trade number not stable: %q != %q", a, b)
	}
	if len(a) != 20 {
		t.Fatalf("length = %d, want 20", len(a))
	}
}

type ecpayFakeStore struct {
	Store
	order       Order
	attempt     ECPayPaymentAttempt
	claimCalls  int
	claimStatus string
	captured    bool
}

func (f *ecpayFakeStore) GetOrderByAccessToken(context.Context, string, string) (Order, error) {
	return f.order, nil
}

func (f *ecpayFakeStore) ListPaymentMethods(context.Context) ([]PaymentMethod, error) {
	return []PaymentMethod{{ID: "pm-ecpay", Method: "ecpay", Enabled: true, ReadinessStatus: "ready"}}, nil
}

func (f *ecpayFakeStore) EnsureECPayAttempt(_ context.Context, attempt ECPayPaymentAttempt) (ECPayPaymentAttempt, error) {
	if f.attempt.OrderID == "" {
		f.attempt = attempt
	}
	return f.attempt, nil
}

func (f *ecpayFakeStore) GetECPayAttemptByMerchantTradeNo(context.Context, string) (ECPayPaymentAttempt, error) {
	if f.attempt.OrderID == "" {
		return ECPayPaymentAttempt{}, ErrNotFound
	}
	return f.attempt, nil
}

func (f *ecpayFakeStore) ClaimECPayCallback(_ context.Context, _, _, _, _, status string, captured bool, _ int64) (bool, error) {
	f.claimCalls++
	f.claimStatus = status
	f.captured = captured
	return f.claimCalls > 1, nil
}

func TestPrepareECPayPaymentUsesDurableOrderAmount(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{order: Order{ID: "ORD-1", Total: 999, PaymentStatus: "unpaid", PaymentMethod: "pm-ecpay"}}
	service := NewService(store).WithECPay(cfg)
	launch, err := service.PrepareECPayPayment(context.Background(), "ORD-1", "secret")
	if err != nil {
		t.Fatalf("PrepareECPayPayment: %v", err)
	}
	if launch.Fields["TotalAmount"] != "999" {
		t.Fatalf("TotalAmount = %q", launch.Fields["TotalAmount"])
	}
	if store.attempt.Amount != 999 || store.attempt.OrderID != "ORD-1" {
		t.Fatalf("attempt = %#v", store.attempt)
	}
}

func TestReceiveECPayCallbackRejectsAmountMismatchBeforeClaim(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{attempt: ECPayPaymentAttempt{OrderID: "ORD-1", MerchantTradeNo: "S1234567890123456789", Amount: 999, Currency: "TWD"}}
	service := NewService(store).WithECPay(cfg)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TotalAmount":     {"998"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-1"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(form, cfg.HashKey, cfg.HashIV))
	if _, err := service.ReceiveECPayCallback(context.Background(), form); !errors.Is(err, ErrECPayAmountMismatch) {
		t.Fatalf("error = %v, want ErrECPayAmountMismatch", err)
	}
	if store.claimCalls != 0 {
		t.Fatalf("claim calls = %d, want 0", store.claimCalls)
	}
}

func TestReceiveECPayCallbackCapturesOnlyVerifiedSuccess(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{attempt: ECPayPaymentAttempt{OrderID: "ORD-1", MerchantTradeNo: "S1234567890123456789", Amount: 999, Currency: "TWD"}}
	service := NewService(store).WithECPay(cfg)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TotalAmount":     {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-1"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(form, cfg.HashKey, cfg.HashIV))
	ack, err := service.ReceiveECPayCallback(context.Background(), form)
	if err != nil {
		t.Fatalf("ReceiveECPayCallback: %v", err)
	}
	if ack != "1|OK" || store.claimStatus != "captured" || !store.captured {
		t.Fatalf("ack=%q status=%q captured=%v", ack, store.claimStatus, store.captured)
	}
}

func TestECPayBrowserReturnVerifiesWithoutStoreMutation(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{}
	service := NewService(store).WithECPay(cfg)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {"S1234567890123456789"},
		"TotalAmount":     {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-1"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(form, cfg.HashKey, cfg.HashIV))
	target, err := service.ECPayBrowserReturn(form)
	if err != nil {
		t.Fatalf("ECPayBrowserReturn: %v", err)
	}
	if target != cfg.SiteReturnURL {
		t.Fatalf("target = %q", target)
	}
	if store.claimCalls != 0 {
		t.Fatalf("browser return mutated payment: claim calls=%d", store.claimCalls)
	}
}

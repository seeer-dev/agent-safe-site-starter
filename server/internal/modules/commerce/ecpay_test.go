package commerce

import (
	"context"
	"errors"
	"net/url"
	"strings"
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

func TestNewECPayConfigRejectsExplicitNon443Port(t *testing.T) {
	_, err := NewECPayConfig("stage", "https://api.example.com:8443", "https://shop.example.com", "merchant", "key", "iv")
	if !errors.Is(err, ErrECPayInvalidConfig) {
		t.Fatalf("error = %v, want ErrECPayInvalidConfig", err)
	}
}

func TestECPayCheckMacValueMatchesOfficialSHA256Vectors(t *testing.T) {
	const (
		hashKey = "pwFHCqoQZGmho4w6"
		hashIV  = "EkRm7iFT261dpevs"
	)
	tests := []struct {
		name     string
		params   url.Values
		expected string
	}{
		{
			name: "baseline AIO",
			params: url.Values{
				"MerchantID":        {"3002607"},
				"MerchantTradeNo":   {"Test1234567890"},
				"MerchantTradeDate": {"2025/01/01 12:00:00"},
				"PaymentType":       {"aio"},
				"TotalAmount":       {"100"},
				"TradeDesc":         {"測試"},
				"ItemName":          {"測試商品"},
				"ReturnURL":         {"https://example.com/notify"},
				"ChoosePayment":     {"ALL"},
				"EncryptType":       {"1"},
			},
			expected: "291CBA324D31FB5A4BBBFDF2CFE5D32598524753AFD4959C3BF590C5B2F57FB2",
		},
		{
			name: "apostrophe",
			params: url.Values{
				"MerchantID":  {"3002607"},
				"ItemName":    {"Tom's Shop"},
				"TotalAmount": {"100"},
			},
			expected: "CF0A3D4901D99459D8641516EC57210700E8A5C9AB26B1D021301E9CB93EF78D",
		},
		{
			name: "tilde",
			params: url.Values{
				"MerchantID":  {"3002607"},
				"ItemName":    {"Test~Product"},
				"TotalAmount": {"200"},
			},
			expected: "CEEAE01D2F9A8E74D4AC0DCE7735B046D73F35A5EC99558A31A2EE03159DA1C9",
		},
		{
			name: "spaces",
			params: url.Values{
				"MerchantID":  {"3002607"},
				"ItemName":    {"My Test Product"},
				"TotalAmount": {"300"},
			},
			expected: "7712A5E6EDC3B57086063C88568084C66CE882A21D40E74DE5ACA3B478C6F316",
		},
		{
			name: "callback TradeAmt",
			params: url.Values{
				"MerchantID":      {"3002607"},
				"MerchantTradeNo": {"Test1234567890"},
				"RtnCode":         {"1"},
				"RtnMsg":          {"Succeeded"},
				"TradeNo":         {"2301011234567890"},
				"TradeAmt":        {"100"},
				"PaymentDate":     {"2025/01/01 12:05:00"},
				"PaymentType":     {"Credit_CreditCard"},
				"TradeDate":       {"2025/01/01 12:00:00"},
				"SimulatePaid":    {"0"},
			},
			expected: "2AB536D86AFF8E1086744D59175040A32538C96B1C28C4135B551BD728E913B8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ecpayCheckMacValue(tt.params, hashKey, hashIV); got != tt.expected {
				t.Fatalf("CheckMacValue = %q, want %q", got, tt.expected)
			}
		})
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

	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {"S1234567890123456789"},
		"TradeAmt":        {"1234"},
		"RtnCode":         {"1"},
		"TradeNo":         {"240827000000001"},
		"SimulatePaid":    {"0"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(withoutCheckMac(form), cfg.HashKey, cfg.HashIV))
	result, err := verifyECPayCallback(cfg, form)
	if err != nil {
		t.Fatalf("verifyECPayCallback: %v", err)
	}
	if result.Amount != 1234 || result.RtnCode != "1" || result.SimulatePaid != "0" {
		t.Fatalf("result = %#v", result)
	}

	form.Set("TradeAmt", "1")
	if _, err := verifyECPayCallback(cfg, form); !errors.Is(err, ErrECPayInvalidCallback) {
		t.Fatalf("tampered callback error = %v", err)
	}
}

func TestECPayLaunchFormCapsItemNameAtOfficialRecommendedLength(t *testing.T) {
	cfg := testECPayConfig(t)
	launch, err := buildECPayLaunchForm(cfg, "S1234567890123456789", 100, strings.Repeat("界", 250), time.Now())
	if err != nil {
		t.Fatalf("buildECPayLaunchForm: %v", err)
	}
	if got := len([]rune(launch.Fields["ItemName"])); got != 200 {
		t.Fatalf("ItemName rune length = %d, want 200", got)
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
	order              Order
	attempt            ECPayPaymentAttempt
	paymentEnvironment string
	claimCalls         int
	claimStatus        string
	captured           bool
}

func (f *ecpayFakeStore) GetOrderByAccessToken(context.Context, string, string) (Order, error) {
	return f.order, nil
}

func (f *ecpayFakeStore) ListPaymentMethods(context.Context) ([]PaymentMethod, error) {
	environment := f.paymentEnvironment
	if environment == "" {
		environment = "sandbox"
	}
	return []PaymentMethod{{ID: "pm-ecpay", Method: "ecpay", Environment: environment, Enabled: true, ReadinessStatus: "ready"}}, nil
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

func TestPrepareECPayPaymentRejectsRuntimeAdminEnvironmentMismatch(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{
		order:              Order{ID: "ORD-1", Total: 999, PaymentStatus: "unpaid", PaymentMethod: "pm-ecpay"},
		paymentEnvironment: "production",
	}
	service := NewService(store).WithECPay(cfg)
	if _, err := service.PrepareECPayPayment(context.Background(), "ORD-1", "secret"); !errors.Is(err, ErrECPayWrongPaymentMethod) {
		t.Fatalf("error = %v, want ErrECPayWrongPaymentMethod", err)
	}
	if store.attempt.OrderID != "" {
		t.Fatalf("mismatched environment created payment attempt: %#v", store.attempt)
	}
}

func TestReceiveECPayCallbackRejectsAmountMismatchBeforeClaim(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{attempt: ECPayPaymentAttempt{OrderID: "ORD-1", MerchantTradeNo: "S1234567890123456789", Amount: 999, Currency: "TWD"}}
	service := NewService(store).WithECPay(cfg)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TradeAmt":        {"998"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-1"},
		"SimulatePaid":    {"0"},
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
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-1"},
		"SimulatePaid":    {"0"},
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

func TestReceiveECPayCallbackDoesNotClaimSimulatedPayment(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{attempt: ECPayPaymentAttempt{OrderID: "ORD-1", MerchantTradeNo: "S1234567890123456789", Amount: 999, Currency: "TWD"}}
	service := NewService(store).WithECPay(cfg)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-simulated"},
		"SimulatePaid":    {"1"},
	}
	form.Set("CheckMacValue", ecpayCheckMacValue(form, cfg.HashKey, cfg.HashIV))
	ack, err := service.ReceiveECPayCallback(context.Background(), form)
	if err != nil {
		t.Fatalf("ReceiveECPayCallback: %v", err)
	}
	if ack != "1|OK" || store.claimCalls != 0 || store.captured {
		t.Fatalf("ack=%q claimCalls=%d captured=%v", ack, store.claimCalls, store.captured)
	}
}

func TestRealCallbackCanCaptureAfterSimulatedNotification(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{attempt: ECPayPaymentAttempt{OrderID: "ORD-1", MerchantTradeNo: "S1234567890123456789", Amount: 999, Currency: "TWD"}}
	service := NewService(store).WithECPay(cfg)
	simulated := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-simulated"},
		"SimulatePaid":    {"1"},
	}
	simulated.Set("CheckMacValue", ecpayCheckMacValue(simulated, cfg.HashKey, cfg.HashIV))
	if _, err := service.ReceiveECPayCallback(context.Background(), simulated); err != nil {
		t.Fatalf("simulated callback: %v", err)
	}
	if store.claimCalls != 0 {
		t.Fatalf("simulated callback consumed claim: %d", store.claimCalls)
	}

	real := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {store.attempt.MerchantTradeNo},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-real"},
	}
	real.Set("CheckMacValue", ecpayCheckMacValue(real, cfg.HashKey, cfg.HashIV))
	ack, err := service.ReceiveECPayCallback(context.Background(), real)
	if err != nil {
		t.Fatalf("real callback: %v", err)
	}
	if ack != "1|OK" || store.claimCalls != 1 || store.claimStatus != "captured" || !store.captured {
		t.Fatalf("ack=%q claimCalls=%d status=%q captured=%v", ack, store.claimCalls, store.claimStatus, store.captured)
	}
}

func TestECPayBrowserReturnVerifiesWithoutStoreMutation(t *testing.T) {
	cfg := testECPayConfig(t)
	store := &ecpayFakeStore{}
	service := NewService(store).WithECPay(cfg)
	form := url.Values{
		"MerchantID":      {cfg.MerchantID},
		"MerchantTradeNo": {"S1234567890123456789"},
		"TradeAmt":        {"999"},
		"RtnCode":         {"1"},
		"TradeNo":         {"provider-1"},
		"SimulatePaid":    {"0"},
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

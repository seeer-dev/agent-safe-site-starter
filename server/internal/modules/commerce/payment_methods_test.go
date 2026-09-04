package commerce

import (
	"context"
	"errors"
	"testing"
)

type paymentAvailabilityStore struct {
	Store
	methods []PaymentMethod
}

func (s *paymentAvailabilityStore) ListPaymentMethods(context.Context) ([]PaymentMethod, error) {
	return append([]PaymentMethod(nil), s.methods...), nil
}

func readyECPayRow(environment string) PaymentMethod {
	return PaymentMethod{
		ID:              "pm-ecpay",
		Method:          "ecpay",
		ProviderLabel:   "綠界信用卡",
		Environment:     environment,
		ReadinessStatus: "ready",
		Enabled:         true,
	}
}

func TestECPayReadyRowIsHiddenWithoutRuntimeConfig(t *testing.T) {
	store := &paymentAvailabilityStore{methods: []PaymentMethod{
		readyECPayRow("sandbox"),
		{ID: "pm-cod", Method: "cod", ProviderLabel: "貨到付款", Environment: "production", ReadinessStatus: "ready", Enabled: true},
	}}
	service := NewService(store)

	public, err := service.ListPublicPaymentMethods(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPaymentMethods: %v", err)
	}
	if len(public) != 1 || public[0].Method != "cod" {
		t.Fatalf("public methods = %#v, want only cod", public)
	}
	if err := service.validatePaymentMethod(context.Background(), "pm-ecpay"); !errors.Is(err, ErrInvalidPaymentMethod) {
		t.Fatalf("validate ECPay error = %v, want ErrInvalidPaymentMethod", err)
	}
}

func TestECPayRuntimeEnvironmentMustMatchAdminEnvironment(t *testing.T) {
	stageCfg, err := NewECPayConfig(
		"stage",
		"https://api.example.com",
		"https://shop.example.com",
		"3002607",
		"pwFHCqoQZGmho4w6",
		"EkRm7iFT261dpevs",
	)
	if err != nil {
		t.Fatalf("NewECPayConfig(stage): %v", err)
	}

	productionCfg, err := NewECPayConfig(
		"production",
		"https://api.example.com",
		"https://shop.example.com",
		"merchant-production",
		"production-key",
		"production-iv",
	)
	if err != nil {
		t.Fatalf("NewECPayConfig(production): %v", err)
	}

	tests := []struct {
		name        string
		rowEnv      string
		cfg         ECPayConfig
		wantVisible bool
	}{
		{name: "stage matches sandbox", rowEnv: "sandbox", cfg: stageCfg, wantVisible: true},
		{name: "stage rejects production row", rowEnv: "production", cfg: stageCfg, wantVisible: false},
		{name: "production matches production", rowEnv: "production", cfg: productionCfg, wantVisible: true},
		{name: "production rejects sandbox row", rowEnv: "sandbox", cfg: productionCfg, wantVisible: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &paymentAvailabilityStore{methods: []PaymentMethod{readyECPayRow(tt.rowEnv)}}
			service := NewService(store).WithECPay(tt.cfg)

			public, err := service.ListPublicPaymentMethods(context.Background())
			if err != nil {
				t.Fatalf("ListPublicPaymentMethods: %v", err)
			}
			if got := len(public) == 1; got != tt.wantVisible {
				t.Fatalf("visible = %v, methods=%#v, want %v", got, public, tt.wantVisible)
			}
			validationErr := service.validatePaymentMethod(context.Background(), "pm-ecpay")
			if tt.wantVisible && validationErr != nil {
				t.Fatalf("validate matching ECPay: %v", validationErr)
			}
			if !tt.wantVisible && !errors.Is(validationErr, ErrInvalidPaymentMethod) {
				t.Fatalf("validate mismatched ECPay error = %v, want ErrInvalidPaymentMethod", validationErr)
			}
		})
	}
}

func TestNonECPayReadyMethodDoesNotRequireRuntimeAdapter(t *testing.T) {
	store := &paymentAvailabilityStore{methods: []PaymentMethod{
		{ID: "pm-atm", Method: "atm", ProviderLabel: "ATM", Environment: "production", ReadinessStatus: "ready", Enabled: true},
	}}
	service := NewService(store)

	public, err := service.ListPublicPaymentMethods(context.Background())
	if err != nil {
		t.Fatalf("ListPublicPaymentMethods: %v", err)
	}
	if len(public) != 1 || public[0].Method != "atm" {
		t.Fatalf("public methods = %#v", public)
	}
	if err := service.validatePaymentMethod(context.Background(), "pm-atm"); err != nil {
		t.Fatalf("validate ATM: %v", err)
	}
}

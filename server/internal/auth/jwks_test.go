package auth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type staticStaffLookup struct {
	row authStaffRowStub
	err error
}

type authStaffRowStub struct {
	ID             string
	DisplayName    string
	Email          string
	RoleLabel      string
	SupabaseUserID string
	Status         string
}

func (s staticStaffLookup) GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (StaffRow, error) {
	if s.err != nil {
		return StaffRow{}, s.err
	}
	return StaffRow{
		ID:             s.row.ID,
		DisplayName:    s.row.DisplayName,
		Email:          s.row.Email,
		RoleLabel:      s.row.RoleLabel,
		SupabaseUserID: s.row.SupabaseUserID,
		Status:         s.row.Status,
	}, nil
}

func createTestECKey(t *testing.T, kid string) (*ecdsa.PrivateKey, jwk.Key) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}
	pubKey, err := jwk.Import(priv.PublicKey)
	if err != nil {
		t.Fatalf("import EC pubkey: %v", err)
	}
	_ = pubKey.Set(jwk.KeyIDKey, kid)
	_ = pubKey.Set(jwk.AlgorithmKey, "ES256")
	_ = pubKey.Set(jwk.KeyUsageKey, "sig")
	return priv, pubKey
}

func createTestRSAKey(t *testing.T, kid string, bits int) (*rsa.PrivateKey, jwk.Key) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	pubKey, err := jwk.Import(priv.PublicKey)
	if err != nil {
		t.Fatalf("import RSA pubkey: %v", err)
	}
	_ = pubKey.Set(jwk.KeyIDKey, kid)
	_ = pubKey.Set(jwk.AlgorithmKey, "RS256")
	_ = pubKey.Set(jwk.KeyUsageKey, "sig")
	return priv, pubKey
}

func buildJWKSJSON(t *testing.T, keys ...jwk.Key) []byte {
	t.Helper()
	set := jwk.NewSet()
	for _, k := range keys {
		if err := set.AddKey(k); err != nil {
			t.Fatalf("add key to set: %v", err)
		}
	}
	buf, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	return buf
}

func signTestToken(
	t *testing.T,
	issuer string,
	alg jwa.SignatureAlgorithm,
	privKey any,
	kid string,
	mutateClaims func(b *jwt.Builder),
	mutateHeaders func(h jws.Headers),
) string {
	return signTestTokenWithTime(t, issuer, alg, privKey, kid, time.Time{}, mutateClaims, mutateHeaders)
}

func signTestTokenWithTime(
	t *testing.T,
	issuer string,
	alg jwa.SignatureAlgorithm,
	privKey any,
	kid string,
	baseTime time.Time,
	mutateClaims func(b *jwt.Builder),
	mutateHeaders func(h jws.Headers),
) string {
	t.Helper()
	if baseTime.IsZero() {
		baseTime = time.Now()
	}
	builder := jwt.NewBuilder().
		Issuer(issuer).
		Audience([]string{"authenticated"}).
		Subject("test-user-id").
		Claim("email", "test@example.com").
		Expiration(baseTime.Add(1 * time.Hour)).
		IssuedAt(baseTime)

	if mutateClaims != nil {
		mutateClaims(builder)
	}

	tok, err := builder.Build()
	if err != nil {
		t.Fatalf("build token: %v", err)
	}

	hdrs := jws.NewHeaders()
	if kid != "" {
		_ = hdrs.Set(jws.KeyIDKey, kid)
	}
	if mutateHeaders != nil {
		mutateHeaders(hdrs)
	}

	signed, err := jwt.Sign(tok, jwt.WithKey(alg, privKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return string(signed)
}

func TestJWKSVerifierSuccessBothAlgorithms(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	rsaPriv, rsaPub := createTestRSAKey(t, "rsa-1", 2048)

	jwksBytes := buildJWKSJSON(t, ecPub, rsaPub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/v1/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(jwksBytes)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	// 1. Test ES256
	ecToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)
	pEC, err := verifier.Verify(context.Background(), ecToken)
	if err != nil {
		t.Fatalf("verify ES256: %v", err)
	}
	if pEC.UserID != "test-user-id" || pEC.Email != "test@example.com" || pEC.Role != "user" {
		t.Errorf("unexpected ES256 principal: %+v", pEC)
	}

	// 2. Test RS256
	rsaToken := signTestToken(t, server.URL+"/auth/v1", jwa.RS256(), rsaPriv, "rsa-1", nil, nil)
	pRSA, err := verifier.Verify(context.Background(), rsaToken)
	if err != nil {
		t.Fatalf("verify RS256: %v", err)
	}
	if pRSA.UserID != "test-user-id" || pRSA.Email != "test@example.com" || pRSA.Role != "user" {
		t.Errorf("unexpected RS256 principal: %+v", pRSA)
	}
}

func TestJWKSVerifierWarmCacheAvoidsNetwork(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	var fetchCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/v1/.well-known/jwks.json" {
			atomic.AddInt64(&fetchCount, 1)
			w.Header().Set("Content-Type", "application/json")
			w.Write(jwksBytes)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	token := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)

	// Call 1: cold cache -> 1 fetch
	p1, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify call 1: %v", err)
	}
	if p1.UserID != "test-user-id" {
		t.Errorf("unexpected user ID: %s", p1.UserID)
	}
	if got := atomic.LoadInt64(&fetchCount); got != 1 {
		t.Fatalf("expected 1 fetch, got %d", got)
	}

	// Call 2: warm cache -> 0 additional fetches
	p2, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify call 2: %v", err)
	}
	if p2.UserID != "test-user-id" {
		t.Errorf("unexpected user ID: %s", p2.UserID)
	}
	if got := atomic.LoadInt64(&fetchCount); got != 1 {
		t.Fatalf("expected 1 fetch across warm calls, got %d", got)
	}
}

func TestJWKSVerifierVerifiedClaimsDoNotGrantCapabilities(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	// Inject arbitrary capability / role claims into JWT
	token := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
		b.Claim("role", "owner")
		b.Claim("capabilities", []string{"twcommerce.admin", "content.publish"})
		b.Claim("app_metadata", map[string]any{"role": "owner"})
		b.Claim("user_metadata", map[string]any{"is_staff": true})
	}, nil)

	// Direct verifier output must have Role="user" and nil/empty Capabilities
	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if principal.Role != "user" {
		t.Errorf("verifier leaked token role: %q", principal.Role)
	}
	if len(principal.Capabilities) > 0 {
		t.Errorf("verifier leaked token capabilities: %+v", principal.Capabilities)
	}

	// Wire into Authenticator with resolver
	lookup := staticStaffLookup{
		row: authStaffRowStub{
			ID:             "staff-99",
			DisplayName:    "Manager Bob",
			Email:          "bob@example.com",
			RoleLabel:      "readonly",
			SupabaseUserID: "test-user-id",
			Status:         "active",
		},
	}
	resolver := NewStaffCapabilityResolver(lookup)
	authenticator := NewAuthenticatorWithResolver(verifier, resolver)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	p, err := authenticator.Principal(req)
	if err != nil {
		t.Fatalf("authenticator principal: %v", err)
	}
	// Role and capabilities must come exclusively from staff resolver (which gave "readonly"), NOT token ("owner")
	if p.Role != "readonly" {
		t.Errorf("role must be resolved from staff row ('readonly'), got %q", p.Role)
	}
	if Can(p, "content.publish") {
		t.Errorf("readonly staff must NOT have content.publish despite token claims")
	}
}

func TestJWKSVerifierRejectsInvalidTokens(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	otherPriv, _ := createTestECKey(t, "ec-other")
	jwksBytes := buildJWKSJSON(t, ecPub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	baseIssuer := server.URL + "/auth/v1"

	tests := []struct {
		name      string
		makeToken func() string
	}{
		{
			name: "missing kid header",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "", nil, nil)
			},
		},
		{
			name: "unknown kid not in JWKS",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "unknown-kid-xyz", nil, nil)
			},
		},
		{
			name: "wrong issuer",
			makeToken: func() string {
				return signTestToken(t, "https://wrong.supabase.co/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)
			},
		},
		{
			name: "missing audience",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Audience(nil)
				}, nil)
			},
		},
		{
			name: "wrong audience (anon instead of authenticated)",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Audience([]string{"anon"})
				}, nil)
			},
		},
		{
			name: "missing exp claim",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Expiration(time.Time{})
				}, nil)
			},
		},
		{
			name: "expired exp claim (expired 2 hours ago)",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Expiration(time.Now().Add(-2 * time.Hour))
				}, nil)
			},
		},
		{
			name: "future nbf claim (valid in 2 hours)",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.NotBefore(time.Now().Add(2 * time.Hour))
				}, nil)
			},
		},
		{
			name: "missing sub claim",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Subject("")
				}, nil)
			},
		},
		{
			name: "empty whitespace sub claim",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Subject("   ")
				}, nil)
			},
		},
		{
			name: "bad signature (signed with different key under same kid)",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), otherPriv, "ec-1", nil, nil)
			},
		},
		{
			name: "oversized token > 16 KiB",
			makeToken: func() string {
				largePadding := strings.Repeat("A", 17000)
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Claim("padding", largePadding)
				}, nil)
			},
		},
		{
			name: "malformed token string",
			makeToken: func() string {
				return "not-a-jwt.compact.string"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tok := tc.makeToken()
			_, err := verifier.Verify(context.Background(), tok)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("expected ErrUnauthorized, got %v", err)
			}
		})
	}
}

func TestJWKSVerifierRejectsSymmetricAndUnsupportedAlgorithms(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	baseIssuer := server.URL + "/auth/v1"

	// 1. HS256 symmetric token
	symKey := []byte("symmetric-secret-key-32-bytes-long!")
	builder := jwt.NewBuilder().
		Issuer(baseIssuer).
		Audience([]string{"authenticated"}).
		Subject("user-123").
		Expiration(time.Now().Add(1 * time.Hour))
	tok, _ := builder.Build()
	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jws.KeyIDKey, "ec-1")
	hsToken, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256(), symKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		t.Fatalf("sign hs256: %v", err)
	}

	_, err = verifier.Verify(context.Background(), string(hsToken))
	if err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for HS256, got %v", err)
	}

	// 2. ES384 unsupported asymmetric algorithm
	priv384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("gen P-384: %v", err)
	}
	es384Token := signTestToken(t, baseIssuer, jwa.ES384(), priv384, "ec-1", nil, nil)
	_, err = verifier.Verify(context.Background(), es384Token)
	if err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for ES384, got %v", err)
	}

	// 3. Algorithm mismatch: header says RS256, but kid is an EC key in JWKS
	validECSigned := signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", nil, nil)
	parts := strings.Split(validECSigned, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in compact token, got %d", len(parts))
	}
	rawHeader := `{"alg":"RS256","kid":"ec-1"}`
	encodedHeader := base64.RawURLEncoding.EncodeToString([]byte(rawHeader))
	mismatchedAlgToken := encodedHeader + "." + parts[1] + "." + parts[2]
	_, err = verifier.Verify(context.Background(), mismatchedAlgToken)
	if err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for mismatched algorithm, got %v", err)
	}
}

func TestJWKSVerifierClockSkewAllowance(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	fixedTime := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }

	verifier, err := newSupabaseJWKSVerifier(server.URL, nil, clock)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	baseIssuer := server.URL + "/auth/v1"

	// 1. Expired 30 seconds ago (fixedTime - 30s) -> Valid because within 60s skew
	tokExp30sAgo := signTestTokenWithTime(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", fixedTime, func(b *jwt.Builder) {
		b.Expiration(fixedTime.Add(-30 * time.Second))
	}, nil)
	if _, err := verifier.Verify(context.Background(), tokExp30sAgo); err != nil {
		t.Errorf("expected token expired 30s ago to pass with 60s skew, got %v", err)
	}

	// 2. Expired 65 seconds ago (fixedTime - 65s) -> Invalid (outside 60s skew)
	tokExp65sAgo := signTestTokenWithTime(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", fixedTime, func(b *jwt.Builder) {
		b.Expiration(fixedTime.Add(-65 * time.Second))
	}, nil)
	if _, err := verifier.Verify(context.Background(), tokExp65sAgo); err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for token expired 65s ago, got %v", err)
	}

	// 3. Not valid until 30 seconds in future (fixedTime + 30s) -> Valid because within 60s skew
	tokNbf30sFuture := signTestTokenWithTime(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", fixedTime, func(b *jwt.Builder) {
		b.NotBefore(fixedTime.Add(30 * time.Second))
	}, nil)
	if _, err := verifier.Verify(context.Background(), tokNbf30sFuture); err != nil {
		t.Errorf("expected token nbf 30s in future to pass with 60s skew, got %v", err)
	}

	// 4. Not valid until 65 seconds in future (fixedTime + 65s) -> Invalid (outside 60s skew)
	tokNbf65sFuture := signTestTokenWithTime(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", fixedTime, func(b *jwt.Builder) {
		b.NotBefore(fixedTime.Add(65 * time.Second))
	}, nil)
	if _, err := verifier.Verify(context.Background(), tokNbf65sFuture); err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for token nbf 65s in future, got %v", err)
	}
}

func TestJWKSVerifierUnknownKIDRefreshesOnce(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	var fetchCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&fetchCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	// Token with known kid populates cache (fetchCount = 1)
	knownToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)
	if _, err := verifier.Verify(context.Background(), knownToken); err != nil {
		t.Fatalf("verify known token: %v", err)
	}
	if got := atomic.LoadInt64(&fetchCount); got != 1 {
		t.Fatalf("expected 1 fetch, got %d", got)
	}

	// Token with unknown kid triggers exactly 1 refresh, then fails with ErrUnauthorized
	unknownToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-missing", nil, nil)
	_, err = verifier.Verify(context.Background(), unknownToken)
	if err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if got := atomic.LoadInt64(&fetchCount); got != 2 {
		t.Fatalf("expected 2 fetches total after unknown kid refresh, got %d", got)
	}
}

func TestJWKSVerifierConcurrentUnknownKIDRefreshIsCoordinated(t *testing.T) {
	t.Parallel()

	oldPriv, oldPub := createTestECKey(t, "old-key")
	newPriv, newPub := createTestECKey(t, "new-key")

	oldJWKSBytes := buildJWKSJSON(t, oldPub)
	bothJWKSBytes := buildJWKSJSON(t, oldPub, newPub)

	var fetchCount int64
	refreshStarted := make(chan struct{}, 1)
	releaseRefresh := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&fetchCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			// Priming request
			w.Write(oldJWKSBytes)
			return
		}
		// Refresh request: signal barrier and wait for release
		select {
		case refreshStarted <- struct{}{}:
		default:
		}
		<-releaseRefresh
		w.Write(bothJWKSBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	// 1. Prime the snapshot with old-key
	oldToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), oldPriv, "old-key", nil, nil)
	if _, err := verifier.Verify(context.Background(), oldToken); err != nil {
		t.Fatalf("prime verify failed: %v", err)
	}
	if got := atomic.LoadInt64(&fetchCount); got != 1 {
		t.Fatalf("expected 1 fetch after priming, got %d", got)
	}

	// 2. Concurrently launch multiple callers with new-key (not in fresh cache gen 1)
	newToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), newPriv, "new-key", nil, nil)

	const concurrency = 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errs := make([]error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = verifier.Verify(context.Background(), newToken)
		}(i)
	}

	// Wait for refresh request to enter handler
	select {
	case <-refreshStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for refresh request to reach server")
	}

	// Release refresh handler
	close(releaseRefresh)
	wg.Wait()

	// 3. Assert all concurrent callers succeeded and shared the single refresh
	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d failed: %v", i, err)
		}
	}

	if got := atomic.LoadInt64(&fetchCount); got != 2 {
		t.Errorf("expected exactly 2 total fetches (1 prime + 1 coordinated refresh), got %d", got)
	}
}

func TestJWKSVerifierKnownKeyNotBlockedDuringRefresh(t *testing.T) {
	t.Parallel()

	knownPriv, knownPub := createTestECKey(t, "known-key")
	unknownPriv, unknownPub := createTestECKey(t, "unknown-key")

	knownJWKSBytes := buildJWKSJSON(t, knownPub)
	bothJWKSBytes := buildJWKSJSON(t, knownPub, unknownPub)

	var fetchCount int64
	refreshEntered := make(chan struct{}, 1)
	releaseRefresh := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&fetchCount, 1)
		w.Header().Set("Content-Type", "application/json")
		if count == 1 {
			// Priming request
			w.Write(knownJWKSBytes)
			return
		}
		// Blocked refresh for unknown-key
		select {
		case refreshEntered <- struct{}{}:
		default:
		}
		<-releaseRefresh
		w.Write(bothJWKSBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	// 1. Prime snapshot with known-key
	knownToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), knownPriv, "known-key", nil, nil)
	if _, err := verifier.Verify(context.Background(), knownToken); err != nil {
		t.Fatalf("prime verify: %v", err)
	}

	// 2. In background, start unknown-key verify that triggers blocked refresh
	unknownToken := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), unknownPriv, "unknown-key", nil, nil)
	unknownResultChan := make(chan error, 1)
	go func() {
		_, err := verifier.Verify(context.Background(), unknownToken)
		unknownResultChan <- err
	}()

	// Wait until the unknown-key refresh is actively blocked in the HTTP handler
	select {
	case <-refreshEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for refresh to block in handler")
	}

	// 3. While refresh is STILL blocked, verify knownToken -> must complete immediately
	knownDoneChan := make(chan error, 1)
	go func() {
		_, err := verifier.Verify(context.Background(), knownToken)
		knownDoneChan <- err
	}()

	select {
	case err := <-knownDoneChan:
		if err != nil {
			t.Fatalf("known-key verify failed during in-flight refresh: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("known-key verification was blocked by concurrent in-flight refresh!")
	}

	// 4. Release refresh barrier and clean up
	close(releaseRefresh)
	select {
	case err := <-unknownResultChan:
		if err != nil {
			t.Fatalf("unknown-key verify failed after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for background unknown-key verify")
	}
}

func TestJWKSVerifierUnavailableWithoutFreshKey(t *testing.T) {
	t.Parallel()

	ecPriv, _ := createTestECKey(t, "ec-1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal db error"}`))
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	token := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)

	_, err = verifier.Verify(context.Background(), token)
	if err == nil || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable on 500 status without fresh key, got %v", err)
	}

	// Test WriteError mapping: must return 503 and not leak provider response
	oldOutput := log.Writer()
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer func() {
		if oldOutput != nil {
			log.SetOutput(oldOutput)
		} else {
			log.SetOutput(os.Stderr)
		}
	}()

	rec := httptest.NewRecorder()
	WriteError(rec, err)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected HTTP 503, got %d", rec.Code)
	}

	if strings.Contains(rec.Body.String(), "internal db error") || strings.Contains(rec.Body.String(), token) {
		t.Errorf("response body leaked sensitive details: %s", rec.Body.String())
	}
	if strings.Contains(logBuf.String(), token) {
		t.Errorf("log leaked token: %s", logBuf.String())
	}
}

func TestJWKSVerifierRejectsOversizedOrInvalidJWKS(t *testing.T) {
	t.Parallel()

	ecPriv, _ := createTestECKey(t, "ec-1")

	tests := []struct {
		name       string
		statusCode int
		handler    http.HandlerFunc
	}{
		{
			name: "body exceeds 1 MiB limit",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// Write 1 MiB + 100 bytes
				w.Write([]byte(`{"keys":[`))
				padding := strings.Repeat(`{"kty":"oct","k":"c2VjcmV0"},`, 30000)
				w.Write([]byte(padding))
				w.Write([]byte(`]}`))
			},
		},
		{
			name: "exceeds 32 keys limit",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				set := jwk.NewSet()
				for i := 0; i < 35; i++ {
					_, pub := createTestECKey(t, fmt.Sprintf("k-%d", i))
					_ = set.AddKey(pub)
				}
				buf, _ := json.Marshal(set)
				w.Write(buf)
			},
		},
		{
			name: "malformed JSON body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{not valid json}`))
			},
		},
		{
			name: "no approved asymmetric keys (only symmetric HS256 key)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"keys":[{"kty":"oct","k":"c2VjcmV0LWtleS0zMi1ieXRlcy1sb25nISEh","kid":"sym-1"}]}`))
			},
		},
		{
			name: "rejects redirect",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/somewhere-else", http.StatusFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()

			verifier, err := NewSupabaseJWKSVerifier(server.URL)
			if err != nil {
				t.Fatalf("new verifier: %v", err)
			}

			token := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)
			_, err = verifier.Verify(context.Background(), token)
			if err == nil || !errors.Is(err, ErrUnavailable) {
				t.Fatalf("expected ErrUnavailable, got %v", err)
			}
		})
	}
}

func TestJWKSVerifierDoesNotUseExpiredKeyAfterRefreshFailure(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	var shouldFail int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&shouldFail) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	curTime := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)
	var timeMu sync.Mutex
	clock := func() time.Time {
		timeMu.Lock()
		defer timeMu.Unlock()
		return curTime
	}

	verifier, err := newSupabaseJWKSVerifier(server.URL, nil, clock)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	token := signTestToken(t, server.URL+"/auth/v1", jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
		b.Expiration(curTime.Add(2 * time.Hour))
		b.IssuedAt(curTime)
	}, nil)

	// Step 1: Initial call at 10:00 -> succeeds, cache expires at 10:10
	if _, err := verifier.Verify(context.Background(), token); err != nil {
		t.Fatalf("initial verify: %v", err)
	}

	// Step 2: Advance time past 10 minutes (to 10:15) so cache becomes expired/stale
	timeMu.Lock()
	curTime = curTime.Add(15 * time.Minute)
	timeMu.Unlock()

	// Server starts failing
	atomic.StoreInt32(&shouldFail, 1)

	// Step 3: Verify must attempt refresh, fail, and NOT use the expired cached key -> ErrUnavailable
	_, err = verifier.Verify(context.Background(), token)
	if err == nil || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable when refresh fails with expired cache, got %v", err)
	}
}

type spyStaffLookup struct {
	mu        sync.Mutex
	callCount int
	calls     []string
}

func (s *spyStaffLookup) GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (StaffRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callCount++
	s.calls = append(s.calls, supabaseUserID)
	return StaffRow{
		ID:             "staff-1",
		DisplayName:    "Test Staff",
		Email:          "staff@example.com",
		RoleLabel:      "owner",
		SupabaseUserID: supabaseUserID,
		Status:         "active",
	}, nil
}

func TestJWKSVerifierInvalidTokenDoesNotInvokeStaffResolver(t *testing.T) {
	t.Parallel()

	ecPriv, ecPub := createTestECKey(t, "ec-1")
	jwksBytes := buildJWKSJSON(t, ecPub)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jwksBytes)
	}))
	defer server.Close()

	verifier, err := NewSupabaseJWKSVerifier(server.URL)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	spyLookup := &spyStaffLookup{}
	resolver := NewStaffCapabilityResolver(spyLookup)
	authenticator := NewAuthenticatorWithResolver(verifier, resolver)

	baseIssuer := server.URL + "/auth/v1"

	// 1. Invalid tokens: must yield ErrUnauthorized and spyLookup callCount must remain ZERO
	invalidTokens := []struct {
		name      string
		makeToken func() string
	}{
		{
			name: "wrong issuer",
			makeToken: func() string {
				return signTestToken(t, "https://wrong.supabase.co/auth/v1", jwa.ES256(), ecPriv, "ec-1", nil, nil)
			},
		},
		{
			name: "expired token",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", func(b *jwt.Builder) {
					b.Expiration(time.Now().Add(-2 * time.Hour))
				}, nil)
			},
		},
		{
			name: "missing kid header",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "", nil, nil)
			},
		},
		{
			name: "unknown kid",
			makeToken: func() string {
				return signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "unknown-kid", nil, nil)
			},
		},
		{
			name: "bad signature",
			makeToken: func() string {
				otherPriv, _ := createTestECKey(t, "other-key")
				return signTestToken(t, baseIssuer, jwa.ES256(), otherPriv, "ec-1", nil, nil)
			},
		},
	}

	for _, tc := range invalidTokens {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
			req.Header.Set("Authorization", "Bearer "+tc.makeToken())

			_, err := authenticator.Principal(req)
			if err == nil || !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("expected ErrUnauthorized, got %v", err)
			}
		})
	}

	// Assert resolver call count stayed 0 across all invalid token attempts
	spyLookup.mu.Lock()
	count := spyLookup.callCount
	spyLookup.mu.Unlock()
	if count != 0 {
		t.Fatalf("expected 0 resolver invocations for invalid tokens, got %d", count)
	}

	// 2. Valid token: must succeed and invoke resolver exactly once
	validToken := signTestToken(t, baseIssuer, jwa.ES256(), ecPriv, "ec-1", nil, nil)
	validReq := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	validReq.Header.Set("Authorization", "Bearer "+validToken)

	p, err := authenticator.Principal(validReq)
	if err != nil {
		t.Fatalf("valid token principal: %v", err)
	}
	if p.Role != "owner" || p.StaffID != "staff-1" {
		t.Fatalf("unexpected principal: %+v", p)
	}

	spyLookup.mu.Lock()
	finalCount := spyLookup.callCount
	spyLookup.mu.Unlock()
	if finalCount != 1 {
		t.Fatalf("expected exactly 1 resolver invocation after valid token, got %d", finalCount)
	}
}

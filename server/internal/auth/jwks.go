package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	defaultHTTPTimeout = 5 * time.Second
	defaultCacheTTL    = 10 * time.Minute
	maxJWKSBodySize    = 1 << 20 // 1 MiB
	maxJWKSKeys        = 32
	maxTokenBytes      = 16384 // 16 KiB
	clockSkewAllowance = 60 * time.Second
)

type jwksSnapshot struct {
	expiresAt time.Time
	rawKeys   map[string]crypto.PublicKey
	keyAlgs   map[string]jwa.SignatureAlgorithm
	gen       uint64
}

// JWKSVerifier verifies Supabase access tokens locally against an asymmetric
// JWKS key set fetched from <SUPABASE_URL>/auth/v1/.well-known/jwks.json.
type JWKSVerifier struct {
	baseURL        string
	issuer         string
	jwksURL        string
	client         *http.Client
	now            func() time.Time
	mu             sync.Mutex
	snapshot       *jwksSnapshot
	refreshCh      chan struct{}
	refreshGen     uint64
	lastRefreshErr error
}

// NewSupabaseJWKSVerifier creates a JWKSVerifier for the given Supabase base URL.
func NewSupabaseJWKSVerifier(baseURL string) (*JWKSVerifier, error) {
	return newSupabaseJWKSVerifier(baseURL, nil, nil)
}

func newSupabaseJWKSVerifier(baseURL string, client *http.Client, clock func() time.Time) (*JWKSVerifier, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil, errors.New("supabase base URL is required")
	}
	u, err := url.Parse(trimmed)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return nil, fmt.Errorf("invalid supabase base URL: %w", err)
	}

	issuer := trimmed + "/auth/v1"
	jwksURL := issuer + "/.well-known/jwks.json"

	if client == nil {
		client = &http.Client{
			Timeout: defaultHTTPTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return errors.New("redirects are forbidden")
			},
		}
	}
	if clock == nil {
		clock = time.Now
	}

	return &JWKSVerifier{
		baseURL: trimmed,
		issuer:  issuer,
		jwksURL: jwksURL,
		client:  client,
		now:     clock,
	}, nil
}

// Verify validates an asymmetric Supabase JWT and returns an initial Principal.
func (v *JWKSVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	if len(token) == 0 || len(token) > maxTokenBytes {
		return Principal{}, ErrUnauthorized
	}

	// 1. Inspect JWS protected header.
	msg, err := jws.Parse([]byte(token))
	if err != nil {
		return Principal{}, fmt.Errorf("%w: parse compact token", ErrUnauthorized)
	}
	sigs := msg.Signatures()
	if len(sigs) != 1 {
		return Principal{}, fmt.Errorf("%w: expected exactly one signature", ErrUnauthorized)
	}
	headers := sigs[0].ProtectedHeaders()
	kid, ok := headers.KeyID()
	if !ok || strings.TrimSpace(kid) == "" {
		return Principal{}, fmt.Errorf("%w: missing or empty kid", ErrUnauthorized)
	}
	alg, ok := headers.Algorithm()
	if !ok || (alg != jwa.ES256() && alg != jwa.RS256()) {
		return Principal{}, fmt.Errorf("%w: disallowed algorithm %s", ErrUnauthorized, alg.String())
	}

	// 2. Key resolution with non-blocking cached reads and in-flight refresh coordination.
	rawKey, keyAlg, err := v.resolveKey(ctx, kid)
	if err != nil {
		return Principal{}, err
	}

	if keyAlg != alg {
		return Principal{}, fmt.Errorf("%w: key algorithm %s does not match token algorithm %s", ErrUnauthorized, keyAlg.String(), alg.String())
	}

	// 3. Cryptographic signature verification.
	tok, err := jwt.Parse(
		[]byte(token),
		jwt.WithKey(alg, rawKey),
		jwt.WithValidate(false),
	)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: token signature verification failed", ErrUnauthorized)
	}

	// 4. Claims validation.
	now := v.now()

	// Issuer: exact match
	iss, ok := tok.Issuer()
	if !ok || iss != v.issuer {
		return Principal{}, fmt.Errorf("%w: issuer mismatch", ErrUnauthorized)
	}

	// Audience: must contain "authenticated"
	auds, ok := tok.Audience()
	if !ok || len(auds) == 0 {
		return Principal{}, fmt.Errorf("%w: missing audience", ErrUnauthorized)
	}
	hasAuthAud := false
	for _, a := range auds {
		if a == "authenticated" {
			hasAuthAud = true
			break
		}
	}
	if !hasAuthAud {
		return Principal{}, fmt.Errorf("%w: audience does not contain authenticated", ErrUnauthorized)
	}

	// Expiration: required and unexpired (with 60s skew)
	exp, ok := tok.Expiration()
	if !ok || exp.IsZero() {
		return Principal{}, fmt.Errorf("%w: missing exp claim", ErrUnauthorized)
	}
	if now.After(exp.Add(clockSkewAllowance)) {
		return Principal{}, fmt.Errorf("%w: token expired", ErrUnauthorized)
	}

	// Not Before (nbf): optional, if present must not be in future (with 60s skew)
	nbf, ok := tok.NotBefore()
	if ok && !nbf.IsZero() {
		if nbf.After(now.Add(clockSkewAllowance)) {
			return Principal{}, fmt.Errorf("%w: token not yet valid", ErrUnauthorized)
		}
	}

	// Subject: non-empty string
	sub, ok := tok.Subject()
	if !ok || strings.TrimSpace(sub) == "" {
		return Principal{}, fmt.Errorf("%w: missing or empty sub claim", ErrUnauthorized)
	}

	// Email: string claim, optional
	var email string
	var emailVal any
	if err := tok.Get("email", &emailVal); err == nil {
		if s, ok := emailVal.(string); ok {
			email = s
		}
	}

	// Principal initial role is always "user", with no capabilities or staff ID.
	return Principal{
		UserID: sub,
		Email:  email,
		Role:   "user",
	}, nil
}

func (v *JWKSVerifier) resolveKey(ctx context.Context, kid string) (crypto.PublicKey, jwa.SignatureAlgorithm, error) {
	now := v.now()

	v.mu.Lock()
	snap := v.snapshot
	// 1. Fast path: Fresh snapshot contains matching key. Return immediately without blocking on any in-flight fetch.
	if snap != nil && now.Before(snap.expiresAt) {
		if raw, ok := snap.rawKeys[kid]; ok {
			alg := snap.keyAlgs[kid]
			v.mu.Unlock()
			return raw, alg, nil
		}
	}

	var observedGen uint64
	if snap != nil {
		observedGen = snap.gen
	}

	for {
		// 2. If a refresh is currently in-flight, wait for it to complete.
		if v.refreshCh != nil {
			ch := v.refreshCh
			targetGen := v.refreshGen
			v.mu.Unlock()

			select {
			case <-ch:
			case <-ctx.Done():
				return nil, jwa.EmptySignatureAlgorithm(), ctx.Err()
			}

			v.mu.Lock()
			newSnap := v.snapshot
			if newSnap != nil && newSnap.gen >= targetGen && now.Before(newSnap.expiresAt) {
				if raw, ok := newSnap.rawKeys[kid]; ok {
					alg := newSnap.keyAlgs[kid]
					v.mu.Unlock()
					return raw, alg, nil
				}
				// Refresh succeeded, but kid is not in the updated keyset -> ErrUnauthorized.
				v.mu.Unlock()
				return nil, jwa.EmptySignatureAlgorithm(), fmt.Errorf("%w: kid %q not found in jwks", ErrUnauthorized, kid)
			}

			refreshErr := v.lastRefreshErr
			v.mu.Unlock()
			if refreshErr != nil {
				return nil, jwa.EmptySignatureAlgorithm(), fmt.Errorf("%w: %v", ErrUnavailable, refreshErr)
			}
			return nil, jwa.EmptySignatureAlgorithm(), fmt.Errorf("%w: jwks refresh did not produce valid snapshot", ErrUnavailable)
		}

		// 3. Recheck: Did another goroutine complete a refresh while we waited for the lock?
		if v.snapshot != nil && v.snapshot.gen > observedGen && now.Before(v.snapshot.expiresAt) {
			if raw, ok := v.snapshot.rawKeys[kid]; ok {
				alg := v.snapshot.keyAlgs[kid]
				v.mu.Unlock()
				return raw, alg, nil
			}
			v.mu.Unlock()
			return nil, jwa.EmptySignatureAlgorithm(), fmt.Errorf("%w: kid %q not found in jwks", ErrUnauthorized, kid)
		}

		// 4. We become the leader for the refresh.
		var nextGen uint64 = 1
		if v.snapshot != nil {
			nextGen = v.snapshot.gen + 1
		}
		refreshCh := make(chan struct{})
		v.refreshCh = refreshCh
		v.refreshGen = nextGen
		v.mu.Unlock() // Mutex is released during network I/O so cached reads proceed concurrently

		newSnap, err := v.fetchJWKS(ctx, nextGen)

		v.mu.Lock()
		if err == nil {
			v.snapshot = newSnap
			v.lastRefreshErr = nil
		} else {
			v.lastRefreshErr = err
		}
		v.refreshCh = nil
		close(refreshCh)
		v.mu.Unlock()

		if err != nil {
			return nil, jwa.EmptySignatureAlgorithm(), fmt.Errorf("%w: %v", ErrUnavailable, err)
		}

		if raw, ok := newSnap.rawKeys[kid]; ok {
			return raw, newSnap.keyAlgs[kid], nil
		}
		return nil, jwa.EmptySignatureAlgorithm(), fmt.Errorf("%w: kid %q not found in jwks", ErrUnauthorized, kid)
	}
}

func (v *JWKSVerifier) fetchJWKS(ctx context.Context, gen uint64) (*jwksSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create jwks request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	lr := io.LimitReader(resp.Body, maxJWKSBodySize+1)
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("read jwks body: %w", err)
	}
	if len(body) > maxJWKSBodySize {
		return nil, errors.New("jwks body exceeds maximum size limit")
	}

	set, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse jwk set: %w", err)
	}

	if set.Len() > maxJWKSKeys {
		return nil, errors.New("jwks exceeds maximum key count")
	}

	rawKeys := make(map[string]crypto.PublicKey)
	keyAlgs := make(map[string]jwa.SignatureAlgorithm)

	for i := 0; i < set.Len(); i++ {
		k, ok := set.Key(i)
		if !ok {
			continue
		}
		kid, ok := k.KeyID()
		if !ok || strings.TrimSpace(kid) == "" {
			continue
		}

		if use, ok := k.KeyUsage(); ok && use != "" && use != "sig" {
			continue
		}

		var raw any
		if err := jwk.Export(k, &raw); err != nil {
			continue
		}

		switch pub := raw.(type) {
		case *ecdsa.PublicKey:
			if pub.Curve != elliptic.P256() {
				continue
			}
			if algVal, ok := k.Algorithm(); ok && algVal.String() != "" && algVal.String() != "ES256" {
				continue
			}
			rawKeys[kid] = pub
			keyAlgs[kid] = jwa.ES256()
		case *rsa.PublicKey:
			if pub.N == nil || pub.N.BitLen() < 2048 {
				continue
			}
			if algVal, ok := k.Algorithm(); ok && algVal.String() != "" && algVal.String() != "RS256" {
				continue
			}
			rawKeys[kid] = pub
			keyAlgs[kid] = jwa.RS256()
		default:
			// Disallowed / unsupported key types
			continue
		}
	}

	if len(rawKeys) == 0 {
		return nil, errors.New("jwks contains no approved asymmetric keys")
	}

	return &jwksSnapshot{
		expiresAt: v.now().Add(defaultCacheTTL),
		rawKeys:   rawKeys,
		keyAlgs:   keyAlgs,
		gen:       gen,
	}, nil
}

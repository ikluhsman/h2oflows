package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Verifier validates Supabase-issued JWTs against a JWKS endpoint.
// The JWKS is fetched once on construction and refreshed periodically in
// the background by the jwk cache.
type Verifier struct {
	jwksURL string
	cache   *jwk.Cache
	keyset  jwk.Set
}

// NewVerifier constructs a Verifier and performs an initial JWKS fetch so
// that failures (e.g. typo in the URL) surface at startup rather than on
// the first authenticated request.
func NewVerifier(ctx context.Context, jwksURL string) (*Verifier, error) {
	if jwksURL == "" {
		return nil, errors.New("auth: JWKS URL is empty")
	}

	cache := jwk.NewCache(ctx)
	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("auth: register jwks: %w", err)
	}
	// Fetch once up-front so we fail fast on misconfiguration.
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("auth: initial jwks fetch: %w", err)
	}

	return &Verifier{
		jwksURL: jwksURL,
		cache:   cache,
		keyset:  jwk.NewCachedSet(cache, jwksURL),
	}, nil
}

// Claims is the minimal set of fields we care about from a Supabase JWT.
type Claims struct {
	UserID string // "sub" — Supabase auth.users UUID
	Email  string // "email" — user's primary email, if present
	Role   string // "app_metadata.role" — "admin" or "" for regular users
}

// Verify parses and validates a JWT string, returning the parsed claims.
// Signature, expiry, and not-before are all checked.
func (v *Verifier) Verify(ctx context.Context, tokenString string) (*Claims, error) {
	tok, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithKeySet(v.keyset),
		jwt.WithValidate(true),
		jwt.WithAcceptableSkew(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("auth: parse/verify token: %w", err)
	}

	sub := tok.Subject()
	if sub == "" {
		return nil, errors.New("auth: token missing sub claim")
	}

	// Email is stored as a private claim by Supabase.
	var email string
	if v, ok := tok.Get("email"); ok {
		if s, ok := v.(string); ok {
			email = s
		}
	}

	// Role comes from app_metadata.role (admin-controlled, not user-editable).
	var role string
	if v, ok := tok.Get("app_metadata"); ok {
		if m, ok := v.(map[string]interface{}); ok {
			if r, ok := m["role"].(string); ok {
				role = r
			}
		}
	}

	return &Claims{UserID: sub, Email: email, Role: role}, nil
}

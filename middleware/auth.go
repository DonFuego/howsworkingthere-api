package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"crypto/rsa"
	"encoding/base64"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// JWKS represents the JSON Web Key Set from Auth0.
type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
}

// JSONWebKey represents a single key in the JWKS.
type JSONWebKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
}

var cache = &jwksCache{
	keys: make(map[string]*rsa.PublicKey),
	ttl:  1 * time.Hour,
}

// Auth0Middleware validates the Auth0 JWT Bearer token and injects user_id into the request context.
// Paths that use their own auth (e.g. Auth0 triggers) are skipped.
func Auth0Middleware(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth0 trigger paths use their own HMAC JWT auth
		if r.URL.Path == "/user/register" {
			authHeader := r.Header.Get("Authorization")
			if err := ValidateTriggerToken(authHeader); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
				return
			}
			inner.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, `{"error":"invalid Authorization header format"}`, http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		domain := os.Getenv("AUTH0_DOMAIN")
		audience := os.Getenv("AUTH0_AUDIENCE")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("missing kid in token header")
			}

			return getPublicKey(domain, kid)
		}, jwt.WithIssuer(fmt.Sprintf("https://%s/", domain)),
			jwt.WithAudience(audience),
			jwt.WithExpirationRequired(),
		)

		if err != nil || !token.Valid {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			http.Error(w, `{"error":"missing sub claim"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, sub)
		inner.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getPublicKey retrieves the RSA public key for the given kid from the JWKS endpoint.
func getPublicKey(domain, kid string) (*rsa.PublicKey, error) {
	cache.mu.RLock()
	if key, ok := cache.keys[kid]; ok && time.Since(cache.fetchedAt) < cache.ttl {
		cache.mu.RUnlock()
		return key, nil
	}
	cache.mu.RUnlock()

	jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", domain)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.keys = make(map[string]*rsa.PublicKey)
	cache.fetchedAt = time.Now()

	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || key.Use != "sig" {
			continue
		}

		pubKey, err := parseRSAPublicKey(key)
		if err != nil {
			continue
		}
		cache.keys[key.Kid] = pubKey
	}

	pubKey, ok := cache.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %s not found in JWKS", kid)
	}

	return pubKey, nil
}

func parseRSAPublicKey(key JSONWebKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// GetUserIDFromContext extracts the authenticated user_id from the request context.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

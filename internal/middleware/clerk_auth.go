package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDContextKey contextKey = "user_id"

type ClerkAuth struct {
	publicKeys map[string]*rsa.PublicKey
	mu         sync.RWMutex
	jwksURL    string
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewClerkAuth(jwksURL string) *ClerkAuth {
	if jwksURL == "" {
		jwksURL = "https://clerk.your-domain.com/.well-known/jwks.json"
	}
	ca := &ClerkAuth{
		publicKeys: make(map[string]*rsa.PublicKey),
		jwksURL:    jwksURL,
	}
	go ca.refreshKeys()
	return ca
}

func (ca *ClerkAuth) refreshKeys() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	ca.fetchKeys()
	for range ticker.C {
		ca.fetchKeys()
	}
}

func (ca *ClerkAuth) fetchKeys() error {
	resp, err := http.Get(ca.jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	for _, key := range jwks.Keys {
		if key.Kty != "RSA" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}

		eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}

		var e int
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}

		pubKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}

		ca.publicKeys[key.Kid] = pubKey
	}

	return nil
}

func (ca *ClerkAuth) getPublicKey(kid string) (*rsa.PublicKey, error) {
	ca.mu.RLock()
	key, exists := ca.publicKeys[kid]
	ca.mu.RUnlock()

	if !exists {
		ca.fetchKeys()
		ca.mu.RLock()
		key, exists = ca.publicKeys[kid]
		ca.mu.RUnlock()
		if !exists {
			return nil, fmt.Errorf("public key not found")
		}
	}

	return key, nil
}

func (ca *ClerkAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}

				kid, ok := token.Header["kid"].(string)
				if !ok {
					return nil, fmt.Errorf("kid header not found")
				}

				return ca.getPublicKey(kid)
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, `{"error":"user ID not found in token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

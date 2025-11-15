package middleware

import (
	"context"
	"gateway/internal/services"
	"net/http"
	"strings"
)

const APIKeyContextKey contextKey = "apikey"

func APIKeyAuth(apiKeyService *services.APIKeyService) func(http.Handler) http.Handler {
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

			apiKey, err := apiKeyService.GetByKey(r.Context(), parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), APIKeyContextKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

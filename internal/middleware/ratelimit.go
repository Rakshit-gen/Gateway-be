package middleware

import (
	"fmt"
	"gateway/internal/models"
	"gateway/internal/services"
	"net/http"
)

func RateLimiting(limiter *services.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey, ok := r.Context().Value(APIKeyContextKey).(*models.APIKey)
			if !ok {
				http.Error(w, `{"error":"missing API key in context"}`, http.StatusInternalServerError)
				return
			}

			key := fmt.Sprintf("apikey:%d", apiKey.ID)
			allowed, err := limiter.Allow(r.Context(), key, apiKey.RateLimitRPM)
			if err != nil {
				http.Error(w, `{"error":"rate limit check failed"}`, http.StatusInternalServerError)
				return
			}

			if !allowed {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimitRPM))
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

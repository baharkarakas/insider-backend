package middleware

import (
	"net/http"

	"github.com/baharkarakas/insider-backend/internal/api/httpx"
)

// RequireRole wraps a handler and allows only the given role.
func RequireRole(need string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaims(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", nil)

				return
			}
			if claims.Role != need {
				httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", nil)

				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

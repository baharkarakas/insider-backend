package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/baharkarakas/insider-backend/internal/auth"
	"github.com/baharkarakas/insider-backend/internal/config"
)


type ctxKey string
const ClaimsKey ctxKey = "claims"


func Auth(cfg config.Config) func(http.Handler) http.Handler {
return func(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
h := r.Header.Get("Authorization")
if !strings.HasPrefix(h, "Bearer ") { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
tok := strings.TrimPrefix(h, "Bearer ")
claims, err := auth.Parse(cfg.JWTSecret, tok)
if err != nil { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
ctx := context.WithValue(r.Context(), ClaimsKey, claims)
next.ServeHTTP(w, r.WithContext(ctx))
})
}
}
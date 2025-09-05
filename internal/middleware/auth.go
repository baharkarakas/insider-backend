// internal/middleware/auth.go
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/baharkarakas/insider-backend/internal/auth"
)

type ctxKey string

const (
    ctxUserIDKey ctxKey = "uid"
    ctxRoleKey   ctxKey = "role"
)

func UserID(ctx context.Context) (string, bool) { // <--- İSİM DEĞİŞTİ
    v, ok := ctx.Value(ctxUserIDKey).(string)
    return v, ok
}
func Role(ctx context.Context) (string, bool) { // <--- İSİM DEĞİŞTİ
    v, ok := ctx.Value(ctxRoleKey).(string)
    return v, ok
}

type AuthMiddleware struct {
    TM     *auth.TokenManager
    AppEnv string
}

func NewAuthMiddleware(tm *auth.TokenManager, appEnv string) *AuthMiddleware {
    return &AuthMiddleware{TM: tm, AppEnv: appEnv}
}

type errResp struct{ Error string `json:"error"` }

func (m *AuthMiddleware) writeErr(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    _ = json.NewEncoder(w).Encode(errResp{Error: msg})
}

// DEV: Bearer dev-<uuid> | PROD/DEV: Bearer <JWT(access)>
func (m *AuthMiddleware) Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ah := r.Header.Get("Authorization")
        if ah == "" || !strings.HasPrefix(strings.ToLower(ah), "bearer ") {
            m.writeErr(w, http.StatusUnauthorized, "missing bearer token")
            return
        }
        token := strings.TrimSpace(ah[len("Bearer "):])

        if m.AppEnv == "dev" && strings.HasPrefix(token, "dev-") {
            uid := strings.TrimPrefix(token, "dev-")
            ctx := context.WithValue(r.Context(), ctxUserIDKey, uid)
            ctx = context.WithValue(ctx, ctxRoleKey, "user")
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        claims, isRefresh, err := m.TM.ParseAny(token)
        if err != nil || isRefresh {
            m.writeErr(w, http.StatusUnauthorized, "invalid access token")
            return
        }
        ctx := context.WithValue(r.Context(), ctxUserIDKey, claims.UserID)
        ctx = context.WithValue(ctx, ctxRoleKey, claims.Role)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

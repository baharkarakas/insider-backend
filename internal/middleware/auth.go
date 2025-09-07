package middleware

import (
	"context"

	"log"
	"net/http"
	"os"
	"strings"

	"github.com/baharkarakas/insider-backend/internal/api/httpx"
	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	ctxUserIDKey ctxKey = "uid"
	ctxRoleKey   ctxKey = "role"
)

func UserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserIDKey).(string)
	return v, ok
}

func UserRole(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxRoleKey).(string)
	return v, ok
}

// Back-compat alias
func Role(ctx context.Context) (string, bool) { return UserRole(ctx) }

type AuthMiddleware struct {
	
	AppEnv string
}

func NewAuthMiddleware(_any interface{}, appEnv string) *AuthMiddleware {
	
	return &AuthMiddleware{AppEnv: appEnv}
}



func contextWithUser(ctx context.Context, uid, role string) context.Context {
	ctx = context.WithValue(ctx, ctxUserIDKey, uid)
	ctx = context.WithValue(ctx, ctxRoleKey, role)
	return ctx
}

// Access token claim’leri
type accessClaims struct {
	UID  string `json:"uid"`
	Role string `json:"role"`
	Typ  string `json:"typ"` // "access" bekliyoruz
	jwt.RegisteredClaims
}

func (m *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if hdr == "" || !strings.HasPrefix(hdr, "Bearer ") {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token", nil)
			return
		}
		tokenStr := strings.TrimPrefix(hdr, "Bearer ")

		secret := os.Getenv("JWT_ACCESS_SECRET")
		if secret == "" {
			httpx.WriteError(w, http.StatusInternalServerError, "server_misconfig", "missing JWT_ACCESS_SECRET", nil)
			return
		}
		issuer := os.Getenv("JWT_ISSUER") // boş ise kontrol etmeyiz

		claims := &accessClaims{}
		tok, err := jwt.ParseWithClaims(
			tokenStr,
			claims,
			func(t *jwt.Token) (interface{}, error) {
				
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenUnverifiable
				}
				return []byte(secret), nil
			},
			
		)
		if err != nil || !tok.Valid {
			if strings.EqualFold(m.AppEnv, "dev") {
				log.Printf("AUTH VERIFY ERROR: parse/valid failed: %v", err)
			}
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid access token", nil)
			return
		}

		// tip kontrolü
		if claims.Typ != "" && claims.Typ != "access" {
			if strings.EqualFold(m.AppEnv, "dev") {
				log.Printf("AUTH VERIFY ERROR: unexpected typ=%s", claims.Typ)
			}
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid access token", nil)
			return
		}
		// issuer kontrolü (set edilmişse)
		if issuer != "" && claims.Issuer != "" && claims.Issuer != issuer {
			if strings.EqualFold(m.AppEnv, "dev") {
				log.Printf("AUTH VERIFY ERROR: issuer mismatch: got=%s want=%s", claims.Issuer, issuer)
			}
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid access token", nil)
			return
		}
		// uid zorunlu
		if claims.UID == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid access token", nil)
			return
		}

		ctx := contextWithUser(r.Context(), claims.UID, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

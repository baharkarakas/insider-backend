package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/baharkarakas/insider-backend/internal/api/httpx"
)

type ctxKey string

const claimsKey ctxKey = "claims"

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role,omitempty"`
}

func GetClaims(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}

func UserID(ctx context.Context) (string, bool) {
	c, ok := GetClaims(ctx)
	if !ok || c.UserID == "" {
		return "", false
	}
	return c.UserID, true
}

// Gün4 geçici kural seti:
// 1) Authorization: Bearer dev-<userID>
// 2) X-User-ID header
// 3) ?user_id=... query
// 4) JSON body: user_id veya from_user_id (okursak body'yi geri sararız)
func Auth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1) Bearer dev-<id>
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				token := strings.TrimSpace(auth[len("Bearer "):])
				if strings.HasPrefix(token, "dev-") && len(token) > 4 {
					id := strings.TrimPrefix(token, "dev-")
					claims := Claims{UserID: id, Role: "user"}
					ctx := context.WithValue(r.Context(), claimsKey, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			// 2) X-User-ID
			if id := strings.TrimSpace(r.Header.Get("X-User-ID")); id != "" {
				claims := Claims{UserID: id, Role: "user"}
				ctx := context.WithValue(r.Context(), claimsKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// 3) ?user_id=
			if id := strings.TrimSpace(r.URL.Query().Get("user_id")); id != "" {
				claims := Claims{UserID: id, Role: "user"}
				ctx := context.WithValue(r.Context(), claimsKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// 4) JSON body
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				idFromBody, newReq, _ := extractUserIDFromJSON(r)
				if idFromBody != "" && newReq != nil {
					claims := Claims{UserID: idFromBody, Role: "user"}
					ctx := context.WithValue(r.Context(), claimsKey, claims)
					next.ServeHTTP(w, newReq.WithContext(ctx))
					return
				}
			}
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
		})
	}
}

func extractUserIDFromJSON(r *http.Request) (string, *http.Request, error) {
	if r.Body == nil {
		return "", r, errors.New("no body")
	}
	orig, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var payload map[string]any
	if err := json.Unmarshal(orig, &payload); err != nil {
		r.Body = io.NopCloser(strings.NewReader(string(orig)))
		return "", r, err
	}
	id := ""
	if v, ok := payload["user_id"]; ok {
		if s, ok2 := v.(string); ok2 {
			id = s
		}
	}
	if id == "" {
		if v, ok := payload["from_user_id"]; ok {
			if s, ok2 := v.(string); ok2 {
				id = s
			}
		}
	}
	r.Body = io.NopCloser(strings.NewReader(string(orig)))
	return id, r, nil
}

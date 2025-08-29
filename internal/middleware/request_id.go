package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ridKey string
const requestIDKey ridKey = "rid"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := uuid.NewString()
		ctx := context.WithValue(r.Context(), requestIDKey, rid)
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

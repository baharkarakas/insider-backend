package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// Bu pakette zaten ctxKey var; çakışmayı önlemek için farklı bir tip kullanıyoruz.
type reqIDKeyType struct{}

var requestIDKey reqIDKeyType

func newReqID() string {
	// bağımlılıksız, basit bir ID
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
func RequestIDFrom(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok { return s }
	}
	return ""
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newReqID()
		// SADECE header + context; body'ye yazmıyoruz
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
	
}

package middleware

import (
	"log/slog"
	"net/http"

	"github.com/baharkarakas/insider-backend/internal/api/httpx"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec)
				httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)

			}
		}()
		next.ServeHTTP(w, r)
	})
}

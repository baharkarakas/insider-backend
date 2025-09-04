package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/baharkarakas/insider-backend/internal/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, code: 200}
		start := time.Now()
		next.ServeHTTP(rec, r)
		_ = start // istersek latency histogram ekleriz
		metrics.RequestsTotal.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(rec.code)).Inc()
	})
}

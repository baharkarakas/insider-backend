package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_requests_latency_seconds",
			Help:    "Latency of HTTP requests.",
			Buckets: prometheus.DefBuckets, 
		},
		[]string{"method", "route", "status"},
	)

	metricsOnce sync.Once
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPMetrics ölçer
func HTTPMetrics(next http.Handler) http.Handler {
	metricsOnce.Do(func() {
		prometheus.MustRegister(httpLatency)
		
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		route := routePattern(r)
		httpLatency.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).
			Observe(time.Since(start).Seconds())
	})
}

func routePattern(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if patt := rc.RoutePattern(); patt != "" {
			return patt
		}
	}
	// fallback 
	return r.URL.Path
}

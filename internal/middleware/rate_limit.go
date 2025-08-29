package middleware

import (
	"net/http"
	"sync"
	"time"
)

type limiter struct { mu sync.Mutex; last time.Time; count int }

func RateLimit(rps int) func(http.Handler) http.Handler {
	var l limiter
	window := time.Second
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l.mu.Lock()
			if time.Since(l.last) > window { l.last = time.Now(); l.count = 0 }
			l.count++
			ok := l.count <= rps
			l.mu.Unlock()
			if !ok { http.Error(w, "rate limited", http.StatusTooManyRequests); return }
			next.ServeHTTP(w, r)
		})
	}
}
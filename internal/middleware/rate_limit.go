package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/baharkarakas/insider-backend/internal/api/httpx"
)

type tokenBucket struct {
	mu     sync.Mutex
	tokens int
	last   time.Time
	rate   int 
	burst  int 
}

func RateLimit(rps int) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	tb := &tokenBucket{
		tokens: rps,
		last:   time.Now(),
		rate:   rps,
		burst:  rps,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tb.mu.Lock()
			now := time.Now()
			elapsed := now.Sub(tb.last).Seconds()
			if elapsed > 0 {
				refill := int(elapsed * float64(tb.rate))
				if refill > 0 {
					tb.tokens += refill
					if tb.tokens > tb.burst {
						tb.tokens = tb.burst
					}
					tb.last = now
				}
			}
			allowed := tb.tokens > 0
			if allowed {
				tb.tokens--
			}
			tb.mu.Unlock()

			if !allowed {
				httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

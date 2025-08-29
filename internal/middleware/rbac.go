package middleware

import (
	"net/http"
)

func RBAC(roles ...string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, r := range roles { allowed[r] = struct{}{} }
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In a real app, fetch role from context claims
			role := r.Header.Get("X-Debug-Role") // temporary helper
			if _, ok := allowed[role]; !ok { http.Error(w, "forbidden", http.StatusForbidden); return }
			next.ServeHTTP(w, r)
		})
	}
}
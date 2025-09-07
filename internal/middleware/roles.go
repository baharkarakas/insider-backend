package middleware

import "net/http"

func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            got, ok := Role(r.Context())
            if !ok || got != role {
                w.WriteHeader(http.StatusForbidden)
                _, _ = w.Write([]byte("forbidden"))
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

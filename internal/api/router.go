package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/baharkarakas/insider-backend/internal/config"
	"github.com/baharkarakas/insider-backend/internal/middleware"
	"github.com/baharkarakas/insider-backend/internal/services"
)

type RouterDeps struct {
	Cfg         config.Config
	UserSvc     *services.UserService
	BalanceSvc  *services.BalanceService
	TxnSvc      *services.TransactionService
}

func NewRouter(cfg config.Config, us *services.UserService, bs *services.BalanceService, ts *services.TransactionService) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recover, middleware.RateLimit(100))
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET","POST","PUT","DELETE","OPTIONS"}, AllowedHeaders: []string{"*"}}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		// auth
		r.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			var req struct{ Username, Email, Password string }
			_ = json.NewDecoder(r.Body).Decode(&req)
			u, err := us.Register(req.Username, req.Email, req.Password)
			if err != nil { http.Error(w, err.Error(), 400); return }
			json.NewEncoder(w).Encode(u)
		})

		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var req struct{ Email, Password string }
			_ = json.NewDecoder(r.Body).Decode(&req)
			tok, err := us.Login(req.Email, req.Password)
			if err != nil { http.Error(w, "invalid credentials", 401); return }
			json.NewEncoder(w).Encode(map[string]string{"token": tok})
		})

		// users (list minimal)
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			users, err := us.List(); if err != nil { http.Error(w, err.Error(), 500); return }
			json.NewEncoder(w).Encode(users)
		})

		// balances
		r.Get("/balances/current", func(w http.ResponseWriter, r *http.Request) {
			uid := r.URL.Query().Get("user_id")
			b, err := bs.Current(uid)
			if err != nil { http.Error(w, err.Error(), 400); return }
			json.NewEncoder(w).Encode(b)
		})

		// transactions simple credit
		r.Post("/transactions/credit", func(w http.ResponseWriter, r *http.Request) {
			var req struct{ UserID string; Amount int64 }
			_ = json.NewDecoder(r.Body).Decode(&req)
			tx, err := ts.Credit(req.UserID, req.Amount)
			if err != nil { http.Error(w, err.Error(), 400); return }
			json.NewEncoder(w).Encode(tx)
		})
		// balances üstüne ekle: transactions - debit
r.Post("/transactions/debit", func(w http.ResponseWriter, r *http.Request) {
	var req struct{ UserID string; Amount int64 }
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.UserID == "" || req.Amount <= 0 {
		http.Error(w, "bad request", 400); return
	}
	tx, err := ts.Debit(req.UserID, req.Amount)
	if err != nil { http.Error(w, err.Error(), 400); return }
	// asenkron işlediğimiz için 202 mantıklı; istersen 200 de olur
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(tx)
})

// transactions - transfer
r.Post("/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
	var req struct{ FromUserID, ToUserID string; Amount int64 }
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.FromUserID == "" || req.ToUserID == "" || req.Amount <= 0 {
		http.Error(w, "bad request", 400); return
	}
	tx, err := ts.Transfer(req.FromUserID, req.ToUserID, req.Amount)
	if err != nil { http.Error(w, err.Error(), 400); return }
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(tx)
})

		
	})

	return r
}
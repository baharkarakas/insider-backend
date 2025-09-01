package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/baharkarakas/insider-backend/internal/config"
	"github.com/baharkarakas/insider-backend/internal/middleware"
	"github.com/baharkarakas/insider-backend/internal/services"
)

type RouterDeps struct {
	Cfg        config.Config
	UserSvc    *services.UserService
	BalanceSvc *services.BalanceService
	TxnSvc     *services.TransactionService
}

func NewRouter(cfg config.Config, us *services.UserService, bs *services.BalanceService, ts *services.TransactionService) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recover, middleware.RateLimit(100))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// health & metrics
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		// ---------- auth ----------
		r.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct{ Username, Email, Password string }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest); return
			}
			u, err := us.Register(req.Username, req.Email, req.Password)
			if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
			_ = json.NewEncoder(w).Encode(u)
		})

		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct{ Email, Password string }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest); return
			}
			tok, err := us.Login(req.Email, req.Password)
			if err != nil { http.Error(w, "invalid credentials", http.StatusUnauthorized); return }
			_ = json.NewEncoder(w).Encode(map[string]string{"token": tok})
		})

		// ---------- users ----------
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			users, err := us.List()
			if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
			_ = json.NewEncoder(w).Encode(users)
		})

		// ---------- balances ----------
		r.Get("/balances/current", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			uid := r.URL.Query().Get("user_id")
			b, err := bs.Current(uid)
			if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
			_ = json.NewEncoder(w).Encode(b)
		})

		// ---------- transactions (idempotency-key destekli) ----------
		// credit
		r.Post("/transactions/credit", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			idem := r.Header.Get("Idempotency-Key")
			var req struct{ UserID string; Amount int64 }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest); return
			}
			tx, err := ts.CreditIdem(req.UserID, req.Amount, idem)
			if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(tx)
		})

		// debit
		r.Post("/transactions/debit", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			idem := r.Header.Get("Idempotency-Key")
			var req struct{ UserID string; Amount int64 }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.Amount <= 0 {
				http.Error(w, "bad request", http.StatusBadRequest); return
			}
			tx, err := ts.DebitIdem(req.UserID, req.Amount, idem)
			if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(tx)
		})

		// transfer
		r.Post("/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			idem := r.Header.Get("Idempotency-Key")
			var req struct{ FromUserID, ToUserID string; Amount int64 }
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FromUserID == "" || req.ToUserID == "" || req.Amount <= 0 {
				http.Error(w, "bad request", http.StatusBadRequest); return
			}
			tx, err := ts.TransferIdem(req.FromUserID, req.ToUserID, req.Amount, idem)
			if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(tx)
		})

		// get by id
		r.Get("/transactions/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := chi.URLParam(r, "id")
			tx, err := ts.GetByID(id)
			if err != nil { http.Error(w, "not found", http.StatusNotFound); return }
			_ = json.NewEncoder(w).Encode(tx)
		})

		// list by user  👉 (eksik olan buydu)
		r.Get("/transactions", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			uid := r.URL.Query().Get("user_id")
			if uid == "" { http.Error(w, "user_id required", http.StatusBadRequest); return }

			limit := 50
			offset := 0
			if v := r.URL.Query().Get("limit"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 { limit = n }
			}
			if v := r.URL.Query().Get("offset"); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n >= 0 { offset = n }
			}

			txs, err := ts.ListByUser(uid, limit, offset)
			if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
			_ = json.NewEncoder(w).Encode(txs)
		})
	})

	return r
}

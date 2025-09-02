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
			var req struct {
				Username string
				Email    string
				Password string
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			u, err := us.Register(req.Username, req.Email, req.Password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(u)
		})

		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				Email    string
				Password string
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			tok, err := us.Login(req.Email, req.Password)
			if err != nil {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"token": tok})
		})

		// ---------- users ----------
		r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			users, err := us.List()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(users)
		})

		// ---------- balances ----------
		r.Get("/balances/current", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			uid := r.URL.Query().Get("user_id")
			b, err := bs.Current(uid)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(b)
		})

		// ---------- transactions (Idempotency-Key destekli) ----------
		// credit
		r.Post("/transactions/credit", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			idem := r.Header.Get("Idempotency-Key")
			var req struct {
				UserID string
				Amount int64
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			tx, err := ts.CreditIdem(req.UserID, req.Amount, idem)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(tx)
		})

		// debit
		r.Post("/transactions/debit", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			idem := r.Header.Get("Idempotency-Key")
			var req struct {
				UserID string
				Amount int64
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.Amount <= 0 {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			tx, err := ts.DebitIdem(req.UserID, req.Amount, idem)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(tx)
		})

		// transfer
		r.Post("/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			idem := r.Header.Get("Idempotency-Key")
			var req struct {
				FromUserID string
				ToUserID   string
				Amount     int64
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FromUserID == "" || req.ToUserID == "" || req.Amount <= 0 {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			tx, err := ts.TransferIdem(req.FromUserID, req.ToUserID, req.Amount, idem)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(tx)
		})

		// history alias — SPESİFİK (önce)
		r.Get("/transactions/history", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			uid := r.URL.Query().Get("user_id")
			if uid == "" {
				http.Error(w, "user_id required", http.StatusBadRequest)
				return
			}
			limit := parseInt(r.URL.Query().Get("limit"), 50, 1)
			offset := parseInt(r.URL.Query().Get("offset"), 0, 0)

			txs, err := ts.ListByUser(uid, limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(txs)
		})

		// list by user — (sonra)
		r.Get("/transactions", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			uid := r.URL.Query().Get("user_id")
			if uid == "" {
				http.Error(w, "user_id required", http.StatusBadRequest)
				return
			}
			limit := parseInt(r.URL.Query().Get("limit"), 50, 1)
			offset := parseInt(r.URL.Query().Get("offset"), 0, 0)

			txs, err := ts.ListByUser(uid, limit, offset)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(txs)
		})

		// get by id — EN SONA + UUID regex
		r.Get(`/transactions/{id:[0-9a-fA-F-]{36}}`, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			id := chi.URLParam(r, "id")
			tx, err := ts.GetByID(id)
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(tx)
		})
	})

	return r
}

// parseInt parses s into int; returns def if empty/invalid; clamps to min.
func parseInt(s string, def, min int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if v < min {
		return def
	}
	return v
}

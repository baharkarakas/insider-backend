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

// NewRouter sets up all routes & middlewares.
func NewRouter(cfg config.Config, us *services.UserService, bs *services.BalanceService, ts *services.TransactionService) http.Handler {
	r := chi.NewRouter()

	// Base middlewares
	r.Use(middleware.RequestID, middleware.Recover, middleware.RateLimit(100))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // dilersen cfg’den oku
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Liveness & metrics
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// ---------- PUBLIC: auth ----------
		r.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				Username string `json:"username"`
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
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
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
				return
			}
			tok, err := us.Login(req.Email, req.Password)
			if err != nil {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"token": tok})
		})

		// ---------- PROTECTED ----------
		r.Group(func(pr chi.Router) {
			// Gün4 auth: Authorization: Bearer dev-<userID> vb. -> context’e Claims
			pr.Use(middleware.Auth())

			// ---- users ----
			pr.Get("/users", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				users, err := us.List()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				_ = json.NewEncoder(w).Encode(users)
			})

			// ---- balances ----
			pr.Get("/balances/current", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					http.Error(w, "unauthorized: user_id not provided", http.StatusUnauthorized)
					return
				}
				b, err := bs.Current(uid)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(b)
			})

			pr.Get("/balances/at-time", func(w http.ResponseWriter, r *http.Request) {
				// Placeholder: ihtiyaç olunca doldur.
				http.Error(w, "not implemented", http.StatusNotImplemented)
			})

			// ---- transactions (Idempotency-Key destekli) ----

			// credit
			pr.Post("/transactions/credit", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					http.Error(w, "unauthorized: user_id not provided", http.StatusUnauthorized)
					return
				}
				idem := r.Header.Get("Idempotency-Key")
				var req struct {
					Amount int64 `json:"amount"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
					if err != nil {
						http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
						return
					}
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				tx, err := ts.CreditIdem(uid, req.Amount, idem)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(w).Encode(tx)
			})

			// debit
			pr.Post("/transactions/debit", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					http.Error(w, "unauthorized: user_id not provided", http.StatusUnauthorized)
					return
				}
				idem := r.Header.Get("Idempotency-Key")
				var req struct {
					Amount int64 `json:"amount"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
					if err != nil {
						http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
						return
					}
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				tx, err := ts.DebitIdem(uid, req.Amount, idem)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(w).Encode(tx)
			})

			// transfer (from = context; body: to_user_id, amount)
			pr.Post("/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				from, ok := middleware.UserID(r.Context())
				if !ok || from == "" {
					http.Error(w, "unauthorized: user_id not provided", http.StatusUnauthorized)
					return
				}
				idem := r.Header.Get("Idempotency-Key")
				var req struct {
					ToUserID string `json:"to_user_id"`
					Amount   int64  `json:"amount"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ToUserID == "" || req.Amount <= 0 {
					if err != nil {
						http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
						return
					}
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				tx, err := ts.TransferIdem(from, req.ToUserID, req.Amount, idem)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusAccepted)
				_ = json.NewEncoder(w).Encode(tx)
			})

			// list/history (kimlik context’ten)
			pr.Get("/transactions/history", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					http.Error(w, "unauthorized: user_id not provided", http.StatusUnauthorized)
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

			// get by id
			pr.Get(`/transactions/{id:[0-9a-fA-F-]{36}}`, func(w http.ResponseWriter, r *http.Request) {
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

package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"errors"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	h "github.com/baharkarakas/insider-backend/internal/api/handlers"
	"github.com/baharkarakas/insider-backend/internal/api/httpx"
	"github.com/baharkarakas/insider-backend/internal/api/validate"
	a "github.com/baharkarakas/insider-backend/internal/auth"
	"github.com/baharkarakas/insider-backend/internal/config"
	"github.com/baharkarakas/insider-backend/internal/middleware"
	"github.com/baharkarakas/insider-backend/internal/services"
)

// NewRouter sets up all routes & middlewares.
func NewRouter(cfg config.Config, us *services.UserService, bs *services.BalanceService, ts *services.TransactionService) http.Handler {
	r := chi.NewRouter()

	// -------- Middlewares --------
	r.Use(middleware.RequestID, middleware.Recover, middleware.RateLimit(100), middleware.HTTPMetrics)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // istersen cfg'den yükle
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Liveness & metrics 
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Handle("/metrics", promhttp.Handler())

	//JWT manager 
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	accessTTL, _ := time.ParseDuration(os.Getenv("JWT_ACCESS_TTL"))
	refreshTTL, _ := time.ParseDuration(os.Getenv("JWT_REFRESH_TTL"))
	appEnv := os.Getenv("APP_ENV")

	tm := a.NewTokenManager(accessSecret, refreshSecret, accessTTL, refreshTTL)
ah := h.NewAuthHandler(tm, us) 

	// API v1 
	r.Route("/api/v1", func(r chi.Router) {

		// PUBLIC: Auth 
		r.Post("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var req struct {
				Username string `json:"username"`
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
				return
			}
			var verr validate.Errs
			if e := validate.Required("username", req.Username); e != nil { verr = append(verr, *e) }
			if e := validate.Required("email", req.Email); e != nil { verr = append(verr, *e) }
			if len(verr) > 0 {
				httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid payload", verr)
				return
			}
			u, err := us.Register(req.Username, req.Email, req.Password)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "register_failed", err.Error(), nil)
				return
			}
			httpx.WriteJSON(w, http.StatusCreated, u)
		})
		r.Post("/auth/login", ah.Login)
r.Post("/auth/refresh", ah.Refresh)

		// ----- PROTECTED -----
		r.Group(func(pr chi.Router) {
			amw := middleware.NewAuthMiddleware(tm, appEnv)
			pr.Use(amw.Auth)

			// --- Debug: kimim? ---
			pr.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				uid, _ := middleware.UserID(r.Context())
				httpx.WriteJSON(w, http.StatusOK, map[string]any{"user_id": uid})
			})

			// --- Users (admin only) ---
			pr.With(middleware.RequireRole("admin")).Get("/users", func(w http.ResponseWriter, r *http.Request) {
				users, err := us.List()
				if err != nil {
					httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
					return
				}
				httpx.WriteJSON(w, http.StatusOK, users)
			})

			// --- Balances ---
			pr.Get("/balances/current", func(w http.ResponseWriter, r *http.Request) {
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}
				b, err := bs.Current(uid)
				if err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "balance_failed", err.Error(), nil)
					return
				}
				httpx.WriteJSON(w, http.StatusOK, b)
			})
			// placeholder: ileride implement
			pr.Get("/balances/at-time", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not implemented", http.StatusNotImplemented)
			})

			// --- Transactions (Idempotency-Key destekli) ---

			// credit
			pr.Post("/transactions/credit", func(w http.ResponseWriter, r *http.Request) {
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}
				idem := r.Header.Get("Idempotency-Key")

				var in struct {
					Amount int64 `json:"amount"`
				}
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
					return
				}
				var verr validate.Errs
				if e := validate.MinInt("amount", in.Amount, 1); e != nil { verr = append(verr, *e) }
				if len(verr) > 0 {
					httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid payload", verr)
					return
				}
				tx, err := ts.CreditIdem(uid, in.Amount, idem)
				if err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "credit_failed", err.Error(), nil)
					return
				}
				httpx.WriteJSON(w, http.StatusAccepted, tx)
			})

			// debit
			pr.Post("/transactions/debit", func(w http.ResponseWriter, r *http.Request) {
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}
				var in struct {
					Amount int64 `json:"amount"`
				}
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
					return
				}
				var verr validate.Errs
				    

				if e := validate.MinInt("amount", in.Amount, 1); e != nil { verr = append(verr, *e) }
				if len(verr) > 0 {
					httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid payload", verr)
					return
				}
				// Idempotent versiyonun yoksa Debit kullan
				tx, err := ts.Debit(uid, in.Amount)
				if err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "debit_failed", err.Error(), nil)
					return
				}
				httpx.WriteJSON(w, http.StatusAccepted, tx)
			})

			// transfer (from = context; body: to_user_id, amount)
			pr.Post("/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
				from, ok := middleware.UserID(r.Context())
				if !ok || from == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}
				idem := r.Header.Get("Idempotency-Key")

				var in struct {
					ToUserID string `json:"to_user_id"`
					Amount   int64  `json:"amount"`
				}
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
					return
				}
				
if _, err := uuid.Parse(in.ToUserID); err != nil {
    httpx.WriteError(w, http.StatusBadRequest, "validation_error", "to_user_id must be a valid UUID", nil)
    return
}

				var verr validate.Errs
				if e := validate.Required("to_user_id", in.ToUserID); e != nil { verr = append(verr, *e) }
				if e := validate.MinInt("amount", in.Amount, 1); e != nil { verr = append(verr, *e) }
				if len(verr) > 0 {
					httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid payload", verr)
					return
				}
				if in.ToUserID == from {
					httpx.WriteError(w, http.StatusBadRequest, "validation_error", "cannot transfer to self", nil)
					return
				}
				    tx, err := ts.TransferIdem(from, in.ToUserID, in.Amount, idem)
    if errors.Is(err, services.ErrRecipientNotFound) {
        httpx.WriteError(w, http.StatusNotFound, "recipient_not_found", err.Error(), nil)
        return
    }
    if err != nil {
        httpx.WriteError(w, http.StatusBadRequest, "transfer_failed", err.Error(), nil)
        return
    }

				httpx.WriteJSON(w, http.StatusAccepted, tx)
			})

			// list/history
			pr.Get("/transactions/history", func(w http.ResponseWriter, r *http.Request) {
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}
				limit := parseInt(r.URL.Query().Get("limit"), 50, 1)
				offset := parseInt(r.URL.Query().Get("offset"), 0, 0)

				txs, err := ts.ListByUser(uid, limit, offset)
				if err != nil {
					httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
					return
				}
				httpx.WriteJSON(w, http.StatusOK, txs)
			})

			// /transactions/{id}
			pr.Get(`/transactions/{id:[0-9a-fA-F-]{36}}`, func(w http.ResponseWriter, r *http.Request) {
				id := chi.URLParam(r, "id")
				tx, err := ts.GetByID(id)
				if err != nil {
					httpx.WriteError(w, http.StatusNotFound, "not_found", "transaction not found", nil)
					return
				}
				httpx.WriteJSON(w, http.StatusOK, tx)
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

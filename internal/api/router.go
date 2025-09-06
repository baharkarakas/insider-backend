package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

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

	// Base middlewares
	r.Use(middleware.RequestID, middleware.Recover, middleware.RateLimit(100), middleware.HTTPMetrics)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // istersen cfg'den oku
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

	// JWT parçaları (ENV'den)
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	accessTTL, _ := time.ParseDuration(os.Getenv("JWT_ACCESS_TTL"))
	refreshTTL, _ := time.ParseDuration(os.Getenv("JWT_REFRESH_TTL"))
	appEnv := os.Getenv("APP_ENV")

	tm := a.NewTokenManager(accessSecret, refreshSecret, accessTTL, refreshTTL)
	ah := h.NewAuthHandler(tm)

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
				httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
				return
			}

			var verr validate.Errs
			if e := validate.Required("username", req.Username); e != nil {
				verr = append(verr, *e)
			}
			if e := validate.Required("email", req.Email); e != nil {
				verr = append(verr, *e)
			}
			// (İstersen validate.MinLen("password", ...) ekleyebilirsin)
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

		// ---------- PROTECTED ----------
		r.Group(func(pr chi.Router) {
			amw := middleware.NewAuthMiddleware(tm, appEnv)
			pr.Use(amw.Auth)

			// ---- users (admin only) ----
			pr.With(middleware.RequireRole("admin")).Get("/users", func(w http.ResponseWriter, r *http.Request) {
				users, err := us.List()
				if err != nil {
					httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
					return
				}
				httpx.WriteJSON(w, http.StatusOK, users)
			})

			// ---- balances ----
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

			pr.Get("/balances/at-time", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not implemented", http.StatusNotImplemented)
			})

			// ---- transactions (Idempotency-Key destekli) ----

			// credit
			pr.Post("/transactions/credit", func(w http.ResponseWriter, r *http.Request) {
				uid, ok := middleware.UserID(r.Context())
				if !ok || uid == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}

				idem := r.Header.Get("Idempotency-Key")

				type creditReq struct {
					Amount int64 `json:"amount"`
				}
				var in creditReq
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
					return
				}

				var verr validate.Errs
				if e := validate.MinInt("amount", in.Amount, 1); e != nil {
					verr = append(verr, *e)
				}
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

			// transfer (from = context; body: to_user_id, amount)
			pr.Post("/transactions/transfer", func(w http.ResponseWriter, r *http.Request) {
				from, ok := middleware.UserID(r.Context())
				if !ok || from == "" {
					httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "user_id not provided", nil)
					return
				}

				idem := r.Header.Get("Idempotency-Key")

				type transferReq struct {
					ToUserID string `json:"to_user_id"`
					Amount   int64  `json:"amount"`
				}
				var in transferReq
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					httpx.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
					return
				}

				var verr validate.Errs
				if e := validate.Required("to_user_id", in.ToUserID); e != nil {
					verr = append(verr, *e)
				}
				if e := validate.MinInt("amount", in.Amount, 1); e != nil {
					verr = append(verr, *e)
				}
				if len(verr) > 0 {
					httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid payload", verr)
					return
				}

				// self-transfer yasak
				if in.ToUserID == from {
					httpx.WriteError(w, http.StatusBadRequest, "validation_error", "cannot transfer to self", nil)
					return
				}

				tx, err := ts.TransferIdem(from, in.ToUserID, in.Amount, idem)
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

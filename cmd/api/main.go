package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baharkarakas/insider-backend/internal/api"
	"github.com/baharkarakas/insider-backend/internal/config"
	"github.com/baharkarakas/insider-backend/internal/db"
	"github.com/baharkarakas/insider-backend/internal/logger"
	"github.com/baharkarakas/insider-backend/internal/metrics"
	"github.com/baharkarakas/insider-backend/internal/repository/postgres"
	"github.com/baharkarakas/insider-backend/internal/services"
	"github.com/baharkarakas/insider-backend/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

// global db connection pool
var dbPool *pgxpool.Pool

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var err error
	dbPool, err = db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	

if os.Getenv("APP_MIGRATE") == "true" {
	if err := db.RunMigrations(ctx, dbPool); err != nil {
		log.Error("migrations", "err", err)
		os.Exit(1)
	}
}


	// cmd/api/main.go 
repos := postgres.NewRepositories(dbPool)
wp := worker.NewPool(4)
defer wp.Stop()

userSvc := services.NewUserService(repos.Users, cfg)
balanceSvc := services.NewBalanceService(repos.Balances)
txnSvc := services.NewTransactionService(
    repos.Transactions,
    repos.Balances,
    repos.AuditLogs,
    repos.Users,   
    wp,
)



	metrics.Init()
	r := api.NewRouter(cfg, userSvc, balanceSvc, txnSvc)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
log.Info("env check",
  "APP_ENV", os.Getenv("APP_ENV"),
  "JWT_ISSUER", os.Getenv("JWT_ISSUER"),
  "ACCESS_SECRET_len", len(os.Getenv("JWT_ACCESS_SECRET")),
)

	go func() {
		log.Info("server starting", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server", "err", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("env check",
    "APP_ENV", os.Getenv("APP_ENV"),
    "JWT_ISSUER", os.Getenv("JWT_ISSUER"),
    "ACCESS_SECRET_len", len(os.Getenv("JWT_ACCESS_SECRET")),
)


}

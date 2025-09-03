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

	// Opsiyonel: Uygulama içi migration'ı env ile aç/kapat.
// Docker Compose'ta APP_MIGRATE set etmeyeceğiz, yani default: kapalı.
if os.Getenv("APP_MIGRATE") == "true" {
	if err := db.RunMigrations(ctx, dbPool); err != nil {
		log.Error("migrations", "err", err)
		os.Exit(1)
	}
}


	repos := postgres.NewRepositories(dbPool)
	wp := worker.NewPool(4)
	defer wp.Stop()

	userSvc := services.NewUserService(repos.Users, cfg)
	balanceSvc := services.NewBalanceService(repos.Balances)
	txnSvc := services.NewTransactionService(repos.Transactions, repos.Balances, repos.AuditLogs, wp)

	metrics.Init()
	r := api.NewRouter(cfg, userSvc, balanceSvc, txnSvc)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

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
}

package repository

import (
	"context"

	"github.com/baharkarakas/insider-backend/internal/models"
	"github.com/jackc/pgx/v5"
)

type Users interface {
	Create(username, email, passwordHash, role string) (models.User, error)
	GetByID(id string) (models.User, error)
	GetByEmail(email string) (models.User, error)
	List() ([]models.User, error)
	Update(u models.User) error
	Delete(id string) error
}

type Balances interface {
	GetOrCreate(userID string) (models.Balance, error)
	UpdateAmount(userID string, delta int64) (models.Balance, error)
	Get(userID string) (models.Balance, error)
}

type Transactions interface {
	Create(tx models.Transaction) (models.Transaction, error)
	GetByID(id string) (models.Transaction, error)
	ListByUser(userID string, limit, offset int) ([]models.Transaction, error)
	UpdateStatus(id string, status models.TransactionStatus) error

	// Atomik iş bloğu: tek DB transaction'ı içinde çalıştır (pgx.Tx).
	WithTx(ctx context.Context, fn func(pgx.Tx) error) error
}

type AuditLogs interface {
	Create(l models.AuditLog) error
}

package postgres

import (
	repo "github.com/baharkarakas/insider-backend/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repositories struct {
	Users       repo.Users
	Balances    repo.Balances
	Transactions repo.Transactions
	AuditLogs   repo.AuditLogs
}

func NewRepositories(pool *pgxpool.Pool) Repositories {
	return Repositories{
		Users:        &usersRepo{pool},
		Balances:     &balancesRepo{pool},
		Transactions: &transactionsRepo{pool},
		AuditLogs:    &auditLogsRepo{pool},
	}
}
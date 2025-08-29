package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/yourname/insider-backend/internal/repository"
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
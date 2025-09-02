package postgres

import (
	"context"

	"github.com/baharkarakas/insider-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionsRepo struct{ pool *pgxpool.Pool }

// ctor'un vardır; yoksa:
// func NewTransactionsRepo(pool *pgxpool.Pool) *transactionsRepo { return &transactionsRepo{pool: pool} }

func (r *transactionsRepo) Create(tx models.Transaction) (models.Transaction, error) {
	if tx.ID == "" {
		tx.ID = uuid.NewString()
	}
	const q = `
INSERT INTO transactions (
  id, from_user_id, to_user_id, amount, type, status, idempotency_key
) VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (idempotency_key) DO UPDATE
SET idempotency_key = EXCLUDED.idempotency_key  -- no-op update; mevcut satırı RETURNING ile alacağız
RETURNING id, from_user_id, to_user_id, amount, type, status, created_at;
`
	err := r.pool.QueryRow(
		context.Background(), q,
		tx.ID, tx.FromUserID, tx.ToUserID, tx.Amount, tx.Type, tx.Status, tx.IdempotencyKey,
	).Scan(&tx.ID, &tx.FromUserID, &tx.ToUserID, &tx.Amount, &tx.Type, &tx.Status, &tx.CreatedAt)
	return tx, err
}



func (r *transactionsRepo) GetByID(id string) (models.Transaction, error) {
	var tx models.Transaction
	err := r.pool.QueryRow(
		context.Background(),
		`SELECT id, from_user_id, to_user_id, amount, type, status, created_at
		   FROM transactions
		  WHERE id=$1`,
		id,
	).Scan(&tx.ID, &tx.FromUserID, &tx.ToUserID, &tx.Amount, &tx.Type, &tx.Status, &tx.CreatedAt)
	return tx, err
}

func (r *transactionsRepo) ListByUser(userID string, limit, offset int) ([]models.Transaction, error) {
	rows, err := r.pool.Query(
		context.Background(),
		`SELECT id, from_user_id, to_user_id, amount, type, status, created_at
		   FROM transactions
		  WHERE from_user_id=$1 OR to_user_id=$1
		  ORDER BY created_at DESC
		  LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(&tx.ID, &tx.FromUserID, &tx.ToUserID, &tx.Amount, &tx.Type, &tx.Status, &tx.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, tx)
	}
	return out, rows.Err()
}

func (r *transactionsRepo) UpdateStatus(id string, status models.TransactionStatus) error {
	_, err := r.pool.Exec(
		context.Background(),
		`UPDATE transactions SET status=$2 WHERE id=$1`,
		id, status,
	)
	return err
}

// 🔐 pgx ile tek transaction çalıştır
func (r *transactionsRepo) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

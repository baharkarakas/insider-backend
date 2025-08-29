package postgres

import (
	"context"

	"github.com/baharkarakas/insider-backend/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type balancesRepo struct{ pool *pgxpool.Pool }

func (r *balancesRepo) GetOrCreate(userID string) (models.Balance, error) {
	if b, err := r.Get(userID); err == nil {
		return b, nil
	}
	_, err := r.pool.Exec(
		context.Background(),
		`INSERT INTO balances(user_id, amount, last_updated_at)
		 VALUES($1, 0, now())
		 ON CONFLICT (user_id) DO NOTHING`,
		userID,
	)
	if err != nil {
		return models.Balance{}, err
	}
	return r.Get(userID)
}

func (r *balancesRepo) UpdateAmount(userID string, delta int64) (models.Balance, error) {
	var b models.Balance
	err := r.pool.QueryRow(
		context.Background(),
		`UPDATE balances
		    SET amount = amount + $2,
		        last_updated_at = now()
		  WHERE user_id = $1
		  RETURNING user_id, amount, last_updated_at`,
		userID, delta,
	).Scan(&b.UserID, &b.Amount, &b.LastUpdatedAt)
	return b, err
}

func (r *balancesRepo) Get(userID string) (models.Balance, error) {
	var b models.Balance
	err := r.pool.QueryRow(
		context.Background(),
		`SELECT user_id, amount, last_updated_at
		   FROM balances
		  WHERE user_id=$1`,
		userID,
	).Scan(&b.UserID, &b.Amount, &b.LastUpdatedAt)
	return b, err
}

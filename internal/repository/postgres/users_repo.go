// internal/repository/postgres/users_repo.go
package postgres

import (
	"context"

	"github.com/baharkarakas/insider-backend/internal/models"
	"github.com/baharkarakas/insider-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type usersRepo struct{ pool *pgxpool.Pool }


func NewUsers(pool *pgxpool.Pool) repository.Users {
	return &usersRepo{pool: pool}
}

func (r *usersRepo) Create(username, email, hash, role string) (models.User, error) {
	id := uuid.NewString()
	_, err := r.pool.Exec(context.Background(),
		`INSERT INTO users(id, username, email, password_hash, role) VALUES($1,$2,$3,$4,$5)`,
		id, username, email, hash, role,
	)
	if err != nil {
		return models.User{}, err
	}
	return r.GetByID(id)
}

func (r *usersRepo) GetByID(id string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(context.Background(),
		`SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (r *usersRepo) GetByEmail(email string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(context.Background(),
		`SELECT id, username, email, password_hash, role, created_at, updated_at FROM users WHERE email=$1`, email,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

func (r *usersRepo) List() ([]models.User, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, username, email, password_hash, role, created_at, updated_at
         FROM users ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *usersRepo) Update(u models.User) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE users SET username=$2, email=$3, role=$4, updated_at=now() WHERE id=$1`,
		u.ID, u.Username, u.Email, u.Role,
	)
	return err
}

func (r *usersRepo) Delete(id string) error {
	_, err := r.pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (r *usersRepo) Exists(ctx context.Context, id string) (bool, error) {
    var exists bool
    err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, id).Scan(&exists)
    return exists, err
}

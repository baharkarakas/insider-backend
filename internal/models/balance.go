package models

import "time"

// DB satırını temsil eden sade DTO.
// Concurrency'yi DB (Postgres) hallediyor; burada mutex'e gerek yok.
type Balance struct {
	UserID        string    `db:"user_id" json:"user_id"`
	Amount        int64     `db:"amount" json:"amount"`
	LastUpdatedAt time.Time `db:"last_updated_at" json:"last_updated_at"`
}

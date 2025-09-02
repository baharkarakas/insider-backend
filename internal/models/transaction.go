package models

import "time"

// ---- Types ----
type TransactionType string
type TransactionStatus string

// ---- Type enums ----
const (
	// types
	TxnCredit   TransactionType = "credit"
	TxnDebit    TransactionType = "debit"
	TxnTransfer TransactionType = "transfer"

	// statuses
	TxnPending    TransactionStatus = "pending"
	TxnCompleted  TransactionStatus = "completed"
	TxnFailed     TransactionStatus = "failed"
	TxnRolledBack TransactionStatus = "rolled_back"
)

// ---- Model ----
type Transaction struct {
    ID         string             `json:"id"`
    FromUserID *string            `json:"from_user_id,omitempty"`
    ToUserID   *string            `json:"to_user_id,omitempty"`
    Amount     int64              `json:"amount"`
    Type       TransactionType    `json:"type"`
    Status     TransactionStatus  `json:"status"`
    CreatedAt  time.Time          `json:"created_at"`

    // ✅ yeni
    IdempotencyKey *string        `json:"idempotency_key,omitempty"`
}
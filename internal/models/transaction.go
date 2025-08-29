package models

import "time"

type TransactionType string
const (
	TxnCredit   TransactionType = "credit"
	TxnDebit    TransactionType = "debit"
	TxnTransfer TransactionType = "transfer"
)

type TransactionStatus string
const (
	TxnPending    TransactionStatus = "pending"
	TxnCompleted  TransactionStatus = "completed"
	TxnFailed     TransactionStatus = "failed"
	TxnRolledBack TransactionStatus = "rolled_back"
)

type Transaction struct {
	ID         string            `json:"id"`
	FromUserID *string           `json:"from_user_id,omitempty"`
	ToUserID   *string           `json:"to_user_id,omitempty"`
	Amount     int64             `json:"amount"`
	Type       TransactionType   `json:"type"`
	Status     TransactionStatus `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
}
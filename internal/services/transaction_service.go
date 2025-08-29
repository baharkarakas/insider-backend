package services

import (
	"errors"

	"github.com/baharkarakas/insider-backend/internal/models"
	repo "github.com/baharkarakas/insider-backend/internal/repository"
	"github.com/baharkarakas/insider-backend/internal/worker"
)

type TransactionService struct {
	trx repo.Transactions
	bal repo.Balances
	log repo.AuditLogs
	wp  *worker.Pool
}

func NewTransactionService(t repo.Transactions, b repo.Balances, l repo.AuditLogs, wp *worker.Pool) *TransactionService {
	return &TransactionService{trx: t, bal: b, log: l, wp: wp}
}

func (s *TransactionService) Credit(userID string, amount int64) (models.Transaction, error) {
	if amount <= 0 { return models.Transaction{}, errors.New("amount must be > 0") }
	tx := models.Transaction{Amount: amount, Type: models.TxnCredit, Status: models.TxnPending, ToUserID: &userID}
	tx, err := s.trx.Create(tx); if err != nil { return models.Transaction{}, err }
	s.wp.Submit(func() { _ = s.processCredit(tx) })
	return tx, nil
}

func (s *TransactionService) processCredit(tx models.Transaction) error {
	if tx.ToUserID == nil { return errors.New("missing to user") }
	_, err := s.bal.UpdateAmount(*tx.ToUserID, tx.Amount)
	if err != nil { _ = s.trx.UpdateStatus(tx.ID, models.TxnFailed); return err }
	return s.trx.UpdateStatus(tx.ID, models.TxnCompleted)
}
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

// ------ VAR OLAN CREDIT ------
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

// ------ YENİ: DEBIT ------
func (s *TransactionService) Debit(userID string, amount int64) (models.Transaction, error) {
	if amount <= 0 { return models.Transaction{}, errors.New("amount must be > 0") }
	tx := models.Transaction{
		Amount:     amount,
		Type:       models.TxnDebit,
		Status:     models.TxnPending,
		FromUserID: &userID,
	}
	tx, err := s.trx.Create(tx)
	if err != nil { return models.Transaction{}, err }
	s.wp.Submit(func() { _ = s.processDebit(tx) })
	return tx, nil
}

func (s *TransactionService) processDebit(tx models.Transaction) error {
	if tx.FromUserID == nil { return s.trx.UpdateStatus(tx.ID, models.TxnFailed) }
	// Bakiyeden düşmeyi dene (eğer repo yetersiz bakiye için hata döndürüyorsa buradan yakalanır)
	_, err := s.bal.UpdateAmount(*tx.FromUserID, -tx.Amount)
	if err != nil {
		_ = s.trx.UpdateStatus(tx.ID, models.TxnFailed)
		return err
	}
	return s.trx.UpdateStatus(tx.ID, models.TxnCompleted)
}

// ------ YENİ: TRANSFER ------
func (s *TransactionService) Transfer(fromID, toID string, amount int64) (models.Transaction, error) {
	if amount <= 0 { return models.Transaction{}, errors.New("amount must be > 0") }
	if fromID == toID { return models.Transaction{}, errors.New("cannot transfer to self") }

	tx := models.Transaction{
		Amount:     amount,
		Type:       models.TxnTransfer,
		Status:     models.TxnPending,
		FromUserID: &fromID,
		ToUserID:   &toID,
	}
	tx, err := s.trx.Create(tx)
	if err != nil { return models.Transaction{}, err }
	s.wp.Submit(func() { _ = s.processTransfer(tx) })
	return tx, nil
}

func (s *TransactionService) processTransfer(tx models.Transaction) error {
	if tx.FromUserID == nil || tx.ToUserID == nil {
		return s.trx.UpdateStatus(tx.ID, models.TxnFailed)
	}

	// 1) debit
	if _, err := s.bal.UpdateAmount(*tx.FromUserID, -tx.Amount); err != nil {
		_ = s.trx.UpdateStatus(tx.ID, models.TxnFailed)
		return err
	}

	// 2) credit
	if _, err := s.bal.UpdateAmount(*tx.ToUserID, tx.Amount); err != nil {
		// basit kompanzasyon (rollback)
		_, _ = s.bal.UpdateAmount(*tx.FromUserID, tx.Amount)
		_ = s.trx.UpdateStatus(tx.ID, models.TxnRolledBack)
		return err
	}

	return s.trx.UpdateStatus(tx.ID, models.TxnCompleted)
}

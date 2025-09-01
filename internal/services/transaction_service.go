package services

import (
	"errors"
	"fmt"
	"sync"

	"github.com/baharkarakas/insider-backend/internal/models"
	repo "github.com/baharkarakas/insider-backend/internal/repository"
	"github.com/baharkarakas/insider-backend/internal/worker"
)

type TransactionService struct {
	trx  repo.Transactions
	bal  repo.Balances
	log  repo.AuditLogs
	wp   *worker.Pool
	idem sync.Map // Idempotency-Key -> txID (process-local demo)
}

func NewTransactionService(t repo.Transactions, b repo.Balances, l repo.AuditLogs, wp *worker.Pool) *TransactionService {
	return &TransactionService{trx: t, bal: b, log: l, wp: wp}
}

// ----------------- Helpers -----------------

func (s *TransactionService) audit(entityID, action, details string) {
	var det map[string]any
	if details != "" {
		det = map[string]any{"message": details}
	}
	_ = s.log.Create(models.AuditLog{
		EntityType: "transaction",
		EntityID:   &entityID, // pointer veriyoruz
		Action:     action,
		Details:    det,       // map[string]any veriyoruz
	})
}

func (s *TransactionService) updateStatus(txID string, status models.TransactionStatus, reason string) error {
	if err := s.trx.UpdateStatus(txID, status); err != nil {
		return err
	}
	s.audit(txID, "status_change", fmt.Sprintf("%s: %s", status, reason))
	return nil
}

func (s *TransactionService) getOrCreateBalance(userID string) error {
	_, err := s.bal.GetOrCreate(userID)
	return err
}

// ----------------- CREDIT -----------------

// Orijinal API (idempotency olmadan)
func (s *TransactionService) Credit(userID string, amount int64) (models.Transaction, error) {
	return s.CreditIdem(userID, amount, "")
}

// Idempotency destekli sürüm
func (s *TransactionService) CreditIdem(userID string, amount int64, idemKey string) (models.Transaction, error) {
	if amount <= 0 {
		return models.Transaction{}, errors.New("amount must be > 0")
	}

	// Idempotency (process-local demo)
	if idemKey != "" {
		if v, ok := s.idem.Load(idemKey); ok {
			return s.trx.GetByID(v.(string))
		}
	}

	tx := models.Transaction{
		Amount:   amount,
		Type:     models.TxnCredit,
		Status:   models.TxnPending,
		ToUserID: &userID,
	}
	tx, err := s.trx.Create(tx)
	if err != nil {
		return models.Transaction{}, err
	}
	s.audit(tx.ID, "created", "credit created")
	if idemKey != "" {
		s.idem.Store(idemKey, tx.ID)
	}

	s.wp.Submit(func() { _ = s.processCredit(tx) })
	return tx, nil
}

func (s *TransactionService) processCredit(tx models.Transaction) error {
	if tx.ToUserID == nil {
		return s.updateStatus(tx.ID, models.TxnFailed, "missing to user")
	}
	if err := s.getOrCreateBalance(*tx.ToUserID); err != nil {
		_ = s.updateStatus(tx.ID, models.TxnFailed, "getOrCreate balance failed")
		return err
	}
	if _, err := s.bal.UpdateAmount(*tx.ToUserID, tx.Amount); err != nil {
		_ = s.updateStatus(tx.ID, models.TxnFailed, "credit update failed")
		return err
	}
	return s.updateStatus(tx.ID, models.TxnCompleted, "credit applied")
}

// ----------------- DEBIT -----------------

// Orijinal API
func (s *TransactionService) Debit(userID string, amount int64) (models.Transaction, error) {
	return s.DebitIdem(userID, amount, "")
}

// Idempotency destekli sürüm
func (s *TransactionService) DebitIdem(userID string, amount int64, idemKey string) (models.Transaction, error) {
	if amount <= 0 {
		return models.Transaction{}, errors.New("amount must be > 0")
	}

	// Idempotency
	if idemKey != "" {
		if v, ok := s.idem.Load(idemKey); ok {
			return s.trx.GetByID(v.(string))
		}
	}

	// Yetersiz bakiye koruması (servis katı)
	if err := s.getOrCreateBalance(userID); err != nil {
		return models.Transaction{}, err
	}
	if b, err := s.bal.Get(userID); err == nil {
		if b.Amount < amount {
			return models.Transaction{}, errors.New("insufficient balance")
		}
	}

	tx := models.Transaction{
		Amount:     amount,
		Type:       models.TxnDebit,
		Status:     models.TxnPending,
		FromUserID: &userID,
	}
	tx, err := s.trx.Create(tx)
	if err != nil {
		return models.Transaction{}, err
	}
	s.audit(tx.ID, "created", "debit created")
	if idemKey != "" {
		s.idem.Store(idemKey, tx.ID)
	}

	s.wp.Submit(func() { _ = s.processDebit(tx) })
	return tx, nil
}

func (s *TransactionService) processDebit(tx models.Transaction) error {
	if tx.FromUserID == nil {
		return s.updateStatus(tx.ID, models.TxnFailed, "missing from user")
	}
	// (İşlem anında da kontrol etmek istersen buraya tekrar Get koyabilirsin)
	if _, err := s.bal.UpdateAmount(*tx.FromUserID, -tx.Amount); err != nil {
		_ = s.updateStatus(tx.ID, models.TxnFailed, "debit update failed")
		return err
	}
	return s.updateStatus(tx.ID, models.TxnCompleted, "debit applied")
}

// ----------------- TRANSFER -----------------

// Orijinal API
func (s *TransactionService) Transfer(fromID, toID string, amount int64) (models.Transaction, error) {
	return s.TransferIdem(fromID, toID, amount, "")
}

// Idempotency destekli sürüm
func (s *TransactionService) TransferIdem(fromID, toID string, amount int64, idemKey string) (models.Transaction, error) {
	if amount <= 0 {
		return models.Transaction{}, errors.New("amount must be > 0")
	}
	if fromID == toID {
		return models.Transaction{}, errors.New("cannot transfer to self")
	}

	// Idempotency
	if idemKey != "" {
		if v, ok := s.idem.Load(idemKey); ok {
			return s.trx.GetByID(v.(string))
		}
	}

	// Yetersiz bakiye koruması
	if err := s.getOrCreateBalance(fromID); err != nil {
		return models.Transaction{}, err
	}
	if err := s.getOrCreateBalance(toID); err != nil {
		return models.Transaction{}, err
	}
	if b, err := s.bal.Get(fromID); err == nil {
		if b.Amount < amount {
			return models.Transaction{}, errors.New("insufficient balance")
		}
	}

	tx := models.Transaction{
		Amount:     amount,
		Type:       models.TxnTransfer,
		Status:     models.TxnPending,
		FromUserID: &fromID,
		ToUserID:   &toID,
	}
	tx, err := s.trx.Create(tx)
	if err != nil {
		return models.Transaction{}, err
	}
	s.audit(tx.ID, "created", "transfer created")
	if idemKey != "" {
		s.idem.Store(idemKey, tx.ID)
	}

	s.wp.Submit(func() { _ = s.processTransfer(tx) })
	return tx, nil
}

func (s *TransactionService) processTransfer(tx models.Transaction) error {
	if tx.FromUserID == nil || tx.ToUserID == nil {
		return s.updateStatus(tx.ID, models.TxnFailed, "missing ends")
	}

	// 1) debit
	if _, err := s.bal.UpdateAmount(*tx.FromUserID, -tx.Amount); err != nil {
		_ = s.updateStatus(tx.ID, models.TxnFailed, "transfer debit failed")
		return err
	}

	// 2) credit
	if _, err := s.bal.UpdateAmount(*tx.ToUserID, tx.Amount); err != nil {
		// rollback
		_, _ = s.bal.UpdateAmount(*tx.FromUserID, tx.Amount)
		_ = s.updateStatus(tx.ID, models.TxnRolledBack, "transfer credit failed, rolled back")
		return err
	}

	return s.updateStatus(tx.ID, models.TxnCompleted, "transfer applied")
}

// ----------------- Queries -----------------

func (s *TransactionService) GetByID(id string) (models.Transaction, error) {
	return s.trx.GetByID(id)
}

func (s *TransactionService) ListByUser(userID string, limit, offset int) ([]models.Transaction, error) {
	return s.trx.ListByUser(userID, limit, offset)
}

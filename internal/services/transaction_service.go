package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/baharkarakas/insider-backend/internal/metrics"
	"github.com/baharkarakas/insider-backend/internal/models"
	repo "github.com/baharkarakas/insider-backend/internal/repository"
	"github.com/baharkarakas/insider-backend/internal/worker"
	"github.com/jackc/pgx/v5"
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
		EntityID:   &entityID,
		Action:     action,
		Details:    det,
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
	if idemKey != "" {
		tx.IdempotencyKey = &idemKey
	}

	created, err := s.trx.Create(tx)
	if err != nil {
		return models.Transaction{}, err
	}
	s.audit(created.ID, "created", "credit created")
	if idemKey != "" {
		s.idem.Store(idemKey, created.ID)
	}

	s.wp.Submit(func() {
		metrics.WorkerQueueDepth.Inc()
		defer metrics.WorkerQueueDepth.Dec()

		if err := s.processCredit(created); err != nil {
			metrics.TransactionsFailed.Inc()
			return
		}
		metrics.TransactionsTotal.WithLabelValues("credit").Inc()
	})

	return created, nil
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
	if b, err := s.bal.Get(userID); err == nil && b.Amount < amount {
		return models.Transaction{}, errors.New("insufficient balance")
	}

	tx := models.Transaction{
		Amount:     amount,
		Type:       models.TxnDebit,
		Status:     models.TxnPending,
		FromUserID: &userID,
	}
	if idemKey != "" {
		tx.IdempotencyKey = &idemKey
	}

	created, err := s.trx.Create(tx)
	if err != nil {
		return models.Transaction{}, err
	}
	s.audit(created.ID, "created", "debit created")
	if idemKey != "" {
		s.idem.Store(idemKey, created.ID)
	}

	s.wp.Submit(func() {
		metrics.WorkerQueueDepth.Inc()
		defer metrics.WorkerQueueDepth.Dec()

		if err := s.processDebit(created); err != nil {
			metrics.TransactionsFailed.Inc()
			return
		}
		metrics.TransactionsTotal.WithLabelValues("debit").Inc()
	})

	return created, nil
}

func (s *TransactionService) processDebit(tx models.Transaction) error {
	if tx.FromUserID == nil {
		return s.updateStatus(tx.ID, models.TxnFailed, "missing from user")
	}
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

	// Idempotency (process-local)
	if idemKey != "" {
		if v, ok := s.idem.Load(idemKey); ok {
			return s.trx.GetByID(v.(string))
		}
	}

	// Balans kayıtlarını hazırla
	if err := s.getOrCreateBalance(fromID); err != nil {
		return models.Transaction{}, err
	}
	if err := s.getOrCreateBalance(toID); err != nil {
		return models.Transaction{}, err
	}
	// Basit ön kontrol (opsiyonel)
	if b, err := s.bal.Get(fromID); err == nil && b.Amount < amount {
		return models.Transaction{}, errors.New("insufficient balance")
	}

	// Pending transaction kaydı
	txModel := models.Transaction{
		Amount:     amount,
		Type:       models.TxnTransfer,
		Status:     models.TxnPending,
		FromUserID: &fromID,
		ToUserID:   &toID,
	}
	if idemKey != "" {
		txModel.IdempotencyKey = &idemKey
	}
	created, err := s.trx.Create(txModel)
	if err != nil {
		return models.Transaction{}, err
	}
	s.audit(created.ID, "created", "transfer created")
	if idemKey != "" {
		s.idem.Store(idemKey, created.ID)
	}

	// 🔐 Atomik blok (pgx.Tx ile)
	err = s.trx.WithTx(context.Background(), func(pgtx pgx.Tx) error {
		// 1) debit (koşullu: amount >= ?)
		tag, err := pgtx.Exec(context.Background(),
			`UPDATE balances
             SET amount = amount - $1, last_updated_at = now()
             WHERE user_id = $2 AND amount >= $1`,
			amount, fromID,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("insufficient balance")
		}

		// 2) credit
		if _, err := pgtx.Exec(context.Background(),
			`UPDATE balances
             SET amount = amount + $1, last_updated_at = now()
             WHERE user_id = $2`,
			amount, toID,
		); err != nil {
			return err
		}

		// 3) status -> completed (aynı transaction içinde)
		if _, err := pgtx.Exec(context.Background(),
			`UPDATE transactions
             SET status = 'completed'
             WHERE id = $1`,
			created.ID,
		); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		_ = s.trx.UpdateStatus(created.ID, models.TxnRolledBack)
		s.audit(created.ID, "status_change", fmt.Sprintf("%s: %s", models.TxnRolledBack, err.Error()))
		metrics.TransactionsFailed.Inc()
		return models.Transaction{}, err
	}

	created.Status = models.TxnCompleted
	s.audit(created.ID, "status_change", "completed: transfer applied")
	metrics.TransactionsTotal.WithLabelValues("transfer").Inc()
	return created, nil
}

// ----------------- Queries -----------------

func (s *TransactionService) GetByID(id string) (models.Transaction, error) {
	return s.trx.GetByID(id)
}

func (s *TransactionService) ListByUser(userID string, limit, offset int) ([]models.Transaction, error) {
	return s.trx.ListByUser(userID, limit, offset)
}

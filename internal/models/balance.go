package models

import (
	"sync"
	"time"
)

type Balance struct {
	UserID        string    `json:"user_id"`
	Amount        int64     `json:"amount"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
	mu            sync.RWMutex
}

func (b *Balance) Add(delta int64) int64 {
	b.mu.Lock(); defer b.mu.Unlock()
	b.Amount += delta
	b.LastUpdatedAt = time.Now()
	return b.Amount
}

func (b *Balance) Get() int64 {
	b.mu.RLock(); defer b.mu.RUnlock(); return b.Amount
}
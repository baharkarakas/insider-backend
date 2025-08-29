package services

import (
	"github.com/baharkarakas/insider-backend/internal/models"
	repo "github.com/baharkarakas/insider-backend/internal/repository"
)

type BalanceService struct { r repo.Balances }

func NewBalanceService(r repo.Balances) *BalanceService { return &BalanceService{r: r} }

func (s *BalanceService) Current(userID string) (models.Balance, error) { return s.r.GetOrCreate(userID) }
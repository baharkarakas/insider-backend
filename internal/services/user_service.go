package services

import (
	"errors"
	"strings"

	"github.com/baharkarakas/insider-backend/internal/auth"
	"github.com/baharkarakas/insider-backend/internal/config"
	"github.com/baharkarakas/insider-backend/internal/models"
	repo "github.com/baharkarakas/insider-backend/internal/repository"
)

type UserService struct {
	r repo.Users
	c config.Config
}

func NewUserService(r repo.Users, c config.Config) *UserService { return &UserService{r: r, c: c} }

func (s *UserService) Register(username, email, password string) (models.User, error) {
	u := models.User{Username: strings.TrimSpace(username), Email: strings.TrimSpace(email), Role: "user"}
	if err := u.Validate(); err != nil { return models.User{}, err }
	hash, err := auth.HashPassword(password)
	if err != nil { return models.User{}, err }
	return s.r.Create(u.Username, u.Email, hash, u.Role)
}

// internal/services/user_service.go
// internal/services/user_service.go
func (s *UserService) Login(email, password string) (string, error) {
    return "", errors.New("deprecated: use /api/v1/auth/login")
}




func (s *UserService) List() ([]models.User, error) { return s.r.List() }
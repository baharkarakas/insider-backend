package services

import (
	"errors"
	"strings"
	"time"

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

func (s *UserService) Login(email, password string) (string, error) {
	u, err := s.r.GetByEmail(strings.TrimSpace(email))
	if err != nil { return "", errors.New("invalid credentials") }
	if !auth.CheckPassword(u.PasswordHash, password) { return "", errors.New("invalid credentials") }
	return auth.NewToken(s.c.JWTSecret, s.c.JWTIssuer, u.ID, u.Role, 24*time.Hour)
}

func (s *UserService) List() ([]models.User, error) { return s.r.List() }
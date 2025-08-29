package models

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	PasswordHash string   `json:"-"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (u *User) Validate() error {
	if len(strings.TrimSpace(u.Username)) < 3 { return errors.New("username too short") }
	if !strings.Contains(u.Email, "@") { return errors.New("invalid email") }
	if u.Role == "" { u.Role = "user" }
	return nil
}
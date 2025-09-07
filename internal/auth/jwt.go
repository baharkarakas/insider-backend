// internal/auth/jwt.go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	Type   string `json:"typ"` // "access" | "refresh"
	jwt.RegisteredClaims
}

// GeneratePair: access + refresh 
func (tm *TokenManager) GeneratePair(userID, role string) (access string, refresh string, accessExp time.Time, err error) {
	now := time.Now()

	accClaims := Claims{
		UserID: userID,
		Role:   role,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessTTL)),
		},
	}
	refClaims := Claims{
		UserID: userID,
		Role:   role,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.refreshTTL)),
		},
	}

	accTok := jwt.NewWithClaims(jwt.SigningMethodHS256, accClaims)
	refTok := jwt.NewWithClaims(jwt.SigningMethodHS256, refClaims)

	access, err = accTok.SignedString(tm.accessSecret)
	if err != nil {
		return "", "", time.Time{}, err
	}
	refresh, err = refTok.SignedString(tm.refreshSecret)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return access, refresh, accClaims.ExpiresAt.Time, nil
}

// ParseAny
func (tm *TokenManager) ParseAny(tokenStr string) (*Claims, bool, error) {
	
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return tm.accessSecret, nil
	})
	if err == nil && claims.Type == "access" {
		return claims, false, nil
	}


	claims = &Claims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return tm.refreshSecret, nil
	})
	if err == nil && claims.Type == "refresh" {
		return claims, true, nil
	}
	return nil, false, errors.New("invalid token")
}

package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/baharkarakas/insider-backend/internal/auth"
	"github.com/baharkarakas/insider-backend/internal/services"
)

type AuthHandler struct {
	TM     *auth.TokenManager
	Users  *services.UserService
	AppEnv string
}

func NewAuthHandler(tm *auth.TokenManager, us *services.UserService) *AuthHandler {
	return &AuthHandler{
		TM:     tm,
		Users:  us,
		AppEnv: os.Getenv("APP_ENV"),
	}
}

type loginReq struct {
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`

	
	UserID string `json:"user_id,omitempty"`
	Role   string `json:"role,omitempty"`
}

type tokenResp struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req loginReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	// 1) Normal: email+password
	if req.Email != "" && req.Password != "" {
		u, err := h.Users.GetByEmailAndPassword(req.Email, req.Password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid credentials"})
			return
		}
		access, refresh, exp, err := h.TM.GeneratePair(u.ID, u.Role)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "token generation failed"})
			return
		}
		_ = json.NewEncoder(w).Encode(tokenResp{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresIn:    time.Until(exp).Truncate(time.Second),
		})
		return
	}

	// 2) DEV kısa yol 
	if h.AppEnv == "dev" {
		if req.UserID == "" {
			req.UserID = "00000000-0000-0000-0000-000000000000"
		}
		if req.Role == "" {
			req.Role = "user"
		}
		access, refresh, exp, err := h.TM.GeneratePair(req.UserID, req.Role)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "token generation failed"})
			return
		}
		_ = json.NewEncoder(w).Encode(tokenResp{
			AccessToken:  access,
			RefreshToken: refresh,
			ExpiresIn:    time.Until(exp).Truncate(time.Second),
		})
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "email & password required"})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}
	claims, isRefresh, err := h.TM.ParseAny(req.RefreshToken)
	if err != nil || !isRefresh {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid refresh token"})
		return
	}
	access, refresh, exp, err := h.TM.GeneratePair(claims.UserID, claims.Role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "token generation failed"})
		return
	}
	_ = json.NewEncoder(w).Encode(tokenResp{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    time.Until(exp).Truncate(time.Second),
	})
}

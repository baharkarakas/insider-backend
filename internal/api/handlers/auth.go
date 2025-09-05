// internal/api/handlers/auth.go
package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/baharkarakas/insider-backend/internal/auth"
)

type AuthHandler struct {
	TM     *auth.TokenManager
	AppEnv string
}

func NewAuthHandler(tm *auth.TokenManager) *AuthHandler {
	return &AuthHandler{
		TM:     tm,
		AppEnv: os.Getenv("APP_ENV"),
	}
}

type loginReq struct {
	// Gelecekte: Email/Username + Password bekleyebilirsin
	// Dev kısa yol: user_id ve role da kabul edelim
	UserID string `json:"user_id,omitempty"`
	Role   string `json:"role,omitempty"`
}

type tokenResp struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    time.Duration `json:"expires_in"` // access süresi
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// DEV hızlı yol:
	if h.AppEnv == "dev" {
		// 1) Header'da dev-<uuid> varsa onu kullan
		if ah := r.Header.Get("Authorization"); len(ah) > 6 && ah[:6] == "Bearer" {
			// middleware zaten dev-<uuid> ile geçmeye izin veriyor, ama login'de JWT üretmek istiyoruz.
			// Bu yüzden body'den de kabul edelim:
		}

		var req loginReq
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.UserID == "" {
			// Çok kısa yol: herhangi bir user id yoksa varsayılan örnek
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

	// PROD (veya dev dışı): Buraya gerçek kimlik doğrulamayı bağlayacağız
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "login not implemented yet"})
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

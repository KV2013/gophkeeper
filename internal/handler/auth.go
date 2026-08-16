package handler

import (
	"encoding/json"
	"net/http"

	"github.com/victor/gophkeeper/internal/service"
)

// credentialsRequest — запрос на регистрацию/аутентификацию.
type credentialsRequest struct {
	// Login — логин пользователя.
	Login string `json:"login"`
	// Password — пароль пользователя.
	Password string `json:"password"`
}

// tokenResponse — ответ с JWT-токеном и солью KDF.
type tokenResponse struct {
	// Token — JWT-токен доступа.
	Token string `json:"token"`
	// Salt — соль KDF для вывода мастер-ключа на клиенте (base64).
	Salt []byte `json:"salt"`
}

// Register обрабатывает POST /api/v1/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	token, salt, err := h.auth.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, tokenResponse{Token: token, Salt: salt})
}

// Login обрабатывает POST /api/v1/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	token, salt, err := h.auth.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, tokenResponse{Token: token, Salt: salt})
}

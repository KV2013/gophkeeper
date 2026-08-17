// Package handler содержит HTTP-обработчики REST API сервера.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/middleware"
	"github.com/victor/gophkeeper/internal/service"
)

// Handler — обработчик REST API.
type Handler struct {
	auth   *service.AuthService
	object *service.ObjectService
	file   *service.FileService
	logger *zap.Logger
}

// New создаёт обработчик REST API.
func New(auth *service.AuthService, object *service.ObjectService, file *service.FileService, logger *zap.Logger) *Handler {
	return &Handler{auth: auth, object: object, file: file, logger: logger}
}

// userID возвращает идентификатор пользователя из контекста запроса.
func userID(ctx context.Context) string {
	return middleware.UserID(ctx)
}

// writeJSON сериализует ответ в JSON.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("не удалось записать ответ", zap.Error(err))
	}
}

// writeError записывает JSON-ответ с ошибкой.
func (h *Handler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrBadRequest):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrInvalidCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	}
	h.writeJSON(w, status, map[string]string{"error": err.Error()})
}

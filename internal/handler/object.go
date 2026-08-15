package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/victor/gophkeeper/internal/model"
	"github.com/victor/gophkeeper/internal/service"
)

// createObjectRequest — запрос на создание/обновление объекта.
type createObjectRequest struct {
	// Type — тип объекта.
	Type model.SecretType `json:"type"`
	// Salt — соль KDF (base64).
	Salt []byte `json:"salt"`
	// Ciphertext — зашифрованные данные (base64).
	Ciphertext []byte `json:"ciphertext"`
}

// metadataRequest — запрос на создание/обновление метаданных.
type metadataRequest struct {
	// Name — имя метаданных.
	Name string `json:"name"`
	// OrderNumber — порядковый номер для сортировки.
	OrderNumber int `json:"order_number"`
	// Options — произвольные пары ключ/значение.
	Options map[string]string `json:"options"`
}

// CreateObject обрабатывает POST /api/v1/object.
func (h *Handler) CreateObject(w http.ResponseWriter, r *http.Request) {
	var req createObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	obj, err := h.object.CreateObject(r.Context(), userID(r.Context()), req.Type, req.Salt, req.Ciphertext)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, obj)
}

// ListObjects обрабатывает GET /api/v1/object.
func (h *Handler) ListObjects(w http.ResponseWriter, r *http.Request) {
	objects, err := h.object.ListObjects(r.Context(), userID(r.Context()))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, objects)
}

// GetObject обрабатывает GET /api/v1/object/{id}.
func (h *Handler) GetObject(w http.ResponseWriter, r *http.Request) {
	obj, err := h.object.GetObject(r.Context(), userID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, obj)
}

// UpdateObject обрабатывает PUT /api/v1/object/{id}.
func (h *Handler) UpdateObject(w http.ResponseWriter, r *http.Request) {
	var req createObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	obj, err := h.object.UpdateObject(r.Context(), userID(r.Context()), chi.URLParam(r, "id"), req.Type, req.Salt, req.Ciphertext)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, obj)
}

// DeleteObject обрабатывает DELETE /api/v1/object/{id}.
func (h *Handler) DeleteObject(w http.ResponseWriter, r *http.Request) {
	if err := h.object.DeleteObject(r.Context(), userID(r.Context()), chi.URLParam(r, "id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateMetadata обрабатывает POST /api/v1/object/{id}/metadata.
func (h *Handler) CreateMetadata(w http.ResponseWriter, r *http.Request) {
	var req metadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	m, err := h.object.CreateMetadata(r.Context(), userID(r.Context()), chi.URLParam(r, "id"), req.Name, req.OrderNumber, req.Options)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, m)
}

// ListMetadata обрабатывает GET /api/v1/object/{id}/metadata.
func (h *Handler) ListMetadata(w http.ResponseWriter, r *http.Request) {
	metadata, err := h.object.ListMetadata(r.Context(), userID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, metadata)
}

// UpdateMetadata обрабатывает PUT /api/v1/object/{id}/metadata/{metaID}.
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	var req metadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	m, err := h.object.UpdateMetadata(r.Context(), userID(r.Context()), chi.URLParam(r, "id"), chi.URLParam(r, "metaID"), req.Name, req.OrderNumber, req.Options)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, m)
}

// DeleteMetadata обрабатывает DELETE /api/v1/object/{id}/metadata/{metaID}.
func (h *Handler) DeleteMetadata(w http.ResponseWriter, r *http.Request) {
	err := h.object.DeleteMetadata(r.Context(), userID(r.Context()), chi.URLParam(r, "id"), chi.URLParam(r, "metaID"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

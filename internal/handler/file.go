package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/model"
	"github.com/victor/gophkeeper/internal/service"
)

// maxFileSize — максимальный размер загружаемого файла (4 ГБ).
const maxFileSize = 4 << 30

// UploadFile обрабатывает PUT /api/v1/files/{id} — потоковая загрузка файла.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := userID(r.Context())

	obj, err := h.object.GetObject(r.Context(), userID, id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if obj.Type != model.SecretTypeBinary {
		h.writeError(w, service.ErrBadRequest)
		return
	}
	if r.ContentLength > maxFileSize {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	body := http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := h.file.Upload(r.Context(), userID, id, body, r.ContentLength); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DownloadFile обрабатывает GET /api/v1/files/{id} — потоковое скачивание файла.
func (h *Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := userID(r.Context())

	obj, err := h.object.GetObject(r.Context(), userID, id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if obj.Type != model.SecretTypeBinary {
		h.writeError(w, service.ErrBadRequest)
		return
	}

	rc, size, err := h.file.Download(r.Context(), userID, id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		h.logger.Debug("ошибка при передаче файла", zap.Error(err))
	}
}

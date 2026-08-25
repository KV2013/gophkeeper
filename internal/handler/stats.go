package handler

import "net/http"

// versionResponse — ответ с информацией о сборке сервера.
type versionResponse struct {
	Version     string `json:"version"`
	BuildDate   string `json:"build_date"`
	BuildCommit string `json:"build_commit"`
}

// Version обрабатывает GET /api/v1/version.
func (h *Handler) Version(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, versionResponse{
		Version:     h.version.Version,
		BuildDate:   h.version.BuildDate,
		BuildCommit: h.version.BuildCommit,
	})
}

// statsResponse — ответ со статистикой пользователя.
type statsResponse struct {
	ObjectsCount   int   `json:"objects_count"`
	FilesCount     int   `json:"files_count"`
	FilesTotalSize int64 `json:"files_total_size"`
}

// Stats обрабатывает GET /api/v1/stats.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	uid := userID(r.Context())

	objects, err := h.object.Count(r.Context(), uid)
	if err != nil {
		h.writeError(w, err)
		return
	}
	files, size, err := h.file.Stats(r.Context(), uid)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, statsResponse{
		ObjectsCount:   objects,
		FilesCount:     files,
		FilesTotalSize: size,
	})
}

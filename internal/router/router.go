// Package router настраивает маршруты HTTP-сервера и подключает middleware.
package router

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/config"
	"github.com/victor/gophkeeper/internal/handler"
	"github.com/victor/gophkeeper/internal/middleware"
)

// Init создаёт HTTP-обработчик со всеми маршрутами и middleware.
func Init(h *handler.Handler, logger *zap.Logger, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	auth := middleware.AuthJWT(cfg.JWTSecret, logger)

	// protected регистрирует маршрут, защищённый JWT-авторизацией.
	protected := func(pattern string, hf http.HandlerFunc) {
		mux.Handle(pattern, auth(hf))
	}

	mux.HandleFunc("GET /api/v1/version", h.Version)
	mux.HandleFunc("POST /api/v1/register", h.Register)
	mux.HandleFunc("POST /api/v1/login", h.Login)

	protected("POST /api/v1/objects", h.CreateObject)
	protected("GET /api/v1/objects", h.ListObjects)
	protected("GET /api/v1/objects/{id}", h.GetObject)
	protected("PUT /api/v1/objects/{id}", h.UpdateObject)
	protected("DELETE /api/v1/objects/{id}", h.DeleteObject)

	protected("GET /api/v1/stats", h.Stats)

	protected("PUT /api/v1/files/{id}", h.UploadFile)
	protected("GET /api/v1/files/{id}", h.DownloadFile)

	protected("POST /api/v1/objects/{id}/metadata", h.CreateMetadata)
	protected("GET /api/v1/objects/{id}/metadata", h.ListMetadata)
	protected("PUT /api/v1/objects/{id}/metadata/{metaID}", h.UpdateMetadata)
	protected("DELETE /api/v1/objects/{id}/metadata/{metaID}", h.DeleteMetadata)

	return middleware.Chain(
		middleware.RequestID(),
		middleware.ZapLogger(logger),
		middleware.GzipCompression,
		middleware.Recoverer(logger),
	)(mux)
}

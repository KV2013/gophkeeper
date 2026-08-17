// Package router настраивает маршруты HTTP-сервера и подключает middleware.
package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/config"
	"github.com/victor/gophkeeper/internal/handler"
	"github.com/victor/gophkeeper/internal/middleware"
)

// Init создаёт роутер со всеми маршрутами и middleware.
func Init(h *handler.Handler, logger *zap.Logger, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.ZapLogger(logger))
	r.Use(middleware.GzipCompression)
	r.Use(chimiddleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthJWT(cfg.JWTSecret, logger))

			r.Post("/objects", h.CreateObject)
			r.Get("/objects", h.ListObjects)
			r.Get("/objects/{id}", h.GetObject)
			r.Put("/objects/{id}", h.UpdateObject)
			r.Delete("/objects/{id}", h.DeleteObject)

			r.Put("/files/{id}", h.UploadFile)
			r.Get("/files/{id}", h.DownloadFile)

			r.Post("/objects/{id}/metadata", h.CreateMetadata)
			r.Get("/objects/{id}/metadata", h.ListMetadata)
			r.Put("/objects/{id}/metadata/{metaID}", h.UpdateMetadata)
			r.Delete("/objects/{id}/metadata/{metaID}", h.DeleteMetadata)
		})
	})

	return r
}

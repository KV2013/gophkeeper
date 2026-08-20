package middleware

import (
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	applogger "github.com/victor/gophkeeper/internal/logger"
	"go.uber.org/zap"
)

// ZapLogger — middleware, логирующий HTTP-запросы с помощью zap.
func ZapLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return chimiddleware.RequestLogger(&applogger.ZapLogFormatter{
		Logger:    logger,
		UserIDKey: UserIDKey,
	})
}

package middleware

import (
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
)

// Recoverer перехватывает панику в обработчиках, логирует стек и возвращает 500.
func Recoverer(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("request panic",
						zap.Any("panic", rec),
						zap.ByteString("stack", debug.Stack()),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// requestIDKey — ключ контекста для хранения идентификатора запроса.
type requestIDKey string

const requestIDKeyValue requestIDKey = "request_id"

// RequestID присваивает каждому запросу уникальный идентификатор (UUID),
// кладёт его в контекст и заголовок X-Request-Id.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := uuid.NewString()
			w.Header().Set("X-Request-Id", id)
			ctx := context.WithValue(r.Context(), requestIDKeyValue, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestIDFromContext возвращает идентификатор запроса из контекста.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKeyValue).(string); ok {
		return v
	}
	return ""
}

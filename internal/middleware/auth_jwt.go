// Package middleware содержит HTTP-промежуточные слои: логирование,
// сжатие и JWT-аутентификацию.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// contextKey — тип ключа контекста для хранения идентификатора пользователя.
type contextKey string

// UserIDKey — ключ контекста, по которому хранится идентификатор пользователя.
const UserIDKey contextKey = "user_id"

// UserID возвращает идентификатор аутентифицированного пользователя из контекста.
func UserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// AuthJWT проверяет JWT-токен из заголовка "Authorization: Bearer <token>"
// и кладёт идентификатор пользователя в контекст запроса.
func AuthJWT(secret string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "missing or invalid token", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
			if err != nil || !token.Valid {
				logger.Debug("недействительный JWT-токен", zap.Error(err))
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			userID, _ := claims["sub"].(string)
			if userID == "" {
				http.Error(w, "invalid token subject", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

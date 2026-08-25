package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const testSecret = "test-secret"

func issueTestToken(t *testing.T, userID string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return s
}

func runAuthMiddleware(t *testing.T, authHeader string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	var captured string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = UserID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	h := AuthJWT(testSecret, zap.NewNop())(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/object", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, captured
}

func TestAuthJWTValidToken(t *testing.T) {
	token := issueTestToken(t, "user-42")
	rec, captured := runAuthMiddleware(t, "Bearer "+token)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if captured != "user-42" {
		t.Fatalf("user id: got %q, want user-42", captured)
	}
}

func TestAuthJWTMissingHeader(t *testing.T) {
	rec, _ := runAuthMiddleware(t, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rec.Code)
	}
}

func TestAuthJWTInvalidToken(t *testing.T) {
	rec, _ := runAuthMiddleware(t, "Bearer not-a-valid-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rec.Code)
	}
}

func TestAuthJWTWrongSecret(t *testing.T) {
	claims := jwt.RegisteredClaims{Subject: "user-42"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte("another-secret"))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	rec, _ := runAuthMiddleware(t, "Bearer "+s)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rec.Code)
	}
}

func TestUserIDEmpty(t *testing.T) {
	if UserID(t.Context()) != "" {
		t.Fatal("UserID должен возвращать пустую строку для пустого контекста")
	}
}

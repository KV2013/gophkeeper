package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestAuthService(repo userRepository) *AuthService {
	return NewAuthService(repo, zap.NewNop(), "test-secret", time.Hour)
}

func TestRegisterSuccess(t *testing.T) {
	svc := newTestAuthService(newMockRepo())

	token, err := svc.Register(context.Background(), "bob", "hunter2")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if token == "" {
		t.Fatal("ожидался непустой токен")
	}
}

func TestRegisterDuplicateLogin(t *testing.T) {
	repo := newMockRepo()
	svc := newTestAuthService(repo)

	if _, err := svc.Register(context.Background(), "bob", "hunter2"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, err := svc.Register(context.Background(), "bob", "hunter2")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("ожидалась ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterEmptyCredentials(t *testing.T) {
	svc := newTestAuthService(newMockRepo())

	if _, err := svc.Register(context.Background(), "", "hunter2"); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("ожидалась ErrBadRequest для пустого логина, got %v", err)
	}
	if _, err := svc.Register(context.Background(), "bob", ""); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("ожидалась ErrBadRequest для пустого пароля, got %v", err)
	}
}

func TestLoginSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := newTestAuthService(repo)

	if _, err := svc.Register(context.Background(), "bob", "hunter2"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	token, err := svc.Login(context.Background(), "bob", "hunter2")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("ожидался непустой токен")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repo := newMockRepo()
	svc := newTestAuthService(repo)

	if _, err := svc.Register(context.Background(), "bob", "hunter2"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, err := svc.Login(context.Background(), "bob", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("ожидалась ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc := newTestAuthService(newMockRepo())

	_, err := svc.Login(context.Background(), "nobody", "hunter2")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("ожидалась ErrInvalidCredentials, got %v", err)
	}
}

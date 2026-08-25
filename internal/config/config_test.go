package config

import (
	"testing"
	"time"
)

func TestNewConfigDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	if cfg.ServerAddress != "localhost:8080" {
		t.Errorf("ServerAddress: got %q, want localhost:8080", cfg.ServerAddress)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want info", cfg.LogLevel)
	}
	if cfg.TokenTTL != time.Hour {
		t.Errorf("TokenTTL: got %v, want 1h", cfg.TokenTTL)
	}
	if !cfg.EnableHTTPS {
		t.Error("EnableHTTPS: по умолчанию должен быть true")
	}
	if cfg.JWTSecret != "secret" {
		t.Errorf("JWTSecret: got %q, want secret", cfg.JWTSecret)
	}
}

func TestNewConfigMissingSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if _, err := NewConfig(); err == nil {
		t.Fatal("ожидалась ошибка при отсутствии JWT_SECRET")
	}
}

func TestNewConfigEnvOverride(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("SERVER_ADDRESS", ":9000")
	t.Setenv("TOKEN_TTL", "1h30m")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig: %v", err)
	}

	if cfg.ServerAddress != ":9000" {
		t.Errorf("ServerAddress: got %q, want :9000", cfg.ServerAddress)
	}
	if cfg.TokenTTL != 90*time.Minute {
		t.Errorf("TokenTTL: got %v, want 1h30m", cfg.TokenTTL)
	}
}

func TestNewConfigInvalidTTL(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("TOKEN_TTL", "not-a-duration")
	if _, err := NewConfig(); err == nil {
		t.Fatal("ожидалась ошибка при неверном TOKEN_TTL")
	}
}

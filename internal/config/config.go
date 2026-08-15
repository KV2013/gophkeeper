// Package config отвечает за загрузку и валидацию конфигурации сервера
// из переменных окружения.
package config

import (
	"errors"
	"time"

	"github.com/caarlos0/env/v6"
)

// Config — конфигурация сервера.
type Config struct {
	// ServerAddress — адрес, на котором слушает HTTP-сервер.
	ServerAddress string `env:"SERVER_ADDRESS" json:"server_address"`
	// LogLevel — уровень логирования (debug, info, warn, error).
	LogLevel string `env:"LOG_LEVEL" json:"log_level"`
	// DatabaseDSN — строка подключения к PostgreSQL.
	DatabaseDSN string `env:"DATABASE_DSN" json:"database_dsn"`
	// JWTSecret — секрет для подписи JWT-токенов.
	JWTSecret string `env:"JWT_SECRET" json:"jwt_secret"`
	// TokenTTL — время жизни JWT-токена.
	TokenTTL time.Duration `env:"TOKEN_TTL" json:"token_ttl"`
	// EnableHTTPS — использовать ли HTTPS с самоподписным сертификатом.
	EnableHTTPS bool `env:"ENABLE_HTTPS" json:"enable_https"`
}

// NewConfig собирает конфигурацию из переменных окружения и валидирует её.
func NewConfig() (*Config, error) {

	parseFlags()

	cfg := Config{
		ServerAddress: "localhost:8080",
		LogLevel:      "info",
		TokenTTL:      1 * time.Hour,
		EnableHTTPS:   true,
	}

	if err := LoadConfigFile(&cfg); err != nil {
		return nil, err
	}

	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	if flagServerAddress != "" {
		cfg.ServerAddress = flagServerAddress
	}
	if flagLogLevel != "" {
		cfg.LogLevel = flagLogLevel
	}
	if flagDatabaseDSN != "" {
		cfg.DatabaseDSN = flagDatabaseDSN
	}
	if flagJWTSecret != "" {
		cfg.JWTSecret = flagJWTSecret
	}
	if flagTokenTTL != 0 {
		cfg.TokenTTL = flagTokenTTL
	}
	if flagEnableHTTPS != false {
		cfg.EnableHTTPS = flagEnableHTTPS
	}

	if cfg.JWTSecret == "" {
		return nil, errors.New("config: переменная окружения JWT_SECRET не задана")
	}

	return &cfg, nil
}

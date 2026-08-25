package config

import (
	"time"

	flag "github.com/spf13/pflag"
)

// Flags — значения флагов командной строки.
type Flags struct {
	// ServerAddress — адрес сервера.
	ServerAddress string
	// LogLevel — уровень логирования.
	LogLevel string
	// DatabaseDSN — строка подключения к PostgreSQL.
	DatabaseDSN string
	// JWTSecret — секрет для подписи JWT-токенов.
	JWTSecret string
	// TokenTTL — время жизни JWT-токена.
	TokenTTL time.Duration
	// EnableHTTPS — использовать ли HTTPS.
	EnableHTTPS bool
	// ConfigPath — путь к файлу конфигурации.
	ConfigPath string
}

// parseFlags регистрирует флаги командной строки и парсит их в f.
func parseFlags(f *Flags) {
	if flag.Parsed() {
		return
	}

	flag.StringVarP(&f.ServerAddress, "server-address", "s", "", "Server address")
	flag.StringVarP(&f.LogLevel, "log-level", "l", "info", "Log level")
	flag.StringVarP(&f.DatabaseDSN, "database-dsn", "d", "", "Database DSN")
	flag.StringVarP(&f.JWTSecret, "jwt-secret", "j", "", "JWT secret")
	flag.DurationVarP(&f.TokenTTL, "token-ttl", "t", 0, "Token TTL")
	flag.BoolVarP(&f.EnableHTTPS, "enable-https", "e", true, "Enable HTTPS")
	flag.StringVarP(&f.ConfigPath, "config", "c", "", "Path to config file")

	flag.Parse()
}

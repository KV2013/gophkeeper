// Package config отвечает за конфигурацию клиента, хранящуюся в SQLite
// (таблица client_config), в частности за TTL производного ключа.
package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/victor/gophkeeper/internal/client/repository"
)

// Ключи и значения конфигурации по умолчанию.
const (
	// KeyMasterKeyTTL — ключ TTL производного мастер-ключа.
	KeyMasterKeyTTL = "master_key_ttl"
	// DefaultMasterKeyTTL — TTL мастер-ключа по умолчанию (5 минут).
	DefaultMasterKeyTTL = 5 * time.Minute
	// MaxMasterKeyTTL — максимальный TTL мастер-ключа (24 часа).
	MaxMasterKeyTTL = 24 * time.Hour
)

// Ключи конфигурации подключения к серверу.
const (
	// KeyConnectServerAddress — адрес сервера.
	KeyConnectServerAddress = "connect_server_address"
	// KeyConnectInsecure — не проверять TLS-сертификат ("true"/"false").
	KeyConnectInsecure = "connect_insecure"
	// KeyConnectCACert — путь к CA-сертификату (PEM).
	KeyConnectCACert = "connect_cacert"
)

// ErrInvalidTTL — переданное значение TTL некорректно.
var ErrInvalidTTL = errors.New("config: неверное значение TTL")

// Reader — минимальный интерфейс чтения конфигурации из хранилища.
type Reader interface {
	// GetConfig возвращает значение конфигурации по ключу.
	GetConfig(ctx context.Context, key string) (string, error)
}

// ParseTTL разбирает строку TTL в формате time.Duration и валидирует её.
//
// Допустимые значения: 0 (без ограничения) и (0; 24h]. Отрицательные значения
// и значения больше MaxMasterKeyTTL отклоняются.
func ParseTTL(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidTTL, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%w: TTL не может быть отрицательным", ErrInvalidTTL)
	}
	if d > MaxMasterKeyTTL {
		return 0, fmt.Errorf("%w: TTL не может превышать 24 часа", ErrInvalidTTL)
	}
	return d, nil
}

// MasterKeyTTL возвращает TTL мастер-ключа из хранилища или значение по
// умолчанию, если конфигурация не задана.
func MasterKeyTTL(ctx context.Context, r Reader) (time.Duration, error) {
	v, err := r.GetConfig(ctx, KeyMasterKeyTTL)
	if errors.Is(err, repository.ErrNotFound) {
		return DefaultMasterKeyTTL, nil
	}
	if err != nil {
		return 0, err
	}
	return ParseTTL(v)
}

// IsExpired возвращает true, если createdAt «прожил» дольше ttl.
// При ttl == 0 значение никогда не считается протухшим.
func IsExpired(createdAt, now time.Time, ttl time.Duration) bool {
	if ttl == 0 {
		return false
	}
	return now.Sub(createdAt) > ttl
}

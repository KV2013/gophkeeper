// Package config отвечает за конфигурацию клиента, хранящуюся в SQLite
// (таблица client_config), в частности за TTL производного ключа.
package config

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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
	// KeyUseCredentialsFile — хранить секреты в файле credentials.json
	// вместо системного keyring ("true"/"false").
	KeyUseCredentialsFile = "use_credentials_file"
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

// UseCredentialsFile возвращает true, если секреты нужно хранить в файле
// credentials.json вместо системного keyring. По умолчанию — false.
func UseCredentialsFile(ctx context.Context, r Reader) (bool, error) {
	v, err := r.GetConfig(ctx, KeyUseCredentialsFile)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("config: неверное значение %s: %w", KeyUseCredentialsFile, err)
	}
	return b, nil
}

// IsExpired возвращает true, если createdAt «прожил» дольше ttl.
// При ttl == 0 значение никогда не считается протухшим.
func IsExpired(createdAt, now time.Time, ttl time.Duration) bool {
	if ttl == 0 {
		return false
	}
	return now.Sub(createdAt) > ttl
}

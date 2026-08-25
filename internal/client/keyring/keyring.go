// Package keyring хранит секреты клиента (JWT-токен, соль) в системном
// keyring с фолбэком на файл с правами 0600.
package keyring

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	gokeyring "github.com/zalando/go-keyring"
)

// Ключи, под которыми хранятся секреты.
const (
	// KeyToken — JWT-токен доступа.
	KeyToken = "token"
	// KeySalt — соль KDF (hex).
	KeySalt = "salt"
	// KeyMasterKey — производный ключ шифрования (hex).
	KeyMasterKey = "master_key"
	// KeyMasterKeyCreatedAt — время создания ключа (unix-секунды).
	KeyMasterKeyCreatedAt = "master_key_created_at"
	// KeyTokenExpiresAt — время истечения JWT-токена (unix-секунды).
	KeyTokenExpiresAt = "token_expires_at"
)

// ErrNotFound — секрет с заданным ключом не найден.
var ErrNotFound = errors.New("keyring: секрет не найден")

// Store — хранилище секретов клиента.
//
// Работает строго в одном режиме: либо системный keyring (useOS=true), либо
// файл-фолбэк credentials.json (useOS=false).
type Store struct {
	service string
	file    string
	useOS   bool
}

// New создаёт Store. При useOS=true используется системный keyring, при
// useOS=false — файл credentials.json.
func New(service, file string, useOS bool) *Store {
	return &Store{service: service, file: file, useOS: useOS}
}

// NewFile создаёт Store, использующий только файл (для headless/тестов).
func NewFile(service, file string) *Store {
	return &Store{service: service, file: file, useOS: false}
}

// Set сохраняет значение секрета под ключом.
func (s *Store) Set(key, value string) error {
	if s.useOS {
		return gokeyring.Set(s.service, key, value)
	}
	return s.setFile(key, value)
}

// Get возвращает значение секрета по ключу.
func (s *Store) Get(key string) (string, error) {
	if s.useOS {
		return gokeyring.Get(s.service, key)
	}
	return s.getFile(key)
}

// Delete удаляет значение секрета по ключу.
func (s *Store) Delete(key string) error {
	if s.useOS {
		return gokeyring.Delete(s.service, key)
	}
	return s.deleteFile(key)
}

// setFile сохраняет значение в файл-фолбэк.
func (s *Store) setFile(key, value string) error {
	m, err := s.readFile()
	if err != nil {
		return err
	}
	m[key] = value
	return s.writeFile(m)
}

// getFile читает значение из файла-фолбэка.
func (s *Store) getFile(key string) (string, error) {
	m, err := s.readFile()
	if err != nil {
		return "", err
	}
	v, ok := m[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

// deleteFile удаляет значение из файла-фолбэка.
func (s *Store) deleteFile(key string) error {
	m, err := s.readFile()
	if err != nil {
		return err
	}
	delete(m, key)
	return s.writeFile(m)
}

// readFile читает карту секретов из файла.
func (s *Store) readFile() (map[string]string, error) {
	m := map[string]string{}
	data, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// writeFile записывает карту секретов в файл с правами 0600.
func (s *Store) writeFile(m map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.file), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.file, data, 0o600)
}

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
)

// ErrNotFound — секрет с заданным ключом не найден.
var ErrNotFound = errors.New("keyring: секрет не найден")

// Store — хранилище секретов клиента.
type Store struct {
	service string
	file    string
	useOS   bool
}

// New создаёт Store, использующий системный keyring с фолбэком на файл.
func New(service, fallbackFile string) *Store {
	return &Store{service: service, file: fallbackFile, useOS: true}
}

// NewFile создаёт Store, использующий только файл.
func NewFile(service, fallbackFile string) *Store {
	return &Store{service: service, file: fallbackFile, useOS: false}
}

// Set сохраняет значение секрета под ключом.
func (s *Store) Set(key, value string) error {
	if s.useOS {
		if err := gokeyring.Set(s.service, key, value); err == nil {
			return nil
		}
	}
	return s.setFile(key, value)
}

// Get возвращает значение секрета по ключу.
func (s *Store) Get(key string) (string, error) {
	if s.useOS {
		if v, err := gokeyring.Get(s.service, key); err == nil {
			return v, nil
		}
	}
	return s.getFile(key)
}

// Delete удаляет значение секрета по ключу.
func (s *Store) Delete(key string) error {
	if s.useOS {
		_ = gokeyring.Delete(s.service, key)
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

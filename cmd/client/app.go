package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/victor/gophkeeper/internal/client/api"
	clientkeyring "github.com/victor/gophkeeper/internal/client/keyring"
	clientpath "github.com/victor/gophkeeper/internal/client/path"
	"github.com/victor/gophkeeper/internal/client/repository"
	"github.com/victor/gophkeeper/internal/client/sync"
	"github.com/victor/gophkeeper/internal/crypto"
)

// keyringService — имя сервиса в системном keyring.
const keyringService = "gophkeeper"

// app — связка API-клиента, локального кэша и keyring.
type app struct {
	api     *api.Client
	store   *repository.Repository
	keyring *clientkeyring.Store
	sync    *sync.Syncer
}

// newApp инициализирует клиентское приложение.
func newApp(serverURL string) (*app, error) {
	dataDir, err := clientpath.DataDir()
	if err != nil {
		return nil, err
	}

	store, err := repository.New(filepath.Join(dataDir, "gophkeeper.db"))
	if err != nil {
		return nil, err
	}

	kr := clientkeyring.New(keyringService, filepath.Join(dataDir, "credentials.json"))

	var opts []api.Option
	if caCertPath != "" {
		opts = append(opts, api.WithCACertFile(caCertPath))
	}
	if insecure {
		opts = append(opts, api.WithInsecure())
	}

	apiClient, err := api.New(serverURL, opts...)
	if err != nil {
		return nil, err
	}

	return &app{
		api:     apiClient,
		store:   store,
		keyring: kr,
		sync:    sync.New(apiClient, store),
	}, nil
}

// mustApp создаёт приложение или завершает процесс с ошибкой.
func mustApp(serverURL string) *app {
	a, err := newApp(serverURL)
	if err != nil {
		fatal("не удалось инициализировать клиент: %v", err)
	}
	return a
}

// close закрывает локальное хранилище.
func (a *app) close() {
	_ = a.store.Close()
}

// requireToken возвращает сохранённый JWT-токен.
func (a *app) requireToken() (string, error) {
	token, err := a.keyring.Get(clientkeyring.KeyToken)
	if err != nil {
		return "", errors.New("не выполнен вход — запустите команду login")
	}
	return token, nil
}

// saveAuth сохраняет токен, соль и производный ключ шифрования в keyring.
func (a *app) saveAuth(token string, salt []byte, masterPassword string) error {
	if err := a.keyring.Set(clientkeyring.KeyToken, token); err != nil {
		return err
	}
	if err := a.keyring.Set(clientkeyring.KeySalt, hex.EncodeToString(salt)); err != nil {
		return err
	}
	key, err := crypto.DeriveKey(masterPassword, salt)
	if err != nil {
		return err
	}
	return a.keyring.Set(clientkeyring.KeyMasterKey, hex.EncodeToString(key[:]))
}

// loadSalt читает соль KDF из keyring.
func (a *app) loadSalt() ([]byte, error) {
	s, err := a.keyring.Get(clientkeyring.KeySalt)
	if err != nil {
		return nil, errors.New("соль не найдена — выполните команду login")
	}
	return hex.DecodeString(s)
}

// masterKey возвращает производный ключ шифрования из keyring; при отсутствии
// выводит его заново из мастер-пароля, запрошенного у пользователя.
func (a *app) masterKey() (crypto.Key, error) {
	if v, err := a.keyring.Get(clientkeyring.KeyMasterKey); err == nil && v != "" {
		raw, err := hex.DecodeString(v)
		if err != nil {
			return crypto.Key{}, err
		}
		if len(raw) != crypto.KeySize {
			return crypto.Key{}, errors.New("неверный размер сохранённого ключа")
		}
		var key crypto.Key
		copy(key[:], raw)
		return key, nil
	}

	salt, err := a.loadSalt()
	if err != nil {
		return crypto.Key{}, err
	}
	password, err := promptSecret("мастер-пароль: ")
	if err != nil {
		return crypto.Key{}, err
	}
	if password == "" {
		return crypto.Key{}, errors.New("мастер-пароль не может быть пустым")
	}
	key, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return crypto.Key{}, err
	}
	if err := a.keyring.Set(clientkeyring.KeyMasterKey, hex.EncodeToString(key[:])); err != nil {
		return crypto.Key{}, err
	}
	return key, nil
}

// clearAuth удаляет сохранённые токен, соль и ключ (выход из сессии).
func (a *app) clearAuth() error {
	if err := a.keyring.Delete(clientkeyring.KeyToken); err != nil {
		return err
	}
	if err := a.keyring.Delete(clientkeyring.KeySalt); err != nil {
		return err
	}
	return a.keyring.Delete(clientkeyring.KeyMasterKey)
}

// ctx возвращает фоновый контекст для запросов.
func ctx() context.Context {
	return context.Background()
}

// prompt выводит приглашение и читает строку из stdin.
func prompt(p string) (string, error) {
	fmt.Fprint(os.Stdout, p)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// promptSecret читает строку из stdin без эха (для паролей).
func promptSecret(p string) (string, error) {
	fmt.Fprint(os.Stdout, p)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stdout)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

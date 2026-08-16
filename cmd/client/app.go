package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/term"

	"github.com/victor/gophkeeper/internal/client/api"
	clientconfig "github.com/victor/gophkeeper/internal/client/config"
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
	if a.sessionExpired() {
		_ = a.clearAuth()
		return "", errors.New("сессия истекла — выполните команду login")
	}
	return token, nil
}

// sessionExpired возвращает true, если JWT-токен истёк.
func (a *app) sessionExpired() bool {
	v, err := a.keyring.Get(clientkeyring.KeyTokenExpiresAt)
	if err != nil {
		return false
	}
	exp, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() > exp
}

// saveAuth сохраняет токен, соль, производный ключ и метки времени в keyring.
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
	if err := a.keyring.Set(clientkeyring.KeyMasterKey, hex.EncodeToString(key[:])); err != nil {
		return err
	}
	if err := a.keyring.Set(clientkeyring.KeyMasterKeyCreatedAt, strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
		return err
	}

	exp, err := tokenExpiresAt(token)
	if err != nil {
		return err
	}
	return a.keyring.Set(clientkeyring.KeyTokenExpiresAt, strconv.FormatInt(exp, 10))
}

// loadSalt читает соль KDF из keyring.
func (a *app) loadSalt() ([]byte, error) {
	s, err := a.keyring.Get(clientkeyring.KeySalt)
	if err != nil {
		return nil, errors.New("соль не найдена — выполните команду login")
	}
	return hex.DecodeString(s)
}

// masterKey возвращает производный ключ шифрования: из keyring, если он свежий,
// иначе выводит его заново из мастер-пароля, запрошенного у пользователя.
func (a *app) masterKey() (crypto.Key, error) {
	if key, ok, err := a.cachedMasterKey(); err != nil {
		return crypto.Key{}, err
	} else if ok {
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
	if err := a.keyring.Set(clientkeyring.KeyMasterKeyCreatedAt, strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
		return crypto.Key{}, err
	}
	return key, nil
}

// cachedMasterKey возвращает свежий ключ из keyring; (nil, false), если его нет
// или он протух (в этом случае протухший ключ удаляется).
func (a *app) cachedMasterKey() (crypto.Key, bool, error) {
	v, err := a.keyring.Get(clientkeyring.KeyMasterKey)
	if err != nil || v == "" {
		return crypto.Key{}, false, nil
	}
	raw, err := hex.DecodeString(v)
	if err != nil {
		return crypto.Key{}, false, err
	}
	if len(raw) != crypto.KeySize {
		return crypto.Key{}, false, errors.New("неверный размер сохранённого ключа")
	}
	if a.masterKeyExpired() {
		_ = a.keyring.Delete(clientkeyring.KeyMasterKey)
		_ = a.keyring.Delete(clientkeyring.KeyMasterKeyCreatedAt)
		return crypto.Key{}, false, nil
	}
	var key crypto.Key
	copy(key[:], raw)
	return key, true, nil
}

// masterKeyExpired возвращает true, если производный ключ протух.
func (a *app) masterKeyExpired() bool {
	ttl, err := clientconfig.MasterKeyTTL(ctx(), a.store)
	if err != nil {
		return false
	}
	if ttl == 0 {
		return false
	}
	createdStr, err := a.keyring.Get(clientkeyring.KeyMasterKeyCreatedAt)
	if err != nil {
		return false
	}
	createdUnix, err := strconv.ParseInt(createdStr, 10, 64)
	if err != nil {
		return false
	}
	return clientconfig.IsExpired(time.Unix(createdUnix, 0), time.Now(), ttl)
}

// clearAuth удаляет сохранённые токен, соль, ключ и метки (выход из сессии).
func (a *app) clearAuth() error {
	for _, k := range []string{
		clientkeyring.KeyToken,
		clientkeyring.KeySalt,
		clientkeyring.KeyMasterKey,
		clientkeyring.KeyMasterKeyCreatedAt,
		clientkeyring.KeyTokenExpiresAt,
	} {
		if err := a.keyring.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

// tokenExpiresAt возвращает время истечения JWT-токена в unix-секундах.
func tokenExpiresAt(token string) (int64, error) {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		return 0, err
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return 0, errors.New("токен не содержит поле exp")
	}
	return int64(exp), nil
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

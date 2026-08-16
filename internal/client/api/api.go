// Package api реализует HTTP-клиент к серверу GophKeeper.
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/victor/gophkeeper/internal/model"
)

// Client — HTTP-клиент сервера.
type Client struct {
	baseURL string
	http    *http.Client
}

// Option — функция настройки клиента.
type Option func(*Client) error

// New создаёт клиент сервера по базовому URL и применяет переданные опции.
func New(baseURL string, opts ...Option) (*Client, error) {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// WithCACertFile добавляет сертификат из PEM-файла в пул корневых CA клиента.
// Используется для доверия самоподписным сертификатам сервера.
func WithCACertFile(path string) Option {
	return func(c *Client) error {
		pem, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("не удалось прочитать CA-сертификат: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return errors.New("не удалось разобрать CA-сертификат")
		}
		c.http.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
		}
		return nil
	}
}

// WithInsecure отключает проверку TLS-сертификата сервера.
// Использовать только в dev-окружении.
func WithInsecure() Option {
	return func(c *Client) error {
		c.http.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return nil
	}
}

// AuthResponse — ответ сервера на регистрацию/вход.
type AuthResponse struct {
	// Token — JWT-токен доступа.
	Token string `json:"token"`
	// Salt — соль KDF для вывода мастер-ключа (base64).
	Salt []byte `json:"salt"`
}

// CreateObjectRequest — запрос на создание/обновление объекта.
type CreateObjectRequest struct {
	// Name — имя объекта.
	Name string `json:"name"`
	// Type — тип объекта.
	Type model.SecretType `json:"type"`
	// Salt — соль KDF (base64).
	Salt []byte `json:"salt"`
	// Ciphertext — зашифрованные данные (base64).
	Ciphertext []byte `json:"ciphertext"`
}

// Error — ошибка, возвращаемая сервером.
type Error struct {
	// StatusCode — HTTP-код ответа.
	StatusCode int
	// Message — сообщение об ошибке.
	Message string
}

// Error возвращает сообщение об ошибке.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("api: HTTP %d", e.StatusCode)
}

// Register регистрирует нового пользователя.
func (c *Client) Register(ctx context.Context, login, password string) (*AuthResponse, error) {
	return c.auth(ctx, "/api/v1/register", login, password)
}

// Login аутентифицирует пользователя.
func (c *Client) Login(ctx context.Context, login, password string) (*AuthResponse, error) {
	return c.auth(ctx, "/api/v1/login", login, password)
}

// CreateObject создаёт объект на сервере.
func (c *Client) CreateObject(ctx context.Context, token string, req CreateObjectRequest) (*model.Object, error) {
	data, err := c.do(ctx, http.MethodPost, "/api/v1/objects", token, req)
	if err != nil {
		return nil, err
	}
	var obj model.Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// ListObjects возвращает все объекты пользователя.
func (c *Client) ListObjects(ctx context.Context, token string) ([]*model.Object, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/objects", token, nil)
	if err != nil {
		return nil, err
	}
	objects := []*model.Object{}
	if err := json.Unmarshal(data, &objects); err != nil {
		return nil, err
	}
	return objects, nil
}

// GetObject возвращает объект по идентификатору.
func (c *Client) GetObject(ctx context.Context, token, id string) (*model.Object, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/objects/"+id, token, nil)
	if err != nil {
		return nil, err
	}
	var obj model.Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// UpdateObject обновляет объект на сервере.
func (c *Client) UpdateObject(ctx context.Context, token, id string, req CreateObjectRequest) (*model.Object, error) {
	data, err := c.do(ctx, http.MethodPut, "/api/v1/objects/"+id, token, req)
	if err != nil {
		return nil, err
	}
	var obj model.Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// DeleteObject удаляет объект на сервере.
func (c *Client) DeleteObject(ctx context.Context, token, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/objects/"+id, token, nil)
	return err
}

// auth выполняет регистрацию или вход.
func (c *Client) auth(ctx context.Context, path, login, password string) (*AuthResponse, error) {
	body := map[string]string{"login": login, "password": password}
	data, err := c.do(ctx, http.MethodPost, path, "", body)
	if err != nil {
		return nil, err
	}
	var resp AuthResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// do выполняет HTTP-запрос и возвращает тело ответа.
func (c *Client) do(ctx context.Context, method, path, token string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, decodeError(data, resp.StatusCode)
	}
	return data, nil
}

// decodeError разбирает JSON-ошибку сервера.
func decodeError(data []byte, status int) error {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || payload.Error == "" {
		return &Error{StatusCode: status, Message: fmt.Sprintf("HTTP %d", status)}
	}
	return &Error{StatusCode: status, Message: payload.Error}
}

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
	baseURL   string
	transport *http.Transport
	http      *http.Client
	file      *http.Client
}

// Option — функция настройки клиента.
type Option func(*Client) error

// New создаёт клиент сервера по базовому URL и применяет переданные опции.
func New(baseURL string, opts ...Option) (*Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	c := &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		transport: transport,
		http:      &http.Client{Timeout: 15 * time.Second, Transport: transport},
		file:      &http.Client{Transport: transport},
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
		c.transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
		return nil
	}
}

// WithInsecure отключает проверку TLS-сертификата сервера.
// Использовать только в dev-окружении.
func WithInsecure() Option {
	return func(c *Client) error {
		c.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
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
// Deprecated: используйте ListObjectsPage для пагинации.
func (c *Client) ListObjects(ctx context.Context, token string) ([]*model.Object, error) {
	objects := []*model.Object{}
	page := 1
	for {
		resp, err := c.ListObjectsPage(ctx, token, page, 100)
		if err != nil {
			return nil, err
		}
		objects = append(objects, resp.Data...)
		if page >= resp.Metadata.Pages || len(resp.Data) == 0 {
			break
		}
		page++
	}
	return objects, nil
}

// ObjectsPage — страница объектов (JSON-API).
type ObjectsPage struct {
	Data     []*model.Object `json:"data"`
	Metadata PageMetadata    `json:"metadata"`
	Links    PageLinks       `json:"links"`
}

// PageMetadata — метаданные пагинации.
type PageMetadata struct {
	Total      int `json:"total"`
	Pages      int `json:"pages"`
	PageSize   int `json:"page_size"`
	PageNumber int `json:"page_number"`
}

// PageLinks — ссылки пагинации.
type PageLinks struct {
	First string  `json:"first"`
	Last  string  `json:"last"`
	Prev  *string `json:"prev"`
	Next  *string `json:"next"`
}

// ListObjectsPage возвращает страницу объектов пользователя.
func (c *Client) ListObjectsPage(ctx context.Context, token string, page, pageSize int) (*ObjectsPage, error) {
	path := fmt.Sprintf("/api/v1/objects?page[number]=%d&page[size]=%d", page, pageSize)
	data, err := c.do(ctx, http.MethodGet, path, token, nil)
	if err != nil {
		return nil, err
	}
	var resp ObjectsPage
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListMetadata возвращает метаданные объекта.
func (c *Client) ListMetadata(ctx context.Context, token, objectID string) ([]*model.Metadata, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/objects/"+objectID+"/metadata", token, nil)
	if err != nil {
		return nil, err
	}
	metadata := []*model.Metadata{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
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

// UploadFile передаёт бинарный файл на сервер потоком. body читается до EOF,
// size — известный размер тела (для Content-Length).
func (c *Client) UploadFile(ctx context.Context, token, id string, body io.Reader, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/api/v1/files/"+id, body)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.file.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return decodeError(data, resp.StatusCode)
	}
	return nil
}

// DownloadFile скачивает бинарный файл с сервера потоком.
func (c *Client) DownloadFile(ctx context.Context, token, id string) (io.ReadCloser, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/files/"+id, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.file.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, 0, decodeError(data, resp.StatusCode)
	}
	return resp.Body, resp.ContentLength, nil
}

// VersionInfo — информация о сборке сервера.
type VersionInfo struct {
	Version     string `json:"version"`
	BuildDate   string `json:"build_date"`
	BuildCommit string `json:"build_commit"`
}

// ServerVersion возвращает версию сервера.
func (c *Client) ServerVersion(ctx context.Context) (*VersionInfo, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/version", "", nil)
	if err != nil {
		return nil, err
	}
	var v VersionInfo
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// Stats — статистика пользователя.
type Stats struct {
	ObjectsCount   int   `json:"objects_count"`
	FilesCount     int   `json:"files_count"`
	FilesTotalSize int64 `json:"files_total_size"`
}

// Stats возвращает статистику пользователя.
func (c *Client) Stats(ctx context.Context, token string) (*Stats, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/stats", token, nil)
	if err != nil {
		return nil, err
	}
	var s Stats
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
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

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/victor/gophkeeper/internal/model"
)

func TestClient(t *testing.T) {
	tests := map[string]struct {
		handler http.HandlerFunc
		act     func(c *Client) error
	}{
		"register возвращает токен и соль": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Login    string `json:"login"`
					Password string `json:"password"`
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req.Login != "bob" || req.Password != "hunter2" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"token": "tok", "salt": "c2FsdA=="})
			},
			act: func(c *Client) error {
				resp, err := c.Register(context.Background(), "bob", "hunter2")
				if err != nil {
					return err
				}
				if resp.Token != "tok" {
					return fmt.Errorf("Token: got %q, want tok", resp.Token)
				}
				if string(resp.Salt) != "salt" {
					return fmt.Errorf("Salt: got %q, want salt", resp.Salt)
				}
				return nil
			},
		},
		"create object отправляет заголовок Authorization": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer my-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				writeJSON(w, http.StatusOK, model.Object{ID: "id-1", Name: "test", Type: model.SecretTypeText})
			},
			act: func(c *Client) error {
				obj, err := c.CreateObject(context.Background(), "my-token", CreateObjectRequest{
					Name:       "test",
					Type:       model.SecretTypeText,
					Salt:       []byte("0123456789abcdef"),
					Ciphertext: []byte("cipher"),
				})
				if err != nil {
					return err
				}
				if obj.ID != "id-1" {
					return fmt.Errorf("ID: got %q, want id-1", obj.ID)
				}
				return nil
			},
		},
		"list objects возвращает список": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, []*model.Object{
					{ID: "a", Name: "one"},
					{ID: "b", Name: "two"},
				})
			},
			act: func(c *Client) error {
				objects, err := c.ListObjects(context.Background(), "tok")
				if err != nil {
					return err
				}
				if len(objects) != 2 {
					return fmt.Errorf("got %d, want 2", len(objects))
				}
				return nil
			},
		},
		"ошибка сервера превращается в *Error": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "объект не найден"})
			},
			act: func(c *Client) error {
				_, err := c.GetObject(context.Background(), "tok", "missing")
				if err == nil {
					return fmt.Errorf("ожидалась ошибка")
				}
				var apiErr *Error
				if !errors.As(err, &apiErr) {
					return fmt.Errorf("ожидалась *api.Error, got %T", err)
				}
				if apiErr.StatusCode != http.StatusNotFound {
					return fmt.Errorf("StatusCode: got %d, want 404", apiErr.StatusCode)
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			if err := tc.act(New(srv.URL)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

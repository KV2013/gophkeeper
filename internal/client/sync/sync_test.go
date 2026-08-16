package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/victor/gophkeeper/internal/client/api"
	"github.com/victor/gophkeeper/internal/client/repository"
	"github.com/victor/gophkeeper/internal/model"
)

func openStore(t *testing.T) *repository.Repository {
	t.Helper()
	s, err := repository.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("repository.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSyncer(t *testing.T) {
	tests := map[string]struct {
		handler http.HandlerFunc
		act     func(s *Syncer, store *repository.Repository) error
	}{
		"pull заменяет локальный кэш": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, []*model.Object{
					{ID: "a", Name: "one", Type: model.SecretTypeText, Salt: []byte("0123456789abcdef"), Ciphertext: []byte("c1")},
					{ID: "b", Name: "two", Type: model.SecretTypeCard, Salt: []byte("0123456789abcdef"), Ciphertext: []byte("c2")},
				})
			},
			act: func(s *Syncer, store *repository.Repository) error {
				if err := s.Pull(context.Background(), "tok"); err != nil {
					return err
				}
				list, err := store.ListObjects(context.Background())
				if err != nil {
					return err
				}
				if len(list) != 2 {
					return fmt.Errorf("got %d, want 2", len(list))
				}
				return nil
			},
		},
		"create object пишет на сервер и в кэш": {
			handler: func(w http.ResponseWriter, r *http.Request) {
				var req api.CreateObjectRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				writeJSON(w, http.StatusOK, model.Object{
					ID:         "id-1",
					Name:       req.Name,
					Type:       req.Type,
					Salt:       req.Salt,
					Ciphertext: req.Ciphertext,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				})
			},
			act: func(s *Syncer, store *repository.Repository) error {
				obj, err := s.CreateObject(context.Background(), "tok", api.CreateObjectRequest{
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
				if _, err := store.GetObject(context.Background(), "id-1"); err != nil {
					return fmt.Errorf("объект должен быть записан в кэш: %v", err)
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			store := openStore(t)
			s := New(api.New(srv.URL), store)

			if err := tc.act(s, store); err != nil {
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

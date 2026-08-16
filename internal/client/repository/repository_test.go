package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/victor/gophkeeper/internal/model"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testObject(id, name string) *model.Object {
	now := time.Now().UTC()
	return &model.Object{
		ID:         id,
		Name:       name,
		Type:       model.SecretTypeText,
		Salt:       []byte("0123456789abcdef"),
		Ciphertext: []byte("cipher-" + id),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestStorage(t *testing.T) {
	tests := map[string]struct {
		fn      func(s *Repository) error
		wantErr error
	}{
		"upsert и get возвращают объект": {
			fn: func(s *Repository) error {
				ctx := context.Background()
				if err := s.UpsertObject(ctx, testObject("id-1", "first")); err != nil {
					return err
				}
				got, err := s.GetObject(ctx, "id-1")
				if err != nil {
					return err
				}
				if got.Name != "first" || got.Type != model.SecretTypeText {
					return fmt.Errorf("got name=%q type=%q", got.Name, got.Type)
				}
				if string(got.Ciphertext) != "cipher-id-1" {
					return fmt.Errorf("got ciphertext %q", got.Ciphertext)
				}
				return nil
			},
		},
		"upsert обновляет существующий объект": {
			fn: func(s *Repository) error {
				ctx := context.Background()
				obj := testObject("id-1", "first")
				if err := s.UpsertObject(ctx, obj); err != nil {
					return err
				}
				obj.Name = "renamed"
				obj.Ciphertext = []byte("new-cipher")
				if err := s.UpsertObject(ctx, obj); err != nil {
					return err
				}
				got, err := s.GetObject(ctx, "id-1")
				if err != nil {
					return err
				}
				if got.Name != "renamed" || string(got.Ciphertext) != "new-cipher" {
					return fmt.Errorf("got name=%q ciphertext=%q", got.Name, got.Ciphertext)
				}
				return nil
			},
		},
		"get отсутствующего объекта": {
			fn: func(s *Repository) error {
				_, err := s.GetObject(context.Background(), "missing")
				return err
			},
			wantErr: ErrNotFound,
		},
		"list сортирует объекты по имени": {
			fn: func(s *Repository) error {
				ctx := context.Background()
				for _, o := range []*model.Object{
					testObject("id-2", "beta"),
					testObject("id-1", "alpha"),
					testObject("id-3", "gamma"),
				} {
					if err := s.UpsertObject(ctx, o); err != nil {
						return err
					}
				}
				list, err := s.ListObjects(ctx)
				if err != nil {
					return err
				}
				if len(list) != 3 {
					return fmt.Errorf("got %d, want 3", len(list))
				}
				for i, want := range []string{"alpha", "beta", "gamma"} {
					if list[i].Name != want {
						return fmt.Errorf("позиция %d: got %q, want %q", i, list[i].Name, want)
					}
				}
				return nil
			},
		},
		"delete удаляет объект": {
			fn: func(s *Repository) error {
				ctx := context.Background()
				if err := s.UpsertObject(ctx, testObject("id-1", "first")); err != nil {
					return err
				}
				if err := s.DeleteObject(ctx, "id-1"); err != nil {
					return err
				}
				return s.DeleteObject(ctx, "id-1")
			},
			wantErr: ErrNotFound,
		},
		"replace all полностью заменяет кэш": {
			fn: func(s *Repository) error {
				ctx := context.Background()
				if err := s.UpsertObject(ctx, testObject("old-1", "old")); err != nil {
					return err
				}
				replacement := []*model.Object{
					testObject("new-1", "one"),
					testObject("new-2", "two"),
				}
				if err := s.ReplaceAll(ctx, replacement); err != nil {
					return err
				}
				list, err := s.ListObjects(ctx)
				if err != nil {
					return err
				}
				if len(list) != 2 {
					return fmt.Errorf("got %d, want 2", len(list))
				}
				if _, err := s.GetObject(ctx, "old-1"); !errors.Is(err, ErrNotFound) {
					return fmt.Errorf("ожидалась ErrNotFound для старого объекта, got %v", err)
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := newTestRepository(t)
			err := tc.fn(s)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("ожидалась ошибка %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
		})
	}
}

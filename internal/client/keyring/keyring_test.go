package keyring

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewFile("gophkeeper-test", filepath.Join(t.TempDir(), "credentials.json"))
}

func TestStore(t *testing.T) {
	tests := map[string]struct {
		fn      func(s *Store) error
		wantErr error
	}{
		"Set и Get возвращают значение": {
			fn: func(s *Store) error {
				if err := s.Set(KeyToken, "abc123"); err != nil {
					return err
				}
				v, err := s.Get(KeyToken)
				if err != nil {
					return err
				}
				if v != "abc123" {
					return fmt.Errorf("got %q, want abc123", v)
				}
				return nil
			},
		},
		"Get отсутствующего значения": {
			fn: func(s *Store) error {
				_, err := s.Get(KeyToken)
				return err
			},
			wantErr: ErrNotFound,
		},
		"Delete удаляет значение": {
			fn: func(s *Store) error {
				if err := s.Set(KeyToken, "abc"); err != nil {
					return err
				}
				if err := s.Delete(KeyToken); err != nil {
					return err
				}
				_, err := s.Get(KeyToken)
				return err
			},
			wantErr: ErrNotFound,
		},
		"файл-фолбэк создаётся с правами 0600": {
			fn: func(s *Store) error {
				if err := s.Set(KeyToken, "abc"); err != nil {
					return err
				}
				info, err := os.Stat(s.file)
				if err != nil {
					return err
				}
				if perm := info.Mode().Perm(); perm != 0o600 {
					return fmt.Errorf("права файла: got %o, want 600", perm)
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
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

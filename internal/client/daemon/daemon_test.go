package daemon

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/victor/gophkeeper/internal/client/config"
	"github.com/victor/gophkeeper/internal/client/keyring"
	"github.com/victor/gophkeeper/internal/client/repository"
)

func TestCleanup(t *testing.T) {
	tests := map[string]struct {
		setup func(ks *keyring.Store, repo *repository.Repository)
		check func(ks *keyring.Store) error
	}{
		"протухший ключ удаляется": {
			setup: func(ks *keyring.Store, repo *repository.Repository) {
				_ = ks.Set(keyring.KeyMasterKey, "aabb")
				_ = ks.Set(keyring.KeyMasterKeyCreatedAt, strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10))
				_ = repo.SetConfig(context.Background(), config.KeyMasterKeyTTL, "5m")
			},
			check: func(ks *keyring.Store) error {
				if _, err := ks.Get(keyring.KeyMasterKey); err == nil {
					return fmt.Errorf("ключ должен быть удалён")
				}
				return nil
			},
		},
		"свежий ключ остаётся": {
			setup: func(ks *keyring.Store, repo *repository.Repository) {
				_ = ks.Set(keyring.KeyMasterKey, "aabb")
				_ = ks.Set(keyring.KeyMasterKeyCreatedAt, strconv.FormatInt(time.Now().Unix(), 10))
				_ = repo.SetConfig(context.Background(), config.KeyMasterKeyTTL, "5m")
			},
			check: func(ks *keyring.Store) error {
				if v, err := ks.Get(keyring.KeyMasterKey); err != nil || v != "aabb" {
					return fmt.Errorf("ключ должен остаться, err=%v", err)
				}
				return nil
			},
		},
		"ttl 0 ничего не удаляет": {
			setup: func(ks *keyring.Store, repo *repository.Repository) {
				_ = ks.Set(keyring.KeyMasterKey, "aabb")
				_ = ks.Set(keyring.KeyMasterKeyCreatedAt, strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10))
				_ = repo.SetConfig(context.Background(), config.KeyMasterKeyTTL, "0")
			},
			check: func(ks *keyring.Store) error {
				if v, err := ks.Get(keyring.KeyMasterKey); err != nil || v != "aabb" {
					return fmt.Errorf("ключ должен остаться, err=%v", err)
				}
				return nil
			},
		},
		"протухшая сессия удаляет всё": {
			setup: func(ks *keyring.Store, repo *repository.Repository) {
				_ = ks.Set(keyring.KeyToken, "token")
				_ = ks.Set(keyring.KeyTokenExpiresAt, strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10))
				_ = ks.Set(keyring.KeyMasterKey, "aabb")
				_ = ks.Set(keyring.KeyMasterKeyCreatedAt, strconv.FormatInt(time.Now().Unix(), 10))
			},
			check: func(ks *keyring.Store) error {
				if _, err := ks.Get(keyring.KeyToken); err == nil {
					return fmt.Errorf("токен должен быть удалён")
				}
				if _, err := ks.Get(keyring.KeyMasterKey); err == nil {
					return fmt.Errorf("ключ должен быть удалён")
				}
				return nil
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ks := keyring.NewFile("test", filepath.Join(t.TempDir(), "credentials.json"))
			repo, err := repository.New(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("repository.New: %v", err)
			}
			defer repo.Close()

			tc.setup(ks, repo)

			d := New(ks, repo, 0)
			if err := d.Cleanup(context.Background()); err != nil {
				t.Fatalf("Cleanup: %v", err)
			}
			if err := tc.check(ks); err != nil {
				t.Fatal(err)
			}
		})
	}
}

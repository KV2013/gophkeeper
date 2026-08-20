// Package daemon реализует фоновый процесс клиента, который следит за
// протухшими производными ключами и сессиями и удаляет их из keyring.
package daemon

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/victor/gophkeeper/internal/client/config"
	"github.com/victor/gophkeeper/internal/client/keyring"
	"github.com/victor/gophkeeper/internal/client/repository"
)

// DefaultPollInterval — интервал опроса по умолчанию.
const DefaultPollInterval = 10 * time.Second

// Daemon — фоновый процесс очистки протухших секретов.
type Daemon struct {
	keyring  *keyring.Store
	repo     *repository.Repository
	interval time.Duration
	logger   *log.Logger
}

// New создаёт демон. Интервал <= 0 заменяется на DefaultPollInterval.
func New(ks *keyring.Store, repo *repository.Repository, interval time.Duration) *Daemon {
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	return &Daemon{
		keyring:  ks,
		repo:     repo,
		interval: interval,
		logger:   log.Default(),
	}
}

// Run запускает цикл очистки до отмены контекста.
func (d *Daemon) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := d.Cleanup(ctx); err != nil {
				d.logger.Printf("ошибка очистки: %v", err)
			}
		}
	}
}

// Cleanup выполняет один проход: удаляет протухший ключ и протухшую сессию.
func (d *Daemon) Cleanup(ctx context.Context) error {
	if err := d.cleanupKey(ctx); err != nil {
		return err
	}
	return d.cleanupSession()
}

// cleanupKey удаляет производный ключ, если он прожил дольше TTL.
func (d *Daemon) cleanupKey(ctx context.Context) error {
	ttl, err := config.MasterKeyTTL(ctx, d.repo)
	if err != nil {
		return err
	}
	if ttl == 0 {
		return nil
	}

	createdStr, err := d.keyring.Get(keyring.KeyMasterKeyCreatedAt)
	if err != nil {
		return nil
	}
	createdUnix, err := strconv.ParseInt(createdStr, 10, 64)
	if err != nil {
		return nil
	}

	if config.IsExpired(time.Unix(createdUnix, 0), time.Now(), ttl) {
		d.delete(keyring.KeyMasterKey)
		d.delete(keyring.KeyMasterKeyCreatedAt)
	}
	return nil
}

// cleanupSession удаляет все секреты сессии, если JWT-токен истёк.
func (d *Daemon) cleanupSession() error {
	v, err := d.keyring.Get(keyring.KeyTokenExpiresAt)
	if err != nil {
		return nil
	}
	exp, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	if time.Now().Unix() <= exp {
		return nil
	}

	for _, k := range []string{
		keyring.KeyToken,
		keyring.KeySalt,
		keyring.KeyMasterKey,
		keyring.KeyMasterKeyCreatedAt,
		keyring.KeyTokenExpiresAt,
	} {
		d.delete(k)
	}
	return nil
}

// delete удаляет ключ из keyring, логируя ошибку при неудаче.
func (d *Daemon) delete(key string) {
	if err := d.keyring.Delete(key); err != nil {
		d.logger.Printf("не удалось удалить ключ %q из keyring: %v", key, err)
	}
}

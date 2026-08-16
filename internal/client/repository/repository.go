// Package repository реализует локальный SQLite-кэш клиента.
//
// Кэш хранит объекты пользователя: содержимое — E2E-шифротекст, а name и type
// — открытым текстом для офлайн-поиска. Сервер остаётся источником истины.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	sqlitemigrate "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"

	"github.com/victor/gophkeeper/internal/client/migrations"
	"github.com/victor/gophkeeper/internal/model"
)

// ErrNotFound — объект не найден в кэше.
var ErrNotFound = errors.New("repository: объект не найден")

// Repository — локальное хранилище объектов.
type Repository struct {
	db *sql.DB
}

// New открывает (или создаёт) SQLite-кэш по пути dbPath и применяет миграции.
func New(dbPath string) (*Repository, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, err
	}

	dsn := "file:" + filepath.ToSlash(dbPath) + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	repo := &Repository{db: db}
	if err := repo.runMigrations(); err != nil {
		db.Close()
		return nil, err
	}
	return repo, nil
}

// runMigrations применяет SQL-миграции из встроенного пакета migrations.
func (r *Repository) runMigrations() error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("не удалось инициализировать источник миграций: %w", err)
	}

	driver, err := sqlitemigrate.WithInstance(r.db, &sqlitemigrate.Config{})
	if err != nil {
		return fmt.Errorf("не удалось создать драйвер миграций: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("не удалось инициализировать migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("не удалось применить миграции: %w", err)
	}
	return nil
}

// Close закрывает соединение с БД.
func (s *Repository) Close() error {
	return s.db.Close()
}

// GetConfig возвращает значение конфигурации по ключу.
func (s *Repository) GetConfig(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM client_config WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetConfig сохраняет (или обновляет) значение конфигурации по ключу.
func (s *Repository) SetConfig(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO client_config (key, value) VALUES ($1, $2)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// UpsertObject вставляет или обновляет объект в кэше.
func (s *Repository) UpsertObject(ctx context.Context, o *model.Object) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO objects (id, name, type, salt, ciphertext, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			salt = excluded.salt,
			ciphertext = excluded.ciphertext,
			updated_at = excluded.updated_at`,
		o.ID, o.Name, string(o.Type), o.Salt, o.Ciphertext, o.CreatedAt.Unix(), o.UpdatedAt.Unix(),
	)
	return err
}

// GetObject возвращает объект из кэша по идентификатору.
func (s *Repository) GetObject(ctx context.Context, id string) (*model.Object, error) {
	o := &model.Object{}
	var typ string
	var created, updated int64

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, salt, ciphertext, created_at, updated_at
		FROM objects WHERE id = $1`, id,
	).Scan(&o.ID, &o.Name, &typ, &o.Salt, &o.Ciphertext, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	o.Type = model.SecretType(typ)
	o.CreatedAt = time.Unix(created, 0).UTC()
	o.UpdatedAt = time.Unix(updated, 0).UTC()
	return o, nil
}

// ListObjects возвращает все объекты из кэша, отсортированные по имени.
func (s *Repository) ListObjects(ctx context.Context) ([]*model.Object, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, salt, ciphertext, created_at, updated_at
		FROM objects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	objects := []*model.Object{}
	for rows.Next() {
		o := &model.Object{}
		var typ string
		var created, updated int64
		if err := rows.Scan(&o.ID, &o.Name, &typ, &o.Salt, &o.Ciphertext, &created, &updated); err != nil {
			return nil, err
		}
		o.Type = model.SecretType(typ)
		o.CreatedAt = time.Unix(created, 0).UTC()
		o.UpdatedAt = time.Unix(updated, 0).UTC()
		objects = append(objects, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return objects, nil
}

// DeleteObject удаляет объект из кэша.
func (s *Repository) DeleteObject(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM objects WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceAll заменяет всё содержимое кэша на переданный список объектов
// (полная синхронизация с сервером).
func (s *Repository) ReplaceAll(ctx context.Context, objects []*model.Object) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM objects`); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO objects (id, name, type, salt, ciphertext, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, o := range objects {
		if _, err := stmt.ExecContext(ctx, o.ID, o.Name, string(o.Type), o.Salt, o.Ciphertext, o.CreatedAt.Unix(), o.UpdatedAt.Unix()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

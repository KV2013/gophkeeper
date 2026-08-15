// Package sqlx содержит реализацию интерфейса repository.Repository на базе
// PostgreSQL через библиотеку sqlx.
package sqlx

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/model"
	repoerrors "github.com/victor/gophkeeper/internal/repository/errors"
)

// Repository — реализация хранилища на PostgreSQL.
type Repository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// New подключается к PostgreSQL и применяет схему БД.
func New(dsn string, logger *zap.Logger) (*Repository, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	repo := &Repository{db: db, logger: logger}
	if err := repo.runMigrations(); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *Repository) runMigrations() error {
	driver, err := migratepgx.WithInstance(r.db.DB, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("не удалось создать драйвер миграций: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("не удалось инициализировать migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("не удалось применить миграции: %w", err)
	}

	r.logger.Debug("Миграции обработаны")

	return nil
}

// Close закрывает соединение с БД.
func (r *Repository) Close() error {
	return r.db.Close()
}

// CreateUser создаёт нового пользователя.
func (r *Repository) CreateUser(ctx context.Context, u *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, login, password_hash, salt, created_at) VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Login, u.PasswordHash, u.Salt, u.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repoerrors.ErrLoginExists
		}
		return err
	}
	return nil
}

// GetUserByLogin возвращает пользователя по логину.
func (r *Repository) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, login, password_hash, salt, created_at FROM users WHERE login = $1`, login,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateObject сохраняет новый объект.
func (r *Repository) CreateObject(ctx context.Context, obj *model.Object) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO objects (id, user_id, type, salt, ciphertext, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		obj.ID, obj.UserID, obj.Type, obj.Salt, obj.Ciphertext, obj.CreatedAt, obj.UpdatedAt,
	)
	return err
}

// GetObject возвращает объект по идентификатору.
func (r *Repository) GetObject(ctx context.Context, userID, id string) (*model.Object, error) {
	var o model.Object
	err := r.db.GetContext(ctx, &o,
		`SELECT id, user_id, type, salt, ciphertext, created_at, updated_at
		 FROM objects WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ListObjects возвращает все объекты пользователя.
func (r *Repository) ListObjects(ctx context.Context, userID string) ([]*model.Object, error) {
	objects := []*model.Object{}
	err := r.db.SelectContext(ctx, &objects,
		`SELECT id, user_id, type, salt, ciphertext, created_at, updated_at
		 FROM objects WHERE user_id = $1 ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, err
	}
	return objects, nil
}

// UpdateObject обновляет объект.
func (r *Repository) UpdateObject(ctx context.Context, obj *model.Object) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE objects SET type = $1, salt = $2, ciphertext = $3, updated_at = $4
		 WHERE id = $5 AND user_id = $6`,
		obj.Type, obj.Salt, obj.Ciphertext, obj.UpdatedAt, obj.ID, obj.UserID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

// DeleteObject удаляет объект.
func (r *Repository) DeleteObject(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM objects WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

// CreateMetadata сохраняет новую запись метаданных.
func (r *Repository) CreateMetadata(ctx context.Context, m *model.Metadata) error {
	options, err := json.Marshal(m.Options)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO metadata (id, user_id, object_id, name, order_number, options)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		m.ID, m.UserID, m.ObjectID, m.Name, m.OrderNumber, string(options),
	)
	return err
}

// GetMetadata возвращает запись метаданных.
func (r *Repository) GetMetadata(ctx context.Context, userID, objectID, metaID string) (*model.Metadata, error) {
	var row metadataRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, user_id, object_id, name, order_number, options
		 FROM metadata WHERE id = $1 AND user_id = $2 AND object_id = $3`,
		metaID, userID, objectID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toModel()
}

// ListMetadata возвращает все записи метаданных объекта.
func (r *Repository) ListMetadata(ctx context.Context, userID, objectID string) ([]*model.Metadata, error) {
	rows := []metadataRow{}
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, user_id, object_id, name, order_number, options
		 FROM metadata WHERE user_id = $1 AND object_id = $2 ORDER BY order_number`,
		userID, objectID,
	)
	if err != nil {
		return nil, err
	}
	result := make([]*model.Metadata, 0, len(rows))
	for i := range rows {
		m, err := rows[i].toModel()
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

// UpdateMetadata обновляет запись метаданных.
func (r *Repository) UpdateMetadata(ctx context.Context, m *model.Metadata) error {
	options, err := json.Marshal(m.Options)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE metadata SET name = $1, order_number = $2, options = $3
		 WHERE id = $4 AND user_id = $5 AND object_id = $6`,
		m.Name, m.OrderNumber, string(options), m.ID, m.UserID, m.ObjectID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

// DeleteMetadata удаляет запись метаданных.
func (r *Repository) DeleteMetadata(ctx context.Context, userID, objectID, metaID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM metadata WHERE id = $1 AND user_id = $2 AND object_id = $3`,
		metaID, userID, objectID,
	)
	if err != nil {
		return err
	}
	return checkAffected(res)
}

// checkAffected возвращает ErrNotFound, если запрос не затронул ни одной строки.
func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repoerrors.ErrNotFound
	}
	return nil
}

// metadataRow — структура для чтения записи метаданных из БД.
type metadataRow struct {
	ID          string `db:"id"`
	UserID      string `db:"user_id"`
	ObjectID    string `db:"object_id"`
	Name        string `db:"name"`
	OrderNumber int    `db:"order_number"`
	Options     string `db:"options"`
}

// toModel преобразует строку БД в доменную модель.
func (r *metadataRow) toModel() (*model.Metadata, error) {
	m := &model.Metadata{
		ID:          r.ID,
		UserID:      r.UserID,
		ObjectID:    r.ObjectID,
		Name:        r.Name,
		OrderNumber: r.OrderNumber,
		Options:     map[string]string{},
	}
	if err := json.Unmarshal([]byte(r.Options), &m.Options); err != nil {
		return nil, err
	}
	return m, nil
}

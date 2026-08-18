package repository

import (
	"context"
	"errors"

	"github.com/victor/gophkeeper/internal/config"
	"github.com/victor/gophkeeper/internal/model"
	"github.com/victor/gophkeeper/internal/repository/sqlx"
	"go.uber.org/zap"
)

type Repository interface {
	Close() error

	// CreateUser создаёт нового пользователя.
	CreateUser(ctx context.Context, user *model.User) error
	// GetUserByLogin возвращает пользователя по логину.
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	// CreateObject сохраняет новый объект.
	CreateObject(ctx context.Context, obj *model.Object) error
	// GetObject возвращает объект по идентификатору.
	GetObject(ctx context.Context, userID, id string) (*model.Object, error)
	// ListObjects возвращает объекты пользователя страницей (limit, offset),
	// отсортированные по created_at по убыванию.
	ListObjects(ctx context.Context, userID string, limit, offset int) ([]*model.Object, error)
	// CountObjects возвращает количество объектов пользователя.
	CountObjects(ctx context.Context, userID string) (int, error)
	// UpdateObject обновляет объект.
	UpdateObject(ctx context.Context, obj *model.Object) error
	// DeleteObject удаляет объект.
	DeleteObject(ctx context.Context, userID, id string) error
	// CreateMetadata сохраняет новую запись метаданных объекта.
	CreateMetadata(ctx context.Context, m *model.Metadata) error
	// GetMetadata возвращает запись метаданных.
	GetMetadata(ctx context.Context, userID, objectID, metaID string) (*model.Metadata, error)
	// ListMetadata возвращает все записи метаданных объекта.
	ListMetadata(ctx context.Context, userID, objectID string) ([]*model.Metadata, error)
	// UpdateMetadata обновляет запись метаданных.
	UpdateMetadata(ctx context.Context, m *model.Metadata) error
	// DeleteMetadata удаляет запись метаданных.
	DeleteMetadata(ctx context.Context, userID, objectID, metaID string) error
}

func New(cfg *config.Config, logger *zap.Logger) (Repository, error) {
	if cfg.DatabaseDSN != "" {
		logger.Debug("creating sqlx repository")
		return sqlx.New(cfg.DatabaseDSN, logger)
	}

	return nil, errors.New("repository: не указано ни одно хранилище")
}

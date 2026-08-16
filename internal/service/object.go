package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/crypto"
	"github.com/victor/gophkeeper/internal/model"
	repoerrors "github.com/victor/gophkeeper/internal/repository/errors"
)

// objectRepository — минимальный интерфейс хранилища, необходимый ObjectService.
type objectRepository interface {
	// CreateObject сохраняет новый объект.
	CreateObject(ctx context.Context, obj *model.Object) error
	// GetObject возвращает объект по идентификатору.
	GetObject(ctx context.Context, userID, id string) (*model.Object, error)
	// ListObjects возвращает все объекты пользователя.
	ListObjects(ctx context.Context, userID string) ([]*model.Object, error)
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

// ObjectService реализует операции с объектами и их метаданными.
type ObjectService struct {
	repo   objectRepository
	logger *zap.Logger
}

// NewObjectService создаёт сервис объектов.
func NewObjectService(repo objectRepository, logger *zap.Logger) *ObjectService {
	return &ObjectService{repo: repo, logger: logger}
}

// CreateObject создаёт новый объект пользователя.
func (s *ObjectService) CreateObject(ctx context.Context, userID, name string, typ model.SecretType, salt, ciphertext []byte) (*model.Object, error) {
	if err := validateObject(name, typ, salt, ciphertext); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	obj := &model.Object{
		ID:         uuid.NewString(),
		UserID:     userID,
		Name:       name,
		Type:       typ,
		Salt:       salt,
		Ciphertext: ciphertext,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateObject(ctx, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// GetObject возвращает объект пользователя.
func (s *ObjectService) GetObject(ctx context.Context, userID, id string) (*model.Object, error) {
	obj, err := s.repo.GetObject(ctx, userID, id)
	if errors.Is(err, repoerrors.ErrNotFound) {
		return nil, ErrNotFound
	}
	return obj, err
}

// ListObjects возвращает все объекты пользователя.
func (s *ObjectService) ListObjects(ctx context.Context, userID string) ([]*model.Object, error) {
	return s.repo.ListObjects(ctx, userID)
}

// UpdateObject обновляет содержимое объекта.
func (s *ObjectService) UpdateObject(ctx context.Context, userID, id, name string, typ model.SecretType, salt, ciphertext []byte) (*model.Object, error) {
	if err := validateObject(name, typ, salt, ciphertext); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetObject(ctx, userID, id)
	if errors.Is(err, repoerrors.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	obj := &model.Object{
		ID:         existing.ID,
		UserID:     userID,
		Name:       name,
		Type:       typ,
		Salt:       salt,
		Ciphertext: ciphertext,
		CreatedAt:  existing.CreatedAt,
		UpdatedAt:  time.Now().UTC(),
	}
	if err := s.repo.UpdateObject(ctx, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// DeleteObject удаляет объект пользователя.
func (s *ObjectService) DeleteObject(ctx context.Context, userID, id string) error {
	err := s.repo.DeleteObject(ctx, userID, id)
	if errors.Is(err, repoerrors.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// CreateMetadata создаёт запись метаданных объекта.
func (s *ObjectService) CreateMetadata(ctx context.Context, userID, objectID, name string, orderNumber int, options map[string]string) (*model.Metadata, error) {
	if name == "" {
		return nil, ErrBadRequest
	}
	if _, err := s.repo.GetObject(ctx, userID, objectID); err != nil {
		if errors.Is(err, repoerrors.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	m := &model.Metadata{
		ID:          uuid.NewString(),
		UserID:      userID,
		ObjectID:    objectID,
		Name:        name,
		OrderNumber: orderNumber,
		Options:     options,
	}
	if m.Options == nil {
		m.Options = map[string]string{}
	}
	if err := s.repo.CreateMetadata(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// ListMetadata возвращает все записи метаданных объекта.
func (s *ObjectService) ListMetadata(ctx context.Context, userID, objectID string) ([]*model.Metadata, error) {
	return s.repo.ListMetadata(ctx, userID, objectID)
}

// UpdateMetadata обновляет запись метаданных.
func (s *ObjectService) UpdateMetadata(ctx context.Context, userID, objectID, metaID, name string, orderNumber int, options map[string]string) (*model.Metadata, error) {
	if name == "" {
		return nil, ErrBadRequest
	}
	existing, err := s.repo.GetMetadata(ctx, userID, objectID, metaID)
	if errors.Is(err, repoerrors.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	m := &model.Metadata{
		ID:          existing.ID,
		UserID:      userID,
		ObjectID:    objectID,
		Name:        name,
		OrderNumber: orderNumber,
		Options:     options,
	}
	if m.Options == nil {
		m.Options = map[string]string{}
	}
	if err := s.repo.UpdateMetadata(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteMetadata удаляет запись метаданных.
func (s *ObjectService) DeleteMetadata(ctx context.Context, userID, objectID, metaID string) error {
	err := s.repo.DeleteMetadata(ctx, userID, objectID, metaID)
	if errors.Is(err, repoerrors.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// validateObject проверяет корректность входных данных объекта.
func validateObject(name string, typ model.SecretType, salt, ciphertext []byte) error {
	if name == "" {
		return ErrBadRequest
	}
	if !typ.Valid() {
		return ErrBadRequest
	}
	if len(salt) != crypto.SaltSize {
		return ErrBadRequest
	}
	if len(ciphertext) == 0 {
		return ErrBadRequest
	}
	return nil
}

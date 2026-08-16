// Package sync связывает серверный API и локальный кэш, реализуя синхронизацию
// «сервер — источник истины»: чтение через кэш с фолбэком на сервер и
// сквозную запись (write-through) на сервер с обновлением кэша.
package sync

import (
	"context"

	"github.com/victor/gophkeeper/internal/client/api"
	"github.com/victor/gophkeeper/internal/client/repository"
	"github.com/victor/gophkeeper/internal/model"
)

// Syncer — механизм синхронизации клиента.
type Syncer struct {
	api   *api.Client
	store *repository.Repository
}

// New создаёт Syncer.
func New(apiClient *api.Client, store *repository.Repository) *Syncer {
	return &Syncer{api: apiClient, store: store}
}

// Pull забирает все объекты пользователя с сервера и заменяет локальный кэш.
func (s *Syncer) Pull(ctx context.Context, token string) error {
	objects, err := s.api.ListObjects(ctx, token)
	if err != nil {
		return err
	}
	return s.store.ReplaceAll(ctx, objects)
}

// GetObject возвращает объект: сначала из кэша, при промахе — с сервера
// (с сохранением в кэш).
func (s *Syncer) GetObject(ctx context.Context, token, id string) (*model.Object, error) {
	if obj, err := s.store.GetObject(ctx, id); err == nil {
		return obj, nil
	}
	obj, err := s.api.GetObject(ctx, token, id)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpsertObject(ctx, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// CreateObject создаёт объект на сервере и обновляет кэш.
func (s *Syncer) CreateObject(ctx context.Context, token string, req api.CreateObjectRequest) (*model.Object, error) {
	obj, err := s.api.CreateObject(ctx, token, req)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpsertObject(ctx, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// UpdateObject обновляет объект на сервере и в кэше.
func (s *Syncer) UpdateObject(ctx context.Context, token, id string, req api.CreateObjectRequest) (*model.Object, error) {
	obj, err := s.api.UpdateObject(ctx, token, id, req)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpsertObject(ctx, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// DeleteObject удаляет объект на сервере и из кэша.
func (s *Syncer) DeleteObject(ctx context.Context, token, id string) error {
	if err := s.api.DeleteObject(ctx, token, id); err != nil {
		return err
	}
	return s.store.DeleteObject(ctx, id)
}

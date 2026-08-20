package service

import (
	"context"
	"io"
	"path"

	"go.uber.org/zap"
)

// fileStorage — минимальный интерфейс хранилища файлов, необходимый FileService.
type fileStorage interface {
	// Put сохраняет объект с ключом key из потока r размером size.
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	// Get возвращает поток и размер объекта по ключу.
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
	// Delete удаляет объект по ключу.
	Delete(ctx context.Context, key string) error
	// List возвращает количество и суммарный размер объектов с префиксом.
	List(ctx context.Context, prefix string) (count int, size int64, err error)
}

// FileService реализует загрузку и скачивание бинарных файлов.
type FileService struct {
	storage fileStorage
	logger  *zap.Logger
}

// NewFileService создаёт сервис файлов.
func NewFileService(storage fileStorage, logger *zap.Logger) *FileService {
	return &FileService{storage: storage, logger: logger}
}

// Upload сохраняет содержимое файла объекта под ключом userID/objectID.
func (s *FileService) Upload(ctx context.Context, userID, objectID string, r io.Reader, size int64) error {
	return s.storage.Put(ctx, s.key(userID, objectID), r, size)
}

// Download возвращает поток содержимого файла объекта.
func (s *FileService) Download(ctx context.Context, userID, objectID string) (io.ReadCloser, int64, error) {
	return s.storage.Get(ctx, s.key(userID, objectID))
}

// Delete удаляет содержимое файла объекта.
func (s *FileService) Delete(ctx context.Context, userID, objectID string) error {
	return s.storage.Delete(ctx, s.key(userID, objectID))
}

// Stats возвращает количество и суммарный размер файлов пользователя.
func (s *FileService) Stats(ctx context.Context, userID string) (int, int64, error) {
	return s.storage.List(ctx, userID+"/")
}

// key строит ключ объекта в хранилище.
func (s *FileService) key(userID, objectID string) string {
	return path.Join(userID, objectID)
}

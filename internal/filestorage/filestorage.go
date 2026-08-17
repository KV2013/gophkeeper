// Package filestorage реализует хранение бинарных файлов в S3-совместимом
// хранилище (MinIO).
package filestorage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config — параметры подключения к S3-хранилищу.
type Config struct {
	// Endpoint — адрес хранилища (host:port).
	Endpoint string
	// AccessKey — ключ доступа.
	AccessKey string
	// SecretKey — секретный ключ.
	SecretKey string
	// Bucket — имя бакета.
	Bucket string
	// UseSSL — использовать TLS.
	UseSSL bool
}

// Storage — хранилище файлов поверх MinIO.
type Storage struct {
	client *minio.Client
	bucket string
}

// New подключается к MinIO и гарантирует наличие бакета.
func New(cfg Config) (*Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	s := &Storage{client: client, bucket: cfg.Bucket}
	if err := s.ensureBucket(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

// Put сохраняет объект с ключом key из потока r размером size.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

// Get возвращает поток и размер объекта по ключу.
func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	stat, err := obj.Stat()
	if err != nil {
		return nil, 0, err
	}
	return obj, stat.Size, nil
}

// Delete удаляет объект по ключу.
func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// ensureBucket создаёт бакет, если его ещё нет.
func (s *Storage) ensureBucket(ctx context.Context) error {
	ok, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

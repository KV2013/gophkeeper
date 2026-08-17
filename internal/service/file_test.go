package service

import (
	"context"
	"io"
	"testing"

	"go.uber.org/zap"
)

// mockFileStorage — записывает переданные ключи для проверки в тестах.
type mockFileStorage struct {
	putKey    string
	getKey    string
	deleteKey string
}

func (m *mockFileStorage) Put(_ context.Context, key string, _ io.Reader, _ int64) error {
	m.putKey = key
	return nil
}

func (m *mockFileStorage) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	m.getKey = key
	return nil, 0, nil
}

func (m *mockFileStorage) Delete(_ context.Context, key string) error {
	m.deleteKey = key
	return nil
}

func TestFileServiceKey(t *testing.T) {
	tests := map[string]struct {
		fn   func(s *FileService, m *mockFileStorage) string
		want string
	}{
		"upload": {
			fn: func(s *FileService, m *mockFileStorage) string {
				_ = s.Upload(context.Background(), "user-1", "obj-1", nil, 0)
				return m.putKey
			},
			want: "user-1/obj-1",
		},
		"download": {
			fn: func(s *FileService, m *mockFileStorage) string {
				_, _, _ = s.Download(context.Background(), "user-1", "obj-1")
				return m.getKey
			},
			want: "user-1/obj-1",
		},
		"delete": {
			fn: func(s *FileService, m *mockFileStorage) string {
				_ = s.Delete(context.Background(), "user-1", "obj-1")
				return m.deleteKey
			},
			want: "user-1/obj-1",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			m := &mockFileStorage{}
			s := NewFileService(m, zap.NewNop())
			if got := tc.fn(s, m); got != tc.want {
				t.Fatalf("key: got %q, want %q", got, tc.want)
			}
		})
	}
}

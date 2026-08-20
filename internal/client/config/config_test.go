package config

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/victor/gophkeeper/internal/client/repository"
)

func TestParseTTL(t *testing.T) {
	tests := map[string]struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		"ноль — без ограничения": {in: "0", want: 0},
		"пять минут":             {in: "5m", want: 5 * time.Minute},
		"триста секунд":          {in: "300s", want: 300 * time.Second},
		"ровно 24 часа":          {in: "24h", want: 24 * time.Hour},
		"больше 24 часов":        {in: "25h", wantErr: true},
		"отрицательный":          {in: "-5m", wantErr: true},
		"мусор":                  {in: "abc", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseTTL(tc.in)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidTTL) {
					t.Fatalf("ожидалась ErrInvalidTTL, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTTL: %v", err)
			}
			if got != tc.want {
				t.Fatalf("ParseTTL: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Now()

	tests := map[string]struct {
		createdAt time.Time
		ttl       time.Duration
		want      bool
	}{
		"ttl 0 — никогда": {
			createdAt: now.Add(-time.Hour),
			ttl:       0,
			want:      false,
		},
		"не протух": {
			createdAt: now.Add(-time.Minute),
			ttl:       5 * time.Minute,
			want:      false,
		},
		"протух": {
			createdAt: now.Add(-6 * time.Minute),
			ttl:       5 * time.Minute,
			want:      true,
		},
		"ровно на границе — не протух": {
			createdAt: now.Add(-5 * time.Minute),
			ttl:       5 * time.Minute,
			want:      false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := IsExpired(tc.createdAt, now, tc.ttl); got != tc.want {
				t.Fatalf("IsExpired: got %v, want %v", got, tc.want)
			}
		})
	}
}

// mockConfigReader — in-memory реализация Reader для тестов.
type mockConfigReader struct {
	values map[string]string
}

func (m mockConfigReader) GetConfig(_ context.Context, key string) (string, error) {
	v, ok := m.values[key]
	if !ok {
		return "", repository.ErrNotFound
	}
	return v, nil
}

func TestUseCredentialsFile(t *testing.T) {
	tests := map[string]struct {
		value   string // "" — ключ отсутствует
		want    bool
		wantErr bool
	}{
		"нет ключа — false": {value: "", want: false},
		"true":              {value: "true", want: true},
		"единица":           {value: "1", want: true},
		"false":             {value: "false", want: false},
		"мусор":             {value: "abc", wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			m := mockConfigReader{}
			if tc.value != "" {
				m.values = map[string]string{KeyUseCredentialsFile: tc.value}
			}

			got, err := UseCredentialsFile(context.Background(), m)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				return
			}
			if err != nil {
				t.Fatalf("UseCredentialsFile: %v", err)
			}
			if got != tc.want {
				t.Fatalf("UseCredentialsFile: got %v, want %v", got, tc.want)
			}
		})
	}
}

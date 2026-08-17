package crypto

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func testKey(t *testing.T) Key {
	t.Helper()
	return Key{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func testSalt() []byte {
	s := make([]byte, SaltSize)
	for i := range s {
		s[i] = byte(i)
	}
	return s
}

func roundtrip(t *testing.T, plaintext []byte, chunkSize int) []byte {
	t.Helper()
	key := testKey(t)

	var enc bytes.Buffer
	r := newEncryptingReader(bytes.NewReader(plaintext), key, testSalt(), int64(len(plaintext)), chunkSize)
	if _, err := io.Copy(&enc, r); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	dr := NewDecryptingReader(bytes.NewReader(enc.Bytes()), key)
	got, err := io.ReadAll(dr)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	return got
}

func TestStreamRoundtrip(t *testing.T) {
	key := testKey(t)

	tests := map[string]struct {
		size      int
		chunkSize int
	}{
		"пустой файл":      {size: 0, chunkSize: 64},
		"меньше чанка":     {size: 10, chunkSize: 64},
		"ровно один чанк":  {size: 64, chunkSize: 64},
		"несколько чанков": {size: 200, chunkSize: 64},
		"граница чанков":   {size: 128, chunkSize: 64},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			plaintext := make([]byte, tc.size)
			for i := range plaintext {
				plaintext[i] = byte(i % 251)
			}

			got := roundtrip(t, plaintext, tc.chunkSize)
			if !bytes.Equal(got, plaintext) {
				t.Fatalf("roundtrip: got %d байт, want %d; содержимое не совпадает", len(got), len(plaintext))
			}
		})
	}

	// Большой файл (несколько МиБ) через публичный конструктор.
	t.Run("большой файл", func(t *testing.T) {
		size := 3*FileChunkSize + 12345
		plaintext := make([]byte, size)
		for i := range plaintext {
			plaintext[i] = byte(i)
		}
		var enc bytes.Buffer
		r := NewEncryptingReader(bytes.NewReader(plaintext), key, testSalt(), int64(size))
		if _, err := io.Copy(&enc, r); err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		dr := NewDecryptingReader(bytes.NewReader(enc.Bytes()), key)
		got, err := io.ReadAll(dr)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Fatal("большой файл: содержимое не совпадает")
		}
	})
}

func TestStreamTampered(t *testing.T) {
	key := testKey(t)
	plaintext := make([]byte, 200)
	for i := range plaintext {
		plaintext[i] = byte(i)
	}

	var enc bytes.Buffer
	r := newEncryptingReader(bytes.NewReader(plaintext), key, testSalt(), int64(len(plaintext)), 64)
	if _, err := io.Copy(&enc, r); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	cipher := enc.Bytes()
	cipher[len(cipher)/2] ^= 0xFF // портим данные в середине

	dr := NewDecryptingReader(bytes.NewReader(cipher), key)
	if _, err := io.ReadAll(dr); err == nil {
		t.Fatal("ожидалась ошибка при повреждении данных")
	}
}

func TestStreamInvalidMagic(t *testing.T) {
	key := testKey(t)
	bad := make([]byte, fileHeaderSize)
	copy(bad, "XXXXXX")

	dr := NewDecryptingReader(bytes.NewReader(bad), key)
	if _, err := io.ReadAll(dr); !errors.Is(err, ErrInvalidFormat) {
		t.Fatalf("ожидалась ErrInvalidFormat, got %v", err)
	}
}

func TestEncryptedFileSize(t *testing.T) {
	tests := map[string]struct {
		size      int64
		chunkSize int
		want      int64
	}{
		"пустой":            {size: 0, chunkSize: 64, want: int64(fileHeaderSize)},
		"ровно один чанк":   {size: 64, chunkSize: 64, want: int64(fileHeaderSize) + 64 + Overhead},
		"два чанка":         {size: 128, chunkSize: 64, want: int64(fileHeaderSize) + 2*(64+Overhead)},
		"неполный чанк":     {size: 70, chunkSize: 64, want: int64(fileHeaderSize) + (64 + Overhead) + (6 + Overhead)},
		"нулевой chunkSize": {size: 10, chunkSize: 0, want: int64(fileHeaderSize) + 10 + Overhead},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := EncryptedFileSize(tc.size, tc.chunkSize); got != tc.want {
				t.Fatalf("EncryptedFileSize: got %d, want %d", got, tc.want)
			}
		})
	}
}

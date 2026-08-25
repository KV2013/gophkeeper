package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestFileSHA256(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"пустая строка": {
			input: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		"abc": {
			input: "abc",
			want:  "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
		"длинная строка": {
			input: strings.Repeat("gophkeeper", 100),
			want:  hashOf(t, []byte(strings.Repeat("gophkeeper", 100))),
		},
		"бинарные байты": {
			input: string([]byte{0x00, 0x01, 0x02, 0xff}),
			want:  hashOf(t, []byte{0x00, 0x01, 0x02, 0xff}),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := fileSHA256(bytes.NewReader([]byte(tc.input)))
			if err != nil {
				t.Fatalf("fileSHA256: %v", err)
			}
			if got != tc.want {
				t.Fatalf("fileSHA256: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetectContentType(t *testing.T) {
	tests := map[string]struct {
		path string
		want string
	}{
		"текстовый файл":  {path: "a.txt", want: "text/plain; charset=utf-8"},
		"без расширения":  {path: "Makefile", want: "application/octet-stream"},
		"png":             {path: "a.png", want: "image/png"},
		"неизвестный тип": {path: "a.unknown", want: "application/octet-stream"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := detectContentType(tc.path); got != tc.want {
				t.Fatalf("detectContentType: got %q, want %q", got, tc.want)
			}
		})
	}
}

// hashOf — ожидаемый sha256 для сравнения.
func hashOf(t *testing.T, data []byte) string {
	t.Helper()
	sum, err := fileSHA256(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("fileSHA256: %v", err)
	}
	return sum
}

package main

import (
	"testing"

	"github.com/victor/gophkeeper/internal/model"
)

func TestLast4(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
	}{
		"длинная строка": {in: "1234567890123456", want: "3456"},
		"ровно 4":        {in: "1234", want: "1234"},
		"короткая":       {in: "12", want: "12"},
		"пустая":         {in: "", want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := last4(tc.in); got != tc.want {
				t.Fatalf("last4: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatDecrypted(t *testing.T) {
	tests := map[string]struct {
		typ       model.SecretType
		plaintext []byte
		reveal    bool
		want      string
	}{
		"пароль скрыт": {
			typ:       model.SecretTypeLoginPassword,
			plaintext: []byte(`{"login":"bob","password":"hunter2"}`),
			reveal:    false,
			want:      "логин: bob\nпароль: *****",
		},
		"пароль раскрыт": {
			typ:       model.SecretTypeLoginPassword,
			plaintext: []byte(`{"login":"bob","password":"hunter2"}`),
			reveal:    true,
			want:      "логин: bob\nпароль: hunter2",
		},
		"карта скрыта": {
			typ:       model.SecretTypeCard,
			plaintext: []byte(`{"number":"1234567890123456","holder":"Bob","exp_month":12,"exp_year":30,"cvv":"123"}`),
			reveal:    false,
			want:      "номер: ...3456\nдержатель: Bob\nсрок: **/**\ncvv: ***",
		},
		"карта раскрыта": {
			typ:       model.SecretTypeCard,
			plaintext: []byte(`{"number":"1234567890123456","holder":"Bob","exp_month":12,"exp_year":30,"cvv":"123"}`),
			reveal:    true,
			want:      "номер: 1234567890123456\nдержатель: Bob\nсрок: 12/30\ncvv: 123",
		},
		"текст": {
			typ:       model.SecretTypeText,
			plaintext: []byte(`{"content":"hello"}`),
			want:      "hello",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := formatDecrypted(tc.typ, tc.plaintext, tc.reveal)
			if err != nil {
				t.Fatalf("formatDecrypted: %v", err)
			}
			if got != tc.want {
				t.Fatalf("formatDecrypted: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatMetadata(t *testing.T) {
	tests := map[string]struct {
		metadata []*model.Metadata
		want     string
	}{
		"пусто": {
			metadata: []*model.Metadata{},
			want:     "(метаданных нет)",
		},
		"с опциями": {
			metadata: []*model.Metadata{{Name: "website", Options: map[string]string{"url": "example.com"}}},
			want:     "- website: map[url:example.com]\n",
		},
		"без опций": {
			metadata: []*model.Metadata{{Name: "tag"}},
			want:     "- tag\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := formatMetadata(tc.metadata); got != tc.want {
				t.Fatalf("formatMetadata: got %q, want %q", got, tc.want)
			}
		})
	}
}

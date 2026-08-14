package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKeySaltSize(t *testing.T) {
	if _, err := DeriveKey("password", []byte("short")); err == nil {
		t.Fatal("ожидалась ошибка для соли неверного размера")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	k1, err := DeriveKey("hunter2", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	k2, err := DeriveKey("hunter2", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	if k1 != k2 {
		t.Fatal("один и тот же пароль и соль должны давать одинаковый ключ")
	}
}

func TestDeriveKeyDiffersBySaltAndPassword(t *testing.T) {
	s1, _ := NewSalt()
	s2, _ := NewSalt()
	k1, _ := DeriveKey("hunter2", s1)
	k2, _ := DeriveKey("hunter2", s2)
	k3, _ := DeriveKey("hunter3", s1)
	if k1 == k2 {
		t.Fatal("разная соль должна давать разные ключи")
	}
	if k1 == k3 {
		t.Fatal("разный пароль должен давать разные ключи")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := DeriveKey("hunter2", mustSalt(t))
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	plaintext := []byte("login: bob, password: super-secret")
	box, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if len(box) != len(plaintext)+Overhead {
		t.Fatalf("неверная длина шифротекста: got %d, want %d", len(box), len(plaintext)+Overhead)
	}
	if bytes.Equal(box, plaintext) {
		t.Fatal("шифротекст не должен совпадать с открытым текстом")
	}

	got, err := Decrypt(key, box)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip не удался: got %q, want %q", got, plaintext)
	}
}

func TestEncryptNonceUnique(t *testing.T) {
	key, _ := DeriveKey("hunter2", mustSalt(t))
	b1, err := Encrypt(key, []byte("same"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	b2, err := Encrypt(key, []byte("same"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(b1, b2) {
		t.Fatal("повторное шифрование должно давать разные шифротексты")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key, _ := DeriveKey("hunter2", mustSalt(t))
	other, _ := DeriveKey("wrong", mustSalt(t))

	box, err := Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := Decrypt(other, box); err == nil {
		t.Fatal("ожидалась ошибка расшифровки чужим ключом")
	}
}

func TestDecryptTampered(t *testing.T) {
	key, _ := DeriveKey("hunter2", mustSalt(t))
	box, err := Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	box[len(box)-1] ^= 0xFF
	if _, err := Decrypt(key, box); err == nil {
		t.Fatal("ожидалась ошибка при повреждённом шифротексте")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key, _ := DeriveKey("hunter2", mustSalt(t))
	if _, err := Decrypt(key, []byte("short")); err == nil {
		t.Fatal("ожидалась ошибка для короткого шифротекста")
	}
}

func mustSalt(t *testing.T) []byte {
	t.Helper()
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	return salt
}

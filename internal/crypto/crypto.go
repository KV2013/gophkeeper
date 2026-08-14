// Package crypto реализует оконечное (end-to-end) шифрование данных
// пользователя на стороне клиента.
//
// Мастер-ключ шифрования выводится из пароля пользователя с помощью
// Argon2id и никогда не покидает клиентское приложение. Данные шифруются
// аутентифицированным шифром (XChaCha20-Poly1305) через secretbox, поэтому
// на сервер отправляется только непрозрачный шифротекст вместе с солью.
package crypto

import (
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	// SaltSize — размер соли для KDF в байтах.
	SaltSize = 16

	// KeySize — размер мастер-ключа в байтах (совпадает с secretbox.KeySize).
	KeySize = 32

	// NonceSize — размер nonce для secretbox в байтах.
	NonceSize = 24

	// Overhead — накладные расходы шифрования: nonce + тег аутентификации.
	Overhead = NonceSize + secretbox.Overhead
)

// Параметры Argon2id зафиксированы в протоколе. Их изменение приведёт к
// невозможности расшифровать данные, зашифрованные ранее другими клиентами.
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // KiB
	argon2Threads = 4
)

// Key — мастер-ключ шифрования, выведенный из пароля пользователя.
type Key [KeySize]byte

// DeriveKey выводит мастер-ключ шифрования из пароля и соли с помощью
// Argon2id. Соль должна иметь длину SaltSize.
func DeriveKey(password string, salt []byte) (Key, error) {
	var key Key
	if len(salt) != SaltSize {
		return key, errors.New("crypto: неверный размер соли")
	}
	dk := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, KeySize)
	copy(key[:], dk)
	return key, nil
}

// NewSalt генерирует криптографически стойкую соль для DeriveKey.
func NewSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt шифрует plaintext мастер-ключом key и возвращает
// nonce || ciphertext одним блоком байтов.
func Encrypt(key Key, plaintext []byte) ([]byte, error) {
	var nonce [NonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], plaintext, &nonce, (*[KeySize]byte)(&key)), nil
}

// Decrypt расшифровывает данные, полученные из Encrypt (nonce || ciphertext),
// и возвращает исходный открытый текст.
func Decrypt(key Key, box []byte) ([]byte, error) {
	if len(box) < NonceSize {
		return nil, errors.New("crypto: шифротекст слишком короткий")
	}
	var nonce [NonceSize]byte
	copy(nonce[:], box[:NonceSize])
	out, ok := secretbox.Open(nil, box[NonceSize:], &nonce, (*[KeySize]byte)(&key))
	if !ok {
		return nil, errors.New("crypto: не удалось расшифровать данные")
	}
	return out, nil
}

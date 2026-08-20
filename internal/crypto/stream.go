package crypto

import (
	"encoding/binary"
	"errors"
	"io"
)

// Параметры потокового формата файла.
const (
	// fileMagic — магическая сигнатура файла.
	fileMagic = "GKPBF1"
	// fileVersion — версия формата.
	fileVersion = byte(1)
	// FileChunkSize — размер чанка шифрования по умолчанию (1 МиБ).
	FileChunkSize = 1 << 20
	// fileHeaderSize — размер заголовка: magic(6) + version(1) +
	// chunkSize(4) + salt(16) + plaintextSize(8).
	fileHeaderSize = len(fileMagic) + 1 + 4 + SaltSize + 8
)

// ErrInvalidFormat — неверный формат зашифрованного файла.
var ErrInvalidFormat = errors.New("crypto: неверный формат файла")

// EncryptedFileSize возвращает размер зашифрованного потока для открытого
// текста заданного размера.
func EncryptedFileSize(plaintextSize int64, chunkSize int) int64 {
	if chunkSize <= 0 {
		chunkSize = FileChunkSize
	}
	full := plaintextSize / int64(chunkSize)
	rem := plaintextSize % int64(chunkSize)
	total := int64(fileHeaderSize) + full*int64(chunkSize+Overhead)
	if rem > 0 {
		total += rem + Overhead
	}
	return total
}

// EncryptingReader шифрует поток открытого текста чанками и выдаёт
// заголовок + зашифрованные чанки.
type EncryptingReader struct {
	src       io.Reader
	key       Key
	chunkSize int

	header    []byte
	headerOff int

	pending    []byte
	pendingOff int

	remaining int64
	buf       []byte
}

// NewEncryptingReader создаёт потоковый шифратор. salt и plaintextSize
// записываются в заголовок.
func NewEncryptingReader(src io.Reader, key Key, salt []byte, plaintextSize int64) *EncryptingReader {
	return newEncryptingReader(src, key, salt, plaintextSize, FileChunkSize)
}

// newEncryptingReader — внутренний конструктор с настраиваемым размером чанка.
func newEncryptingReader(src io.Reader, key Key, salt []byte, plaintextSize int64, chunkSize int) *EncryptingReader {
	if chunkSize <= 0 {
		chunkSize = FileChunkSize
	}
	return &EncryptingReader{
		src:       src,
		key:       key,
		chunkSize: chunkSize,
		header:    buildFileHeader(salt, chunkSize, plaintextSize),
		remaining: plaintextSize,
		buf:       make([]byte, chunkSize),
	}
}

// Read реализует io.Reader.
func (r *EncryptingReader) Read(p []byte) (int, error) {
	if r.headerOff < len(r.header) {
		n := copy(p, r.header[r.headerOff:])
		r.headerOff += n
		return n, nil
	}
	if r.pendingOff < len(r.pending) {
		n := copy(p, r.pending[r.pendingOff:])
		r.pendingOff += n
		return n, nil
	}
	if r.remaining <= 0 {
		return 0, io.EOF
	}

	chunk := r.chunkSize
	if int64(chunk) > r.remaining {
		chunk = int(r.remaining)
	}
	n, err := io.ReadFull(r.src, r.buf[:chunk])
	switch {
	case err == io.ErrUnexpectedEOF, err == io.EOF:
		r.remaining = 0
	case err != nil:
		return 0, err
	default:
		r.remaining -= int64(n)
	}
	if n == 0 {
		return 0, io.EOF
	}

	enc, err := Encrypt(r.key, r.buf[:n])
	if err != nil {
		return 0, err
	}
	r.pending = enc
	r.pendingOff = 0

	written := copy(p, r.pending)
	r.pendingOff = written
	return written, nil
}

// DecryptingReader читает зашифрованный поток и выдаёт расшифрованный текст.
type DecryptingReader struct {
	src io.Reader
	key Key

	header    []byte
	headerOff int
	parsed    bool

	chunkSize int
	remaining int64

	out    []byte
	outOff int
}

// NewDecryptingReader создаёт потоковый расшифровщик поверх зашифрованного
// потока src.
func NewDecryptingReader(src io.Reader, key Key) *DecryptingReader {
	return &DecryptingReader{
		src:    src,
		key:    key,
		header: make([]byte, fileHeaderSize),
	}
}

// Read реализует io.Reader.
func (r *DecryptingReader) Read(p []byte) (int, error) {
	if !r.parsed {
		if _, err := io.ReadFull(r.src, r.header); err != nil {
			return 0, err
		}
		if err := r.parseHeader(); err != nil {
			return 0, err
		}
		r.parsed = true
	}

	if r.outOff < len(r.out) {
		n := copy(p, r.out[r.outOff:])
		r.outOff += n
		return n, nil
	}
	if r.remaining <= 0 {
		return 0, io.EOF
	}

	plainLen := int64(r.chunkSize)
	if plainLen > r.remaining {
		plainLen = r.remaining
	}
	enc := make([]byte, plainLen+Overhead)
	if _, err := io.ReadFull(r.src, enc); err != nil {
		return 0, err
	}
	plain, err := Decrypt(r.key, enc)
	if err != nil {
		return 0, err
	}
	r.remaining -= plainLen
	r.out = plain
	r.outOff = 0

	n := copy(p, r.out)
	r.outOff = n
	return n, nil
}

// parseHeader разбирает заголовок зашифрованного файла.
func (r *DecryptingReader) parseHeader() error {
	if string(r.header[:len(fileMagic)]) != fileMagic {
		return ErrInvalidFormat
	}
	if r.header[len(fileMagic)] != fileVersion {
		return ErrInvalidFormat
	}
	r.chunkSize = int(binary.LittleEndian.Uint32(r.header[len(fileMagic)+1 : len(fileMagic)+5]))
	if r.chunkSize <= 0 {
		return ErrInvalidFormat
	}
	r.remaining = int64(binary.LittleEndian.Uint64(r.header[len(fileMagic)+5+SaltSize:]))
	return nil
}

// buildFileHeader собирает заголовок файла.
func buildFileHeader(salt []byte, chunkSize int, plaintextSize int64) []byte {
	h := make([]byte, fileHeaderSize)
	copy(h[:len(fileMagic)], fileMagic)
	h[len(fileMagic)] = fileVersion
	binary.LittleEndian.PutUint32(h[len(fileMagic)+1:], uint32(chunkSize))
	copy(h[len(fileMagic)+5:len(fileMagic)+5+SaltSize], salt)
	binary.LittleEndian.PutUint64(h[len(fileMagic)+5+SaltSize:], uint64(plaintextSize))
	return h
}

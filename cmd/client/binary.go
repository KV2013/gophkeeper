package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/victor/gophkeeper/internal/client/api"
	"github.com/victor/gophkeeper/internal/crypto"
	"github.com/victor/gophkeeper/internal/model"
)

// binaryMeta — метаданные бинарного файла, хранимые в поле ciphertext объекта.
type binaryMeta struct {
	// Size — размер исходного файла в байтах.
	Size int64 `json:"size"`
	// ChunkSize — размер чанка шифрования.
	ChunkSize int `json:"chunk_size"`
	// ContentType — MIME-тип содержимого.
	ContentType string `json:"content_type"`
	// SHA256 — sha256-хэш исходного файла (hex).
	SHA256 string `json:"sha256"`
}

// detectContentType определяет MIME-тип по расширению файла.
func detectContentType(path string) string {
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// fileSHA256 вычисляет sha256-хэш потока r (hex).
func fileSHA256(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// addBinaryFile создаёт бинарный объект и загружает файл на сервер потоком.
func addBinaryFile(a *app, token string, salt []byte, name, filePath string) {
	p := filePath
	if p == "" {
		var err error
		if p, err = prompt("путь к файлу: "); err != nil {
			fatal("не удалось прочитать путь: %v", err)
		}
	}
	if p == "" {
		fatal("путь к файлу обязателен")
	}

	f, err := os.Open(p)
	if err != nil {
		fatal("не удалось открыть файл: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		fatal("не удалось прочитать файл: %v", err)
	}

	sum, err := fileSHA256(f)
	if err != nil {
		fatal("не удалось вычислить хэш файла: %v", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		fatal("не удалось перечитать файл: %v", err)
	}

	key, err := a.masterKey()
	if err != nil {
		fatal("%v", err)
	}

	metaJSON, err := json.Marshal(binaryMeta{
		Size:        stat.Size(),
		ChunkSize:   crypto.FileChunkSize,
		ContentType: detectContentType(p),
		SHA256:      sum,
	})
	if err != nil {
		fatal("не удалось сериализовать метаданные: %v", err)
	}

	obj, err := a.sync.CreateObject(ctx(), token, api.CreateObjectRequest{
		Name:       name,
		Type:       model.SecretTypeBinary,
		Salt:       salt,
		Ciphertext: metaJSON,
	})
	if err != nil {
		fatal("не удалось создать объект: %v", err)
	}

	encSize := crypto.EncryptedFileSize(stat.Size(), crypto.FileChunkSize)
	encReader := crypto.NewEncryptingReader(f, key, salt, stat.Size())
	if err := a.api.UploadFile(ctx(), token, obj.ID, encReader, encSize); err != nil {
		fatal("не удалось загрузить файл: %v", err)
	}
	fmt.Printf("создан объект %s\n", obj.ID)
}

// cmdDownload обрабатывает команду download (скачивание файла по id объекта).
func cmdDownload(serverURL string, args []string) {
	if len(args) == 0 {
		fatal("укажите идентификатор объекта: download <id>")
	}
	id := args[0]

	a := mustApp(serverURL)
	defer a.close()

	token, err := a.requireToken()
	if err != nil {
		fatal("%v", err)
	}

	obj, err := a.sync.GetObject(ctx(), token, id)
	if err != nil {
		fatal("не удалось получить объект: %v", err)
	}
	if obj.Type != model.SecretTypeBinary {
		fatal("объект %s не является файлом", id)
	}

	getBinaryFile(a, token, obj)
}

// printBinaryInfo выводит метаданные бинарного объекта из таблицы objects.
func printBinaryInfo(obj *model.Object) {
	fmt.Printf("имя: %s\n", obj.Name)
	fmt.Printf("тип: %s\n", obj.Type)

	var meta binaryMeta
	if err := json.Unmarshal(obj.Ciphertext, &meta); err != nil {
		fmt.Printf("(метаданные файла недоступны: %v)\n", err)
		return
	}
	fmt.Printf("размер: %d байт\n", meta.Size)
	fmt.Printf("content-type: %s\n", meta.ContentType)
	fmt.Printf("sha256: %s\n", meta.SHA256)
}

// addBinaryFileKey загружает бинарный файл, используя уже готовый ключ.
// Не запрашивает ввод пользователя; возвращает ошибку при неудаче.
func addBinaryFileKey(a *app, token string, salt []byte, key crypto.Key, name, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}

	sum, err := fileSHA256(f)
	if err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	metaJSON, err := json.Marshal(binaryMeta{
		Size:        stat.Size(),
		ChunkSize:   crypto.FileChunkSize,
		ContentType: detectContentType(path),
		SHA256:      sum,
	})
	if err != nil {
		return err
	}

	obj, err := a.sync.CreateObject(context.Background(), token, api.CreateObjectRequest{
		Name:       name,
		Type:       model.SecretTypeBinary,
		Salt:       salt,
		Ciphertext: metaJSON,
	})
	if err != nil {
		return err
	}

	encSize := crypto.EncryptedFileSize(stat.Size(), crypto.FileChunkSize)
	encReader := crypto.NewEncryptingReader(f, key, salt, stat.Size())
	return a.api.UploadFile(context.Background(), token, obj.ID, encReader, encSize)
}

// getBinaryFile скачивает бинарный файл и сохраняет его расшифрованным на диск.
func getBinaryFile(a *app, token string, obj *model.Object) {
	var meta binaryMeta
	if err := json.Unmarshal(obj.Ciphertext, &meta); err != nil {
		fatal("не удалось разобрать метаданные файла: %v", err)
	}

	key, err := a.masterKey()
	if err != nil {
		fatal("%v", err)
	}

	rc, _, err := a.api.DownloadFile(ctx(), token, obj.ID)
	if err != nil {
		fatal("не удалось скачать файл: %v", err)
	}
	defer rc.Close()

	outPath, err := prompt(fmt.Sprintf("куда сохранить [имя: %s]: ", obj.Name))
	if err != nil {
		fatal("не удалось прочитать путь: %v", err)
	}
	if outPath == "" {
		fatal("путь для сохранения обязателен")
	}

	out, err := os.Create(outPath)
	if err != nil {
		fatal("не удалось создать файл: %v", err)
	}
	defer out.Close()

	h := sha256.New()
	dec := crypto.NewDecryptingReader(rc, key)
	if _, err := io.Copy(out, io.TeeReader(dec, h)); err != nil {
		fatal("не удалось расшифровать/сохранить файл: %v", err)
	}

	if meta.SHA256 != "" {
		if got := hex.EncodeToString(h.Sum(nil)); got != meta.SHA256 {
			fatal("контрольная сумма не совпадает: ожидалось %s, получено %s", meta.SHA256, got)
		}
	}
	fmt.Printf("файл сохранён: %s\n", outPath)
}

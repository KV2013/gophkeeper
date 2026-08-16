package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gofrs/flock"

	clientdaemon "github.com/victor/gophkeeper/internal/client/daemon"
	clientkeyring "github.com/victor/gophkeeper/internal/client/keyring"
	clientpath "github.com/victor/gophkeeper/internal/client/path"
	"github.com/victor/gophkeeper/internal/client/repository"
)

// cmdDaemon запускает фоновый процесс очистки протухших секретов.
func cmdDaemon(serverURL string) {
	dataDir, err := clientpath.DataDir()
	if err != nil {
		fatal("не удалось определить каталог данных: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		fatal("не удалось создать каталог данных: %v", err)
	}

	fileLock := flock.New(filepath.Join(dataDir, "gophkeeper.daemon.lock"))
	locked, err := fileLock.TryLock()
	if err != nil {
		fatal("не удалось захватить блокировку: %v", err)
	}
	if !locked {
		fatal("демон уже запущен")
	}
	defer fileLock.Unlock()

	store, err := repository.New(filepath.Join(dataDir, "gophkeeper.db"))
	if err != nil {
		fatal("не удалось открыть хранилище: %v", err)
	}
	defer store.Close()

	ks := clientkeyring.New(keyringService, filepath.Join(dataDir, "credentials.json"))

	d := clientdaemon.New(ks, store, 0)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	fmt.Println("демон запущен")
	if err := d.Run(ctx); err != nil {
		fatal("демон завершился с ошибкой: %v", err)
	}
	fmt.Println("демон остановлен")
}

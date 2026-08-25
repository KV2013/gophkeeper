// Package path определяет каталог хранения данных клиента на разных ОС.
package path

import (
	"os"
	"path/filepath"
	"runtime"
)

// appName — имя приложения, используемое в пути.
const appName = "gophkeeper"

// DataDir возвращает каталог данных приложения для текущей ОС.
//
//   - Linux:   $XDG_DATA_HOME/gophkeeper (по умолчанию ~/.local/share/gophkeeper)
//   - macOS:   ~/Library/Application Support/gophkeeper
//   - Windows: %LOCALAPPDATA%\gophkeeper
func DataDir() (string, error) {
	base, err := osBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// DBPath возвращает путь к файлу SQLite-кэша клиента.
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gophkeeper.db"), nil
}

// osBaseDir возвращает базовый каталог данных конкретной ОС.
func osBaseDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return base, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Local"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	default: // linux и прочие
		if base := os.Getenv("XDG_DATA_HOME"); base != "" {
			return base, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}

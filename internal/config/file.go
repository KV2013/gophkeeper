package config

import (
	"encoding/json"
	"os"
)

// LoadConfigFile загружает конфигурацию из JSON-файла по пути configPath.
// Если путь пуст, используется значение переменной окружения CONFIG.
func LoadConfigFile(cfg *Config, configPath string) error {
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}
	if configPath == "" {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, cfg)
}

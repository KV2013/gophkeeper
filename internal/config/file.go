package config

import (
	"encoding/json"
	"os"
)

func LoadConfigFile(cfg *Config) error {
	cfgPath := os.Getenv("CONFIG")
	if flagConfigPath != "" {
		cfgPath = flagConfigPath
	}
	if cfgPath == "" {
		return nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, cfg)
}

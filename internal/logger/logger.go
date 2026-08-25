// Package logger предоставляет конструктор zap-логгера сервера.
package logger

import "go.uber.org/zap"

// New создаёт zap-логгер с заданным уровнем логирования.
func New(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Level = lvl
	return cfg.Build()
}

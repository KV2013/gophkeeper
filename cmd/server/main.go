package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/victor/gophkeeper/internal/config"
	"github.com/victor/gophkeeper/internal/handler"
	"github.com/victor/gophkeeper/internal/logger"
	"github.com/victor/gophkeeper/internal/repository"
	"github.com/victor/gophkeeper/internal/router"
	"github.com/victor/gophkeeper/internal/service"
	"github.com/victor/gophkeeper/internal/tlscert"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal("Ошибка при сборке конфига", zap.Error(err))
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatal("Ошибка при создании логгера", zap.Error(err))
	}

	repo, err := repository.New(cfg, log)
	if err != nil {
		log.Fatal("Ошибка при создании репозитория", zap.Error(err))
	}
	defer repo.Close()

	authService := service.NewAuthService(repo, log, cfg.JWTSecret, cfg.TokenTTL)
	objectService := service.NewObjectService(repo, log)
	h := handler.New(authService, objectService, log)

	mux := router.Init(h, log, cfg)
	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	var certPaths tlscert.CertPaths
	if cfg.EnableHTTPS {
		certPaths, err = tlscert.ProvideCertAndKey()
		if err != nil {
			log.Fatal("Не удалось сгенерировать TLS сертификат", zap.Error(err))
		}
	}

	go func() {
		if cfg.EnableHTTPS {
			log.Info("Сервер запущен (HTTPS)",
				zap.String("serverAddress", cfg.ServerAddress),
				zap.String("logLevel", cfg.LogLevel))
			if err := srv.ListenAndServeTLS(certPaths.CertPath, certPaths.KeyPath); err != nil && err != http.ErrServerClosed {
				log.Fatal("Не удалось запустить сервер", zap.Error(err))
			}
		} else {
			log.Info("Сервер запущен",
				zap.String("serverAddress", cfg.ServerAddress),
				zap.String("logLevel", cfg.LogLevel))
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal("Не удалось запустить сервер", zap.Error(err))
			}
		}
	}()

	// Ожидаем сигналов для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	log.Info("Получен сигнал завершения. Начинаем graceful shutdown...")

	// Graceful shutdown с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Graceful shutdown не удался", zap.Error(err))
	}

	log.Info("Сервер успешно остановлен")
}

func nA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", nA(buildVersion))
	fmt.Printf("Build date: %s\n", nA(buildDate))
	fmt.Printf("Build commit: %s\n", nA(buildCommit))
}

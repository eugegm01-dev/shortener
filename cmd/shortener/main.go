package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/service"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/eugegm01-dev/shortener/migrations"
	"github.com/eugegm01-dev/shortener/pkg/logger"
	"github.com/go-chi/chi/v5"
)

func main() {
	logger.Init()
	cfg := config.LoadConfig()

	go func() {
		logger.Logger.Info().Msg("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			logger.Logger.Error().Err(err).Msg("pprof server failed")
		}
	}()

	var store storage.Storage
	var err error

	if cfg.DatabaseDSN != "" {
		// Передаём встроенные миграции
		store, err = storage.NewPostgresStorage(cfg.DatabaseDSN, migrations.FS)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to create postgres storage")
		}
	} else if cfg.FileStoragePath != "" {
		store, err = storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to create file storage")
		}
	} else {
		store = storage.NewMemoryStorage()
	}

	// Инициализация фонового воркера (Fan-In Batch Deleter)
	deleter := service.NewDeleter(store, 100, 5*time.Second)

	h := handler.New(store, cfg, deleter)
	router := chi.NewRouter()
	h.RegisterRoutes(router)

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Logger.Info().Msgf("Starting server on %s", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	<-stop
	logger.Logger.Info().Msg("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Останавливаем приём новых HTTP-запросов
	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Error().Err(err).Msg("Server shutdown error")
	}

	// 2. Flush-им фоновые задачи (гарантированная доставка удалений в БД)
	logger.Logger.Info().Msg("Flushing background deleter...")
	deleter.Close()

	// 3. Закрываем соединение с БД / Файлом
	if err := store.Close(); err != nil {
		logger.Logger.Error().Err(err).Msg("Storage close error")
	}

	logger.Logger.Info().Msg("Server stopped gracefully")
}

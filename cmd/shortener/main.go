package main

import (
	"context"
	"net/http"
	_ "net/http/pprof" // <-- добавлено
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/eugegm01-dev/shortener/pkg/logger"
	"github.com/go-chi/chi/v5"
)

func main() {
	logger.Init()
	cfg := config.LoadConfig()

	// Запускаем pprof сервер в отдельной горутине
	go func() {
		logger.Logger.Info().Msg("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			logger.Logger.Error().Err(err).Msg("pprof server failed")
		}
	}()

	var store storage.Storage
	var err error

	if cfg.DatabaseDSN != "" {
		store, err = storage.NewPostgresStorage(cfg.DatabaseDSN)
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
	defer store.Close()

	h := handler.New(store, cfg)

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
		logger.Logger.Info().Msgf("Base URL: %s", cfg.BaseURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	<-stop
	logger.Logger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Server shutdown error")
	}

	logger.Logger.Info().Msg("Server stopped gracefully")
}
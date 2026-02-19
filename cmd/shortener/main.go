package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/eugegm01-dev/shortener/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Init()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize storage based on config
	var store storage.Storage
	if cfg.FileStoragePath != "" {
		fileStore, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to create file storage")
		}
		store = fileStore
	} else {
		store = storage.NewMemoryStorage()
	}
	defer store.Close()

	// Initialize handlers
	h := handler.New(store, cfg)

	// Setup router
	router := chi.NewRouter()
	h.RegisterRoutes(router)

	// Configure HTTP server
	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	// Graceful shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run server in goroutine
	go func() {
		logger.Logger.Info().Msgf("Starting server on %s", cfg.ServerAddr)
		logger.Logger.Info().Msgf("Base URL: %s", cfg.BaseURL)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	// Wait for stop signal
	<-stop
	logger.Logger.Info().Msg("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Server shutdown error")
	}

	logger.Logger.Info().Msg("Server stopped gracefully")
}

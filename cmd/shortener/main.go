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
	// Инициализация логгера
	logger.Init()

	// Загрузка конфигурации
	cfg := config.LoadConfig()

	// Инициализация хранилища
	store := storage.NewMemoryStorage()
	defer store.Close()

	// Инициализация обработчиков
	h := handler.New(store, cfg)

	// Настройка маршрутов
	router := chi.NewRouter()
	h.RegisterRoutes(router)

	// Настройка HTTP сервера
	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: router,
	}

	// Канал для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Запуск сервера в отдельной горутине
	go func() {
		logger.Logger.Info().Msgf("Starting server on %s", cfg.ServerAddr)
		logger.Logger.Info().Msgf("Base URL: %s", cfg.BaseURL)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal().Err(err).Msg("Server error")
		}
	}()

	// Ожидание сигнала остановки
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

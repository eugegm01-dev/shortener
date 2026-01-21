package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/storage"
)

func main() {
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
		log.Printf("Starting server on %s", cfg.ServerAddr)
		log.Printf("Base URL: %s", cfg.BaseURL)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Ожидание сигнала остановки
	<-stop
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped gracefully")
}

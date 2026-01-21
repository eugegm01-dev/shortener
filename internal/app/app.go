package app

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/storage"
)

// Application представляет основное приложение
type Application struct {
	cfg     *config.Config
	storage storage.Storage
	router  chi.Router
	server  *http.Server
}

// New создает новое приложение
func New(cfg *config.Config, storage storage.Storage) *Application {
	app := &Application{
		cfg:     cfg,
		storage: storage,
		router:  chi.NewRouter(),
	}

	// Инициализация обработчиков
	h := handler.New(storage, cfg)
	h.RegisterRoutes(app.router)

	// Настройка HTTP сервера
	app.server = &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: app.router,
	}

	return app
}

// Run запускает приложение
func (app *Application) Run() error {
	return app.server.ListenAndServe()
}

// Shutdown корректно останавливает приложение
func (app *Application) Shutdown(ctx context.Context) error {
	// Закрываем хранилище
	if err := app.storage.Close(); err != nil {
		return err
	}

	// Останавливаем сервер
	return app.server.Shutdown(ctx)
}

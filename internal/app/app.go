package app

import (
	"context"
	"net/http"
	"time"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/service"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Application представляет основное приложение
type Application struct {
	cfg     *config.Config
	storage storage.Storage
	router  chi.Router
	server  *http.Server
	deleter *service.Deleter
}

// New создает новое приложение
func New(cfg *config.Config, storage storage.Storage) *Application {
	// Инициализируем deleter, чтобы app.go компилировался
	deleter := service.NewDeleter(storage, 100, 5*time.Second)

	app := &Application{
		cfg:     cfg,
		storage: storage,
		router:  chi.NewRouter(),
		deleter: deleter,
	}

	h := handler.New(storage, cfg, deleter)
	h.RegisterRoutes(app.router)

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
	if app.deleter != nil {
		app.deleter.Close()
	}
	if err := app.storage.Close(); err != nil {
		return err
	}
	return app.server.Shutdown(ctx)
}

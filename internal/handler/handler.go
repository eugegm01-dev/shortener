package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/models"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/eugegm01-dev/shortener/pkg/logger"
	mw "github.com/eugegm01-dev/shortener/pkg/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// Handler обработчик HTTP запросов
type Handler struct {
	storage storage.Storage
	cfg     *config.Config
}

// New создает новый обработчик
func New(storage storage.Storage, cfg *config.Config) *Handler {
	return &Handler{
		storage: storage,
		cfg:     cfg,
	}
}

// RegisterRoutes регистрирует маршруты
func (h *Handler) RegisterRoutes(r chi.Router) {
    r.Use(mw.DecompressMiddleware)
    r.Use(mw.LoggerMiddleware)
    r.Use(chiMiddleware.Recoverer)
    r.Use(chiMiddleware.Compress(5))

    r.Get("/ping", h.Ping)
    r.Post("/", h.ShortenURL)
    r.Get("/{id}", h.RedirectURL)
    r.Post("/api/shorten", h.ShortenURLJSON)
    r.Post("/api/shorten/batch", h.ShortenURLBatch) // ← новый хендлер
}
// Ping проверяет доступность сервиса
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.storage.Ping(); err != nil {
		// Log the actual error for debugging
		logger.Logger.Error().Err(err).Msg("Ping failed")
		h.sendJSONError(w, "Storage unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

// ShortenURL сокращает URL из формы
func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendPlainError(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := string(body)
	if originalURL == "" {
		h.sendPlainError(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	id, err := h.storage.Save(originalURL)
	if err != nil {
		h.sendPlainError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", h.cfg.BaseURL, id)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// ShortenURLJSON сокращает URL из JSON запроса
func (h *Handler) ShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	var req models.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.URL == "" {
		h.sendJSONError(w, "URL is required", http.StatusBadRequest)
		return
	}

	id, err := h.storage.Save(req.URL)
	if err != nil {
		h.sendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.ShortenResponse{
		Result: fmt.Sprintf("%s/%s", h.cfg.BaseURL, id),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// RedirectURL перенаправляет по короткой ссылке
func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.sendPlainError(w, "ID is required", http.StatusBadRequest)
		return
	}

	originalURL, err := h.storage.Get(id)
	if err != nil {
		h.sendPlainError(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// sendJSONError отправляет ошибку в формате JSON
func (h *Handler) sendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

// sendPlainError отправляет ошибку в виде plain text
func (h *Handler) sendPlainError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(message))
}
// ShortenURLBatch сокращает несколько URL из JSON-массива
func (h *Handler) ShortenURLBatch(w http.ResponseWriter, r *http.Request) {
        // 🔍 DEBUG: читаем тело ДО декодирования
        body, err := io.ReadAll(r.Body)
        if err != nil {
            logger.Logger.Error().Err(err).Msg("Failed to read request body")
            h.sendJSONError(w, "Cannot read body", http.StatusBadRequest)
            return
        }
        logger.Logger.Info().Str("raw_body", string(body)).Msg("Batch request received")

        // Восстанавливаем body для json.Decoder
        r.Body = io.NopCloser(bytes.NewBuffer(body))

        // Дальше твой код...
        var req []models.BatchShortenRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            logger.Logger.Error().Err(err).Str("body", string(body)).Msg("JSON decode failed")
            h.sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
            return
        }
    defer r.Body.Close()

    // Пустой батч — возвращаем пустой ответ
    if len(req) == 0 {
        w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)

        json.NewEncoder(w).Encode([]models.BatchShortenResponse{})
        return
    }

    // Валидация и сбор оригинальных URL
    originalURLs := make([]string, 0, len(req))
    for _, item := range req {
        if item.OriginalURL == "" {
            h.sendJSONError(w, "URL cannot be empty", http.StatusBadRequest)
            return
        }
        originalURLs = append(originalURLs, item.OriginalURL)
    }

    // Сохраняем батч
    ids, err := h.storage.SaveBatch(originalURLs)
    if err != nil {
        h.sendJSONError(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Формируем ответ с сохранением порядка correlation_id
    resp := make([]models.BatchShortenResponse, 0, len(req))
    for i, item := range req {
        resp = append(resp, models.BatchShortenResponse{
            CorrelationID: item.CorrelationID,
            ShortURL:      fmt.Sprintf("%s/%s", h.cfg.BaseURL, ids[i]),
        })
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted) // 202 — принято на обработку
    json.NewEncoder(w).Encode(resp)
}
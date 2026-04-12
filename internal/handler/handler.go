package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eugegm01-dev/shortener/internal/audit"
	"github.com/eugegm01-dev/shortener/internal/auth"
	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/models"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/eugegm01-dev/shortener/pkg/logger"
	mw "github.com/eugegm01-dev/shortener/pkg/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const userIDKey contextKey = "userID"

type Handler struct {
	storage      storage.Storage
	cfg          *config.Config
	auditSubject *audit.Subject
}

func New(storage storage.Storage, cfg *config.Config) *Handler {
	h := &Handler{
		storage:      storage,
		cfg:          cfg,
		auditSubject: audit.NewSubject(),
	}

	// Attach file observer if path provided
	if cfg.AuditFile != "" {
		fo, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			logger.Logger.Error().Err(err).Str("path", cfg.AuditFile).Msg("Failed to create file audit observer")
		} else {
			h.auditSubject.Attach(fo)
			logger.Logger.Info().Str("path", cfg.AuditFile).Msg("File audit observer enabled")
		}
	}

	// Attach HTTP observer if URL provided
	if cfg.AuditURL != "" {
		ho := audit.NewHTTPObserver(cfg.AuditURL)
		h.auditSubject.Attach(ho)
		logger.Logger.Info().Str("url", cfg.AuditURL).Msg("HTTP audit observer enabled")
	}

	return h
}

// RegisterRoutes registers all HTTP routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Use(mw.DecompressMiddleware)
	r.Use(mw.LoggerMiddleware)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Compress(5))
	r.Use(h.authMiddleware)

	r.Get("/ping", h.Ping)
	r.Post("/", h.ShortenURL)
	r.Get("/{id}", h.RedirectURL)
	r.Post("/api/shorten", h.ShortenURLJSON)
	r.Post("/api/shorten/batch", h.ShortenURLBatch)
	r.Get("/api/user/urls", h.GetUserURLs)
	r.Delete("/api/user/urls", h.DeleteUserURLs)
}

// authMiddleware checks or sets the authentication cookie.
func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth.VerifyCookie(r, h.cfg.SecretKey)
		if err != nil {
			userID = generateUserID()
			cookie, err := auth.SignCookie(userID, h.cfg.SecretKey)
			if err != nil {
				logger.Logger.Error().Err(err).Msg("Failed to sign cookie")
			} else {
				http.SetCookie(w, cookie)
			}
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateUserID creates a random user ID.
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// getUserIDFromContext extracts userID from request context.
func (h *Handler) getUserIDFromContext(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(userIDKey).(string)
	return userID, ok
}

// Ping checks storage health.
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.storage.Ping(); err != nil {
		logger.Logger.Error().Err(err).Msg("Ping failed")
		h.sendJSONError(w, "Storage unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

// ShortenURL handles plain text shortening.
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

	userID, _ := h.getUserIDFromContext(r)
	var id string
	var created bool

	if storageWithUser, ok := h.storage.(interface {
		SaveWithUser(url, userID string) (string, bool, error)
	}); ok && userID != "" {
		id, created, err = storageWithUser.SaveWithUser(originalURL, userID)
	} else {
		id, created, err = h.storage.Save(originalURL)
	}

	if err != nil {
		h.sendPlainError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", h.cfg.BaseURL, id)
	w.Header().Set("Content-Type", "text/plain")
	if !created {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write([]byte(shortURL))

	// Audit event
	h.auditSubject.Notify(audit.Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    originalURL,
	})
}

// ShortenURLJSON handles JSON shortening.
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

	userID, _ := h.getUserIDFromContext(r)
	var id string
	var created bool
	var err error

	if storageWithUser, ok := h.storage.(interface {
		SaveWithUser(url, userID string) (string, bool, error)
	}); ok && userID != "" {
		id, created, err = storageWithUser.SaveWithUser(req.URL, userID)
	} else {
		id, created, err = h.storage.Save(req.URL)
	}

	if err != nil {
		h.sendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.ShortenResponse{
		Result: fmt.Sprintf("%s/%s", h.cfg.BaseURL, id),
	}
	w.Header().Set("Content-Type", "application/json")
	if !created {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to encode JSON response")
	}

	// Audit event
	h.auditSubject.Notify(audit.Event{
		TS:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    req.URL,
	})
}

// RedirectURL redirects short URL to original.
func (h *Handler) RedirectURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.sendPlainError(w, "ID is required", http.StatusBadRequest)
		return
	}

	originalURL, err := h.storage.Get(id)
	if err != nil {
		if errors.Is(err, storage.ErrGone) {
			h.sendPlainError(w, "URL has been deleted", http.StatusGone)
			return
		}
		h.sendPlainError(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)

	userID, _ := h.getUserIDFromContext(r)
	h.auditSubject.Notify(audit.Event{
		TS:     time.Now().Unix(),
		Action: "follow",
		UserID: userID,
		URL:    originalURL,
	})
}

// GetUserURLs returns all URLs belonging to the authenticated user.
func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.VerifyCookie(r, h.cfg.SecretKey)
	if err != nil {
		if _, cookieErr := r.Cookie("auth_token"); cookieErr == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if storageWithUser, ok := h.storage.(interface {
		GetByUser(userID string) ([]models.UserURL, error)
	}); ok {
		urls, err := storageWithUser.GetByUser(userID)
		if err != nil {
			logger.Logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user URLs")
			h.sendJSONError(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if len(urls) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		response := make([]map[string]string, len(urls))
		for i, url := range urls {
			shortURL := h.cfg.BaseURL + "/" + strings.TrimPrefix(url.ShortURL, "http://localhost:8080/")
			response[i] = map[string]string{
				"short_url":    shortURL,
				"original_url": url.OriginalURL,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Logger.Error().Err(err).Msg("Failed to encode user URLs response")
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// ShortenURLBatch shortens multiple URLs in one request.
func (h *Handler) ShortenURLBatch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to read request body")
		h.sendJSONError(w, "Cannot read body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	var req []models.BatchShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Logger.Error().Err(err).Str("body", string(body)).Msg("JSON decode failed")
		h.sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(req) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]models.BatchShortenResponse{})
		return
	}

	originalURLs := make([]string, 0, len(req))
	for _, item := range req {
		if item.OriginalURL == "" {
			h.sendJSONError(w, "URL cannot be empty", http.StatusBadRequest)
			return
		}
		originalURLs = append(originalURLs, item.OriginalURL)
	}

	ids, err := h.storage.SaveBatch(originalURLs)
	if err != nil {
		h.sendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]models.BatchShortenResponse, 0, len(req))
	for i, item := range req {
		resp = append(resp, models.BatchShortenResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", h.cfg.BaseURL, ids[i]),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// DeleteUserURLs soft-deletes user's URLs asynchronously.
func (h *Handler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromContext(r)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		h.sendJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(shortIDs) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	go func() {
		if storageWithDelete, ok := h.storage.(interface {
			DeleteUserURLs(userID string, shortIDs []string) error
		}); ok {
			if err := storageWithDelete.DeleteUserURLs(userID, shortIDs); err != nil {
				logger.Logger.Error().Err(err).Str("user_id", userID).Msg("Failed to delete URLs")
			}
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

// sendJSONError sends an error response in JSON format.
func (h *Handler) sendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

// sendPlainError sends a plain text error response.
func (h *Handler) sendPlainError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(message))
}

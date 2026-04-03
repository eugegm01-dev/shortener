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

// RegisterRoutes registers routes.
func (h *Handler) RegisterRoutes(r chi.Router) {
	// ... (existing middleware registration)
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

// ... (authMiddleware, generateUserID, getUserIDFromContext unchanged)

// ShortenURL shortens URL from plain text form.
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
		Ts:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    originalURL,
	})
}

// ShortenURLJSON shortens URL from JSON request.
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
	json.NewEncoder(w).Encode(resp)

	// Audit event
	h.auditSubject.Notify(audit.Event{
		Ts:     time.Now().Unix(),
		Action: "shorten",
		UserID: userID,
		URL:    req.URL,
	})
}

// RedirectURL redirects to original URL.
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
	// Audit event
	h.auditSubject.Notify(audit.Event{
		Ts:     time.Now().Unix(),
		Action: "follow",
		UserID: userID,
		URL:    originalURL,
	})
}

// ... (other methods unchanged: GetUserURLs, ShortenURLBatch, DeleteUserURLs, etc.)

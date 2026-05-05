// Package models defines data transfer objects (DTOs) for the URL shortener API.
package models

// URL represents a stored URL mapping.
type URL struct {
	ID  string `json:"id"`  // Short identifier (key)
	URL string `json:"url"` // Original URL
}

// ShortenRequest is the JSON request body for POST /api/shorten.
type ShortenRequest struct {
	URL string `json:"url"` // The original URL to shorten
}

// ShortenResponse is the JSON response for POST /api/shorten.
type ShortenResponse struct {
	Result string `json:"result"` // The shortened URL
}

// ErrorResponse is the standard error response format for API errors.
type ErrorResponse struct {
	Error string `json:"error"` // Human‑readable error message
}

// BatchShortenRequest represents a single item in a batch shortening request.
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"` // Client‑supplied ID to match response items
	OriginalURL   string `json:"original_url"`   // The URL to shorten
}

// BatchShortenResponse represents a single item in a batch shortening response.
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"` // Same as in the request
	ShortURL      string `json:"short_url"`      // The shortened URL
}

// UserURL represents a user's short‑to‑original URL mapping.
type UserURL struct {
	ShortURL    string `json:"short_url"`    // The shortened URL
	OriginalURL string `json:"original_url"` // The original URL
}

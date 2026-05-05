// Package audit implements an observer‑based audit logging system.
package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// Event represents an audit log entry.
type Event struct {
	TS     int64  `json:"ts"`                // Unix timestamp (seconds)
	Action string `json:"action"`            // Action type: "shorten" or "follow"
	UserID string `json:"user_id,omitempty"` // User ID (may be empty for anonymous)
	URL    string `json:"url"`               // The original URL that was shortened or followed
}

// Observer defines the interface that all audit observers must implement.
type Observer interface {
	// Send delivers an audit event to the observer.
	// The implementation must be safe for concurrent use.
	Send(event Event) error
}

// Subject manages a list of observers and dispatches events to them asynchronously.
type Subject struct {
	observers []Observer
	mu        sync.RWMutex
}

// NewSubject creates a new audit subject with no observers.
func NewSubject() *Subject {
	return &Subject{
		observers: make([]Observer, 0),
	}
}

// Attach adds an observer to the subject.
func (s *Subject) Attach(o Observer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, o)
}

// Notify sends the event to all attached observers in separate goroutines.
// Errors from observers are ignored to avoid blocking the main request flow.
func (s *Subject) Notify(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.observers {
		go func(obs Observer, ev Event) {
			_ = obs.Send(ev)
		}(o, event)
	}
}

// FileObserver writes audit events as newline‑separated JSON lines to a file.
type FileObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileObserver creates a file observer. It verifies that the file
// can be opened for appending or created, but does not keep it open.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &FileObserver{filePath: path}, nil
}

// Send appends a JSON‑encoded event line to the audit file.
func (f *FileObserver) Send(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = file.Write(data)
	return err
}

// HTTPObserver sends audit events via HTTP POST requests.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTP observer with a 5‑second timeout.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Send performs a POST request with the event JSON as the body.
// Returns an error if the request fails or the response status is >= 400.
func (h *HTTPObserver) Send(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("audit HTTP observer got status %d", resp.StatusCode)
	}
	return nil
}

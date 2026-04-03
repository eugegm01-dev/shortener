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

// Event represents an audit event.
type Event struct {
	TS     int64  `json:"ts"`                // unix timestamp
	Action string `json:"action"`            // "shorten" or "follow"
	UserID string `json:"user_id,omitempty"` // may be empty
	URL    string `json:"url"`               // original URL
}

// Observer defines the interface for audit observers.
type Observer interface {
	Send(event Event) error
}

// Subject manages observers and notifies them.
type Subject struct {
	observers []Observer
	mu        sync.RWMutex
}

// NewSubject creates a new audit subject.
func NewSubject() *Subject {
	return &Subject{
		observers: make([]Observer, 0),
	}
}

// Attach adds an observer.
func (s *Subject) Attach(o Observer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, o)
}

// Notify sends the event to all observers asynchronously.
func (s *Subject) Notify(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.observers {
		go func(obs Observer, ev Event) {
			_ = obs.Send(ev) // errors are ignored for now
		}(o, event)
	}
}

// FileObserver writes events to a file (one JSON per line).
type FileObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileObserver creates a file observer.
// It ensures the file exists (or creates it) but does not keep it open.
func NewFileObserver(path string) (*FileObserver, error) {
	// Create or open file to verify permissions
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &FileObserver{filePath: path}, nil
}

// Send appends a JSON line to the file.
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

// HTTPObserver sends events via HTTP POST.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver creates an HTTP observer.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Send performs a POST request with the event JSON.
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
	// discard body to reuse connection
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("audit HTTP observer got status %d", resp.StatusCode)
	}
	return nil
}

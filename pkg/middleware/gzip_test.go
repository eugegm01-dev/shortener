package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecompressMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})
	mw := DecompressMiddleware(handler)

	// Сжимаем тело
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("hello"))
	gz.Close()

	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Body.String() != "hello" {
		t.Errorf("Expected 'hello', got '%s'", rr.Body.String())
	}
}

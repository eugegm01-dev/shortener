// Package middleware provides HTTP middleware for logging, decompression, etc.
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// DecompressMiddleware decompresses the request body if the
// Content-Encoding header is "gzip". If the body is not a valid
// gzip stream, it returns HTTP 400 Bad Request.
func DecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			r.Body = &gzipReadCloser{
				Reader:   gzReader,
				original: r.Body,
			}
		}
		next.ServeHTTP(w, r)
	})
}

// gzipReadCloser wraps a gzip.Reader and closes both the reader and the original body.
type gzipReadCloser struct {
	*gzip.Reader
	original io.ReadCloser
}

// Close closes the gzip reader and the original request body.
func (g *gzipReadCloser) Close() error {
	g.Reader.Close()
	return g.original.Close()
}

package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// DecompressMiddleware decompresses the request body if Content-Encoding: gzip is set.
func DecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the client sent a gzipped body
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			// Create a gzip reader
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				// If the body isn't valid gzip, return 400 Bad Request
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			// Replace the request body with the decompressed reader
			// Important: we also need to close the gzip reader when done
			r.Body = &gzipReadCloser{
				Reader:   gzReader,
				original: r.Body,
			}
		}
		// Pass the (possibly modified) request to the next handler
		next.ServeHTTP(w, r)
	})
}

// gzipReadCloser ensures that both the gzip.Reader and the original body are closed.
type gzipReadCloser struct {
	*gzip.Reader
	original io.ReadCloser
}

func (g *gzipReadCloser) Close() error {
	// Close both the gzip reader and the original body
	g.Reader.Close()
	return g.original.Close()
}

package middleware

import (
	"net/http"
	"time"

	"github.com/eugegm01-dev/shortener/pkg/logger"
)

// LoggerMiddleware логирует запросы и ответы
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Запоминаем время начала запроса
		start := time.Now()

		// Создаем обертку для ответа, чтобы получить код статуса и размер
		lrw := &loggingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Выполняем следующий обработчик
		next.ServeHTTP(lrw, r)

		// Вычисляем время выполнения
		elapsed := time.Since(start)

		// Логируем информацию о запросе и ответе
		// Все сообщения на уровне Info
		logger.Logger.Info().
			Str("uri", r.RequestURI).
			Str("method", r.Method).
			Int("status", lrw.statusCode).
			Int("size", lrw.size).
			Dur("duration", elapsed).
			Msg("processed request")
	})
}

// loggingResponseWriter обертка для http.ResponseWriter для перехвата статуса и размера
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader перехватывает код статуса
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write перехватывает размер ответа
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(b)
	lrw.size += size
	return size, err
}

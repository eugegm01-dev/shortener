package models

// URL представляет собой модель данных для хранения ссылок
type URL struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// ShortenRequest представляет запрос на сокращение URL
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет ответ на запрос сокращения URL
type ShortenResponse struct {
	Result string `json:"result"`
}

// ErrorResponse представляет структуру для ошибок API
type ErrorResponse struct {
	Error string `json:"error"`
}
// BatchShortenRequest элемент запроса на пакетное сокращение
type BatchShortenRequest struct {
    CorrelationID string `json:"correlation_id"`
    OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse элемент ответа на пакетное сокращение
type BatchShortenResponse struct {
    CorrelationID string `json:"correlation_id"`
    ShortURL      string `json:"short_url"`
}

// UserURL — ссылка с привязкой к пользователю
type UserURL struct {
    ShortURL    string `json:"short_url"`
    OriginalURL string `json:"original_url"`
}
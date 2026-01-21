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

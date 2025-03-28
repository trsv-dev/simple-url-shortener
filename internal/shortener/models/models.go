package models

// Request Модель запроса
type Request struct {
	URL string `json:"url"`
}

// Response Модель ответа
type Response struct {
	Result string `json:"result"`
}

// BatchItem Модель для множественной обработки URL
type BatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
	ShortURL      string `json:"short_url"`
}

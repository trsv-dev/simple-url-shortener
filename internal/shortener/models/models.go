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
	CorrelationID string `json:"correlation_id,omitempty"`
	OriginalURL   string `json:"original_url"`
	ShortURL      string `json:"short_url"`
}

// Item Модель для записи URL в файловую БД
type Item struct {
	UUID      string `json:"uuid"`
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"long_url"`
	UserID    string `json:"user_id"`
	IsDeleted bool   `json:"is_deleted"`
}

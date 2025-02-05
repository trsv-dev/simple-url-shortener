package models

// Request Модель запроса
type Request struct {
	URL string `json:"url"`
}

// Response Модель ответа
type Response struct {
	Result string `json:"result"`
}

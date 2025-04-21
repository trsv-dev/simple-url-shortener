package mocks

import "fmt"

// MockGenerator Мок-генератор, который либо возвращает заданную ошибку, либо заданный shortURL
type MockGenerator struct {
	ShortURL string
	Err      error
	Urls     []string
	Call     int
}

// MakeShortURL Метод мок-генератора для выдачи сокращенных URL
func (mg *MockGenerator) MakeShortURL(length int) (string, error) {
	// если при инициализации структуры передана ошибка - возвращаем ошибку
	if mg.Err != nil {
		return "", mg.Err
	}

	// если при инициализации структуры передан список коротких URL - возвращаем их по одному
	// и считаем их оставшееся количество когда выдаем для разных запросов
	if len(mg.Urls) > 0 {
		if mg.Call < len(mg.Urls) {
			url := mg.Urls[mg.Call]
			mg.Call++
			return url, nil
		}
		// Если закончились подготовленные URL
		return "", fmt.Errorf("нет больше shortURL в списке")
	}

	// если при инициализации структуры передан shortURL - возвращаем shortURL
	return mg.ShortURL, nil
}

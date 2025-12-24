package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func ExampleAppHandler_PostHandler() {
	secretKey := "Secret_Key"
	tokenExp := 10 * time.Minute

	// имитируем хранилище с заранее известной связкой "длинный URL → короткий URL"
	storage := &MockStorage{
		mockData: map[string]string{
			"https://example.com": "shortURL",
		},
	}

	// создаём обработчик приложения с этим хранилищем
	h := NewAppHandler(storage, secretKey, tokenExp)

	// формируем HTTP-запрос на POST с длинным URL в теле
	r := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("https://example.com"),
	)

	// создаём ResponseRecorder, чтобы перехватить ответ обработчика
	w := httptest.NewRecorder()

	// вызываем обработчик
	h.PostHandler(w, r)

	// выводим HTTP-статус
	fmt.Println(w.Code)

	// выводим тело ответа (короткий URL)
	fmt.Println(strings.TrimSpace(w.Body.String()))

	// Output:
	// 201
	// http://localhost:8080/shortURL
}

func ExampleAppHandler_GetHandler() {
	// имитируем хранилище с заранее известной связкой "короткий URL → длинный URL"
	storage := &MockStorage{
		mockData: map[string]string{
			"shortURL": "https://example.com",
		},
	}

	secretKey := "Secret_Key"
	tokenExp := 10 * time.Minute

	// создаём обработчик приложения с этим хранилищем
	h := NewAppHandler(storage, secretKey, tokenExp)

	// формируем HTTP-запрос GET по короткой ссылке.
	r := httptest.NewRequest(http.MethodGet, "/shortURL", nil)

	// создаём ResponseRecorder, чтобы перехватить ответ обработчика
	w := httptest.NewRecorder()

	// вызываем обработчик
	h.GetHandler(w, r)

	// выводим HTTP-статус
	fmt.Println(w.Code)

	// выводим заголовок Location (куда перенаправляет)
	fmt.Println(w.Header().Get("Location"))

	// Output:
	// 307
	// https://example.com
}

func ExampleAppHandler_APIShorten() {
	// имитируем хранилище с заранее известной связкой "длинный URL → короткий URL".
	storage := &MockStorage{
		mockData: map[string]string{
			"https://example.com": "shortURL",
		},
	}

	secretKey := "Secret_Key"
	tokenExp := 10 * time.Minute

	// создаём обработчик приложения с этим хранилищем
	h := NewAppHandler(storage, secretKey, tokenExp)

	// формируем HTTP-запрос POST с JSON-телом
	r := httptest.NewRequest(
		http.MethodPost,
		"/api/shorten",
		strings.NewReader(`{"url":"https://example.com"}`),
	)

	r.Header.Set("Content-Type", "application/json")

	// создаём ResponseRecorder, чтобы перехватить ответ обработчика
	w := httptest.NewRecorder()

	// вызываем обработчик
	h.APIShorten(w, r)

	// выводим статус ответа
	fmt.Println(w.Code)

	// выводим тело ответа
	fmt.Println(strings.TrimSpace(w.Body.String()))

	// Output:
	// 201
	// {"result":"http://localhost:8080/shortURL"}
}

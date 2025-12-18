package generator

import (
	"errors"
	"math/rand"
	"strings"
	"time"
)

// URLGenerator Интерфейс генератора коротких ссылок
type URLGenerator interface {
	MakeShortURL(int) (string, error)
}

var latin = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890"

// DefaultGenerator Генератор коротких ссылок по умолчанию.
// Random в структуре нужен, чтобы избежать возможных гонок.
type DefaultGenerator struct {
	random *rand.Rand
}

// NewDefaultGenerator Конструктор, возвращающий дефолтовый генератор URL.
func NewDefaultGenerator() *DefaultGenerator {
	return &DefaultGenerator{random: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// MakeShortURL Создание рандомной строки заданной длины из константного набора символов.
func (dg *DefaultGenerator) MakeShortURL(length int) (string, error) {
	if length < 8 {
		return "", errors.New("переданная длина слишком мала для генерации URL")
	}
	var builder strings.Builder
	builder.Grow(length) // Предварительное выделение памяти

	for i := 0; i < length; i++ {
		builder.WriteByte(latin[dg.random.Intn(len(latin))])
	}
	return builder.String(), nil

	// для тестов
	//return "12345678", nil
}

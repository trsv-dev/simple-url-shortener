package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsValidURL(t *testing.T) {
	testTable := []struct {
		name           string
		longURL        string
		expectedAnswer bool
	}{
		{name: "Invalid URL. Without domain zone (https://example)", longURL: "https://example", expectedAnswer: false},
		{name: "Invalid URL. FTP instead of HTTPS(S) (ftp://example.com)", longURL: "ftp://example.com", expectedAnswer: false},
		{name: "Invalid URL. Incomplete (://example.com)", longURL: "://example.com", expectedAnswer: false},
		{name: "Invalid URL. Without domain name (https://)", longURL: "https://", expectedAnswer: false},
		{name: "Valid URL", longURL: "https://example.com", expectedAnswer: true},
		{name: "Valid URL. With subdomain", longURL: "http://subdomain.example.com/path", expectedAnswer: true},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedAnswer, IsValidURL(tt.longURL))
		})
	}
}

func TestMakeShortURL(t *testing.T) {
	testTable := []struct {
		name          string
		length        int
		wantError     bool
		expectedError string
	}{
		{name: "Generate URL normal length", length: 8, wantError: false},
		{name: "Generate URL small length", length: 4, wantError: true, expectedError: "переданная длина слишком мала для генерации URL"},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			generated, err := MakeShortURL(tt.length)
			if tt.wantError {
				assert.Error(t, err, "Ожидалась ошибка генерации короткого URL")
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}
			assert.NoError(t, err, "Ошибка генерации короткого URL")
			assert.Equal(t, len(generated), tt.length)

			for _, char := range generated {
				assert.Contains(t, latin, string(char))
			}
		})
	}
}

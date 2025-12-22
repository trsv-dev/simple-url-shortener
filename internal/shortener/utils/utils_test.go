package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

package pgstorage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertDSNToPostgresURI(t *testing.T) {
	testTable := []struct {
		name        string
		testDSN     string
		expectedURI string
		emptyResult bool
	}{
		{
			name:        "Valid PostgreSQL DSN string",
			testDSN:     "host=localhost port=5432 user=test password=testpassword dbname=testdb sslmode=disable",
			expectedURI: "postgres://test:testpassword@localhost:5432/testdb?sslmode=disable",
			emptyResult: false,
		},
		{
			name:        "Empty PostgreSQL DSN string",
			testDSN:     "",
			expectedURI: "postgres://:@:/?sslmode=",
			emptyResult: true,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			dbURI := ConvertDSNToPostgresURI(tt.testDSN)

			if !tt.emptyResult {
				assert.Equal(t, tt.expectedURI, dbURI)
				return
			}

			assert.Equal(t, tt.expectedURI, dbURI)
		})
	}
}

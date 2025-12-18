package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Тестирование дефольного генератора коротких URL.
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
			gen := NewDefaultGenerator()
			generated, err := gen.MakeShortURL(tt.length)
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

package models

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRequestSerialization(t *testing.T) {
	t.Run("Request serialization", func(t *testing.T) {
		request := Request{URL: "http://example.com"}
		expectedJSON := `{"url":"http://example.com"}`

		data, err := json.Marshal(request)
		require.NoError(t, err)

		require.Equal(t, expectedJSON, string(data))
		assert.JSONEq(t, expectedJSON, string(data))
	})
}

func TestRequestDeserialization(t *testing.T) {
	t.Run("Request deserialization", func(t *testing.T) {
		var request Request
		data := []byte(`{"url":"http://example.com"}`)

		err := json.Unmarshal(data, &request)
		require.NoError(t, err)

		require.Equal(t, "http://example.com", request.URL)
	})
}

func TestResponseSerialization(t *testing.T) {
	t.Run("Response serialization", func(t *testing.T) {
		request := Response{Result: "shortURL"}
		expectedJSON := `{"result":"shortURL"}`

		data, err := json.Marshal(request)
		require.NoError(t, err)

		require.Equal(t, expectedJSON, string(data))
		assert.JSONEq(t, expectedJSON, string(data))
	})
}

func TestResponseDeserialization(t *testing.T) {
	t.Run("Request deserialization", func(t *testing.T) {
		var response Response
		data := []byte(`{"result":"shortURL"}`)

		err := json.Unmarshal(data, &response)
		require.NoError(t, err)

		require.Equal(t, "shortURL", response.Result)
	})
}

package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitConfig(t *testing.T) {

	testTable := []struct {
		name                  string
		longURL               string
		shortURL              string
		expectedServerAddress string
		expectedBaseURL       string
	}{
		{name: "Valid data in InitConfig()", expectedServerAddress: "localhost:8080", expectedBaseURL: "http://localhost:8080"},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			cfg := InitConfig()

			assert.Equal(t, tt.expectedServerAddress, cfg.ServerAddress, "ServerAddress не совпадает с ожидаемым")
			assert.Equal(t, tt.expectedBaseURL, cfg.BaseURL, "BaseURL не совпадает с ожидаемым")
		})
	}
}

func TestGetServerConfig(t *testing.T) {
	testTable := []struct {
		name                 string
		serverConfig         *Config
		expectedServerConfig *Config
	}{
		{name: "Initialized config struct", serverConfig: &Config{ServerAddress: "host-from-flags:8001", BaseURL: "http://address-from-flags:8001"}, expectedServerConfig: &Config{ServerAddress: "host-from-flags:8001", BaseURL: "http://address-from-flags:8001"}},
		{name: "Not initialized config struct", serverConfig: nil, expectedServerConfig: &Config{ServerAddress: "localhost:8080", BaseURL: "http://localhost:8080"}},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {

			ServerConfig = tt.serverConfig
			// Если tt.serverConfig = nil, то GetServerConfig() вернёт конфигурацию сервера по умолчанию.
			mockServerConfig := GetServerConfig()

			assert.Equal(t, tt.expectedServerConfig, mockServerConfig)
		})
	}
}

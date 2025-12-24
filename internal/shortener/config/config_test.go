package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Вспомогательная функция сброса состояния флагов, так как при использовании InitConfig() запускается flag.Parse().
// Однако flag.Parse() можно вызвать только один раз за время выполнения программы, поэтому в каждом тесте, где мы
// используем InitConfig() необходимо в начале сбросить флаги.
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

func TestInitConfig(t *testing.T) {
	testTable := []struct {
		name                  string
		expectedServerAddress string
		expectedBaseURL       string
	}{
		{name: "Valid data in InitConfig()", expectedServerAddress: "localhost:8080", expectedBaseURL: "http://localhost:8080"},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()

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
		{
			name:                 "Initialized config struct",
			serverConfig:         &Config{ServerAddress: "host-from-flags:8001", BaseURL: "http://address-from-flags:8001"},
			expectedServerConfig: &Config{ServerAddress: "host-from-flags:8001", BaseURL: "http://address-from-flags:8001"},
		},
		{
			name:         "Not initialized config struct",
			serverConfig: nil,
			expectedServerConfig: &Config{ServerAddress: "localhost:8080",
				BaseURL: "http://localhost:8080",
				Logging: true, LogLevel: "Info",
				FileDB:    "/tmp/short-url-storage.json",
				SecretKey: "Change_Me_On_Production",
				TokenExp:  86400000000000,
			},
		},
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

func TestOverrideFromEnv(t *testing.T) {
	testTable := []struct {
		name                 string
		serverConfig         *Config
		expectedServerConfig *Config
		envKey1              string
		envValue1            string
		envKey2              string
		envValue2            string
		envKey3              string
		envValue3            string
		envKey4              string
		envValue4            string
		envKey5              string
		envValue5            string
		envKey6              string
		envValue6            string
		envKey7              string
		envValue7            string
	}{
		{
			name: "Override server address and base url config fields given from default config to environment values",
			expectedServerConfig: &Config{ServerAddress: "host-from-env:8002",
				BaseURL:   "http://address-from-env:8002",
				Logging:   true,
				LogLevel:  "Debug",
				FileDB:    "/tmp/env-storage.json",
				SecretKey: "Change_Me_On_Production",
				TokenExp:  86400000000000,
			},
			envKey1:   "SERVER_ADDRESS",
			envValue1: "host-from-env:8002",
			envKey2:   "BASE_URL",
			envValue2: "http://address-from-env:8002",
			envKey3:   "LOGGING",
			envValue3: "True",
			envKey4:   "LOG_LEVEL",
			envValue4: "Debug",
			envKey5:   "FILE_STORAGE_PATH",
			envValue5: "/tmp/env-storage.json",
		},
		{
			name: "Override server address and base url config fields given from flags to environment values",
			serverConfig: &Config{ServerAddress: "host-from-flags:8001",
				BaseURL:   "http://address-from-flags:8001",
				Logging:   false,
				LogLevel:  "Info",
				FileDB:    "/tmp/flag-storage.json",
				SecretKey: "Key_From_Flags",
				TokenExp:  120000,
			},
			expectedServerConfig: &Config{ServerAddress: "host-from-env:8002",
				BaseURL:   "http://address-from-env:8002",
				Logging:   true,
				LogLevel:  "Debug",
				FileDB:    "/tmp/env-storage.json",
				SecretKey: "Change_Me_On_Production",
				TokenExp:  86400000000000,
			},
			envKey1:   "SERVER_ADDRESS",
			envValue1: "host-from-env:8002",
			envKey2:   "BASE_URL",
			envValue2: "http://address-from-env:8002",
			envKey3:   "LOGGING",
			envValue3: "True",
			envKey4:   "LOG_LEVEL",
			envValue4: "Debug",
			envKey5:   "FILE_STORAGE_PATH",
			envValue5: "/tmp/env-storage.json",
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()

			if tt.envKey1 != "" && tt.envKey2 != "" {
				os.Setenv(tt.envKey1, tt.envValue1)
				os.Setenv(tt.envKey2, tt.envValue2)
				os.Setenv(tt.envKey3, tt.envValue3)
				os.Setenv(tt.envKey4, tt.envValue4)
				os.Setenv(tt.envKey5, tt.envValue5)
				os.Setenv(tt.envKey6, tt.envValue6)
				os.Setenv(tt.envKey7, tt.envValue7)
				defer os.Unsetenv(tt.envKey1)
				defer os.Unsetenv(tt.envKey2)
				defer os.Unsetenv(tt.envKey3)
				defer os.Unsetenv(tt.envKey4)
				defer os.Unsetenv(tt.envKey5)
				defer os.Unsetenv(tt.envKey6)
				defer os.Unsetenv(tt.envKey7)
			}

			if tt.serverConfig != nil {
				ServerConfig = tt.serverConfig
			}

			ServerConfig = InitConfig()

			assert.Equal(t, tt.expectedServerConfig, ServerConfig)
		})
	}
}

func TestOverrideFromEnvUnsupportedType(t *testing.T) {
	var unsupportedType float64
	envKey := "UNSUPPORTED_TYPE"
	envValue := "1.23456789"

	os.Setenv(envKey, envValue)
	defer os.Unsetenv(envKey)

	// Вызываем OverrideFromEnv с неподдерживаемым типом;
	// переменная unsupportedType не должна перезаписаться
	OverrideFromEnv(&unsupportedType, envKey)

	// Проверяем, что значение переменной не изменилось
	if unsupportedType != 0 {
		t.Errorf("Expected unsupportedType to remain 0, got %v", unsupportedType)
	}
}

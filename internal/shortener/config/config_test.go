package config

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// resetFlags сбрасывает состояние флагов, потому что в InitConfig() вызывается flag.Parse().
// flag.Parse() можно вызвать только один раз за программу, поэтому в каждом тесте с InitConfig()
// нужно в начале очищать флаги.
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

// TestInitConfig проверяет, что InitConfig() Возвращает конфигурацию со значениями по умолчанию.
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

// TestGetServerConfig Проверяет GetServerConfig() для инициализированной и не инициализированной конфигурации.
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
				FileDB:        "/tmp/short-url-storage.json",
				SecretKey:     "Change_Me_On_Production",
				TokenExp:      86400000000000,
				TrustedSubnet: "127.0.0.1/8",
			},
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {

			ServerConfig = tt.serverConfig
			// если tt.serverConfig = nil, то GetServerConfig() вернёт конфигурацию сервера по умолчанию.
			mockServerConfig := GetServerConfig()

			assert.Equal(t, tt.expectedServerConfig, mockServerConfig)
		})
	}
}

// TestOverrideFromEnv Проверяет переопределение значений конфигурации из переменных окружения.
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

			// не регистрируем флаги вручную — InitConfig() это сделает;
			// если нужно имитировать флаги, можно использовать flag.Set после InitConfig(),
			// но в этих сценариях флаги не задаются явно.

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

// TestOverrideFromEnvUnsupportedType Проверяет, что неподдерживаемый тип не перезаписывается из ENV.
func TestOverrideFromEnvUnsupportedType(t *testing.T) {
	var unsupportedType float64
	envKey := "UNSUPPORTED_TYPE"
	envValue := "1.23456789"

	os.Setenv(envKey, envValue)
	defer os.Unsetenv(envKey)

	// вызываем OverrideFromEnv с неподдерживаемым типом;
	// переменная unsupportedType не должна перезаписаться
	OverrideFromEnv(&unsupportedType, envKey)

	// проверяем, что значение переменной не изменилось
	if unsupportedType != 0 {
		t.Errorf("Expected unsupportedType to remain 0, got %v", unsupportedType)
	}
}

// TestGetConfigFileMissingFile Проверяет GetConfigFile для несуществующего файла.
func TestGetConfigFileMissingFile(t *testing.T) {
	path := filepath.Join(os.TempDir(), "nonexistent_config.json")

	cfg, err := GetConfigFile(path)

	assert.Nil(t, cfg)
	assert.Error(t, err)
}

// TestGetConfigFileInvalidJSON Проверяет GetConfigFile для некорректного JSON.
func TestGetConfigFileInvalidJSON(t *testing.T) {
	path := filepath.Join(os.TempDir(), "invalid_config.json")
	content := "{ not valid json }"

	err := os.WriteFile(path, []byte(content), 0o600)
	assert.NoError(t, err)
	defer os.Remove(path)

	cfg, err := GetConfigFile(path)

	assert.Nil(t, cfg)
	assert.Error(t, err)
}

// TestGetConfigFileValidJSON проверяет GetConfigFile для корректного JSON с заполненными полями.
func TestGetConfigFileValidJSON(t *testing.T) {
	path := filepath.Join(os.TempDir(), "valid_config.json")
	cfgInput := Config{
		ServerAddress: "host-from-file:9000",
		BaseURL:       "http://file-host:9000",
		Logging:       false,
		LogLevel:      "Warn",
		FileDB:        "/tmp/file-db.json",
		SecretKey:     "Key_From_File",
		TokenExp:      12 * time.Hour,
	}

	data, err := json.Marshal(cfgInput)
	assert.NoError(t, err)

	err = os.WriteFile(path, data, 0o600)
	assert.NoError(t, err)
	defer os.Remove(path)

	cfg, err := GetConfigFile(path)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, cfgInput.ServerAddress, cfg.ServerAddress)
	assert.Equal(t, cfgInput.BaseURL, cfg.BaseURL)
	assert.Equal(t, cfgInput.Logging, cfg.Logging)
	assert.Equal(t, cfgInput.LogLevel, cfg.LogLevel)
	assert.Equal(t, cfgInput.FileDB, cfg.FileDB)
	assert.Equal(t, cfgInput.SecretKey, cfg.SecretKey)
	assert.Equal(t, cfgInput.TokenExp, cfg.TokenExp)
}

// TestLoadConfigFromFile Проверяет LoadConfigFromFile в разных сценариях:
// - когда флаги не заданы и значения из файла применяются;
// - когда флаги заданы и значения из файла игнорируются;
// - когда в файле некоторые поля пустые и не должны перезаписывать конфиг.
func TestLoadConfigFromFile(t *testing.T) {
	testTable := []struct {
		name           string
		fileConfig     *Config
		expectedConfig *Config
		fileContent    Config
		flagA          string
		flagB          string
		flagL          string
		flagF          string
	}{
		{
			name:  "File values applied when no flags set",
			flagA: "",
			flagB: "",
			flagL: "",
			flagF: "",
			fileConfig: &Config{
				ServerAddress: "host-from-file:9000",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "Warn",
				FileDB:        "/tmp/file-db.json",
			},
			expectedConfig: &Config{
				ServerAddress: "host-from-file:9000",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "Warn",
				FileDB:        "/tmp/file-db.json",
			},
			fileContent: Config{
				ServerAddress: "host-from-file:9000",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "Warn",
				FileDB:        "/tmp/file-db.json",
			},
		},
		{
			name:  "File values ignored when flags set for a and b",
			flagA: "host-from-flag:8888",
			flagB: "http://flag-host:8888",
			flagL: "",
			flagF: "",
			fileConfig: &Config{
				ServerAddress: "host-from-file:9000",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "Warn",
				FileDB:        "/tmp/file-db.json",
			},
			expectedConfig: &Config{
				ServerAddress: "host-from-flag:8888",
				BaseURL:       "http://flag-host:8888",
				Logging:       false,
				LogLevel:      "Warn",
				FileDB:        "/tmp/file-db.json",
			},
			fileContent: Config{
				ServerAddress: "host-from-file:9000",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "Warn",
				FileDB:        "/tmp/file-db.json",
			},
		},
		{
			name:  "Only non-empty file fields applied",
			flagA: "",
			flagB: "",
			flagL: "",
			flagF: "",
			fileConfig: &Config{
				ServerAddress: "",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "", // empty
				FileDB:        "/tmp/file-db.json",
			},
			expectedConfig: &Config{
				ServerAddress: "localhost:8080", // из дефолта флага, файл пустой
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "",
				FileDB:        "/tmp/file-db.json",
			},
			fileContent: Config{
				ServerAddress: "",
				BaseURL:       "http://file-host:9000",
				Logging:       false,
				LogLevel:      "",
				FileDB:        "/tmp/file-db.json",
			},
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags()

			// регистрируем флаги так же, как в InitConfig;
			// это нужно только для тестов LoadConfigFromFile, где мы не вызываем InitConfig.
			cfgFlag := &Config{}
			flag.StringVar(&cfgFlag.ServerAddress, "a", "localhost:8080", "HTTP server address")
			flag.StringVar(&cfgFlag.BaseURL, "b", "http://localhost:8080", "Base URL for shortened links")
			flag.BoolVar(&cfgFlag.Logging, "l", true, "Enable or disable logging by -l=true or -l=false")
			flag.StringVar(&cfgFlag.FileDB, "f", "/tmp/short-url-storage.json", "Set JSON file to save short URLs. To disable this feature - set empty key")

			// если в тесте заданы значения флагов, устанавливаем их через flag.Set
			if tt.flagA != "" {
				flag.Set("a", tt.flagA)
			}
			if tt.flagB != "" {
				flag.Set("b", tt.flagB)
			}
			if tt.flagL != "" {
				flag.Set("l", tt.flagL)
			}
			if tt.flagF != "" {
				flag.Set("f", tt.flagF)
			}

			flag.Parse() // <-- Parse после flag.Set

			path := filepath.Join(os.TempDir(), "load_config_test.json")
			data, err := json.Marshal(tt.fileContent)
			assert.NoError(t, err)

			err = os.WriteFile(path, data, 0o600)
			assert.NoError(t, err)
			defer os.Remove(path)

			// передаём в LoadConfigFromFile структуру, связанную с флагами
			result, err := LoadConfigFromFile(cfgFlag, path)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedConfig.ServerAddress, result.ServerAddress)
			assert.Equal(t, tt.expectedConfig.BaseURL, result.BaseURL)
			assert.Equal(t, tt.expectedConfig.Logging, result.Logging)
			assert.Equal(t, tt.expectedConfig.LogLevel, result.LogLevel)
			assert.Equal(t, tt.expectedConfig.FileDB, result.FileDB)
		})
	}
}

// TestOverrideFromEnvDuration Проверяет переопределение time.Duration из ENV.
// Охватывает:
// - корректные записи длительности ("24h", "45m", "60s");
// - некорректные записи, которые не должны менять значение.
func TestOverrideFromEnvDuration(t *testing.T) {
	testTable := []struct {
		name           string
		initial        time.Duration
		envValue       string
		expected       time.Duration
		shouldOverride bool
	}{
		{
			name:           "Valid duration: 24h",
			initial:        1 * time.Hour,
			envValue:       "24h",
			expected:       24 * time.Hour,
			shouldOverride: true,
		},
		{
			name:           "Valid duration: 45m",
			initial:        1 * time.Hour,
			envValue:       "45m",
			expected:       45 * time.Minute,
			shouldOverride: true,
		},
		{
			name:           "Valid duration: 60s",
			initial:        1 * time.Hour,
			envValue:       "60s",
			expected:       60 * time.Second,
			shouldOverride: true,
		},
		{
			name:           "Invalid duration: should not override",
			initial:        1 * time.Hour,
			envValue:       "not-a-duration",
			expected:       1 * time.Hour,
			shouldOverride: false,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			envKey := "TOKEN_EXP_TEST"
			os.Setenv(envKey, tt.envValue)
			defer os.Unsetenv(envKey)

			d := tt.initial
			OverrideFromEnv(&d, envKey)

			if tt.shouldOverride {
				assert.Equal(t, tt.expected, d, "Expected duration to be overridden")
			} else {
				assert.Equal(t, tt.initial, d, "Expected duration to remain unchanged")
			}
		})
	}
}

// TestOverrideFromEnvBool Проверяет переопределение bool из ENV.
// Охватывает:
// - "true", "false", "True", "1", "0";
// - некорректные значения, которые не должны менять bool.
func TestOverrideFromEnvBool(t *testing.T) {
	testTable := []struct {
		name           string
		initial        bool
		envValue       string
		expected       bool
		shouldOverride bool
	}{
		{
			name:           "bool true: true",
			initial:        false,
			envValue:       "true",
			expected:       true,
			shouldOverride: true,
		},
		{
			name:           "bool false: false",
			initial:        true,
			envValue:       "false",
			expected:       false,
			shouldOverride: true,
		},
		{
			name:           "bool True (capital)",
			initial:        false,
			envValue:       "True",
			expected:       true,
			shouldOverride: true,
		},
		{
			name:           "bool 1 -> true",
			initial:        false,
			envValue:       "1",
			expected:       true,
			shouldOverride: true,
		},
		{
			name:           "bool 0 -> false",
			initial:        true,
			envValue:       "0",
			expected:       false,
			shouldOverride: true,
		},
		{
			name:           "Invalid bool: should not override",
			initial:        false,
			envValue:       "yes",
			expected:       false,
			shouldOverride: false,
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, func(t *testing.T) {
			envKey := "LOGGING_TEST"
			os.Setenv(envKey, tt.envValue)
			defer os.Unsetenv(envKey)

			b := tt.initial
			OverrideFromEnv(&b, envKey)

			if tt.shouldOverride {
				assert.Equal(t, tt.expected, b, "Expected bool to be overridden")
			} else {
				assert.Equal(t, tt.initial, b, "Expected bool to remain unchanged")
			}
		})
	}
}

// TestPriorityFileVsEnv Проверяет приоритет: ENV > файл.
// Сценарий:
// - флаг не задан;
// - файл устанавливает ServerAddress = "from-file";
// - ENV устанавливает ServerAddress = "from-env";
// Ожидаемо: ENV побеждает.
func TestPriorityFileVsEnv(t *testing.T) {
	envKey := "SERVER_ADDRESS"
	envValue := "from-env:1111"
	os.Setenv(envKey, envValue)
	defer os.Unsetenv(envKey)

	path := filepath.Join(os.TempDir(), "priority_test.json")
	fileCfg := Config{
		ServerAddress: "from-file:2222",
		BaseURL:       "http://file:2222",
	}
	data, err := json.Marshal(fileCfg)
	assert.NoError(t, err)

	err = os.WriteFile(path, data, 0o600)
	assert.NoError(t, err)
	defer os.Remove(path)

	// Явно выполняем цепочку: базовый конфиг → load из файла → override из ENV.
	base := &Config{}
	base, err = LoadConfigFromFile(base, path)
	assert.NoError(t, err)

	OverrideFromEnv(&base.ServerAddress, envKey)

	assert.Equal(t, "from-env:1111", base.ServerAddress, "ENV should override file value")
}

// TestPriorityFlagVsFile Проверяет приоритет: флаг > файл.
// Сценарий:
// - флаг -a задан как "from-flag:3333";
// - файл устанавливает ServerAddress = "from-file:2222";
// Ожидаемо: флаг побеждает, значение из файла игнорируется.
func TestPriorityFlagVsFile(t *testing.T) {
	resetFlags()

	// Регистрируем флаги так же, как в InitConfig.
	// Это нужно только для тестов, где мы не вызываем InitConfig.
	cfgFlag := &Config{}
	flag.StringVar(&cfgFlag.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.Set("a", "from-flag:3333")

	flag.Parse()

	path := filepath.Join(os.TempDir(), "priority_flag_file.json")
	fileCfg := Config{
		ServerAddress: "from-file:2222",
		BaseURL:       "http://file:2222",
	}
	data, err := json.Marshal(fileCfg)
	assert.NoError(t, err)

	err = os.WriteFile(path, data, 0o600)
	assert.NoError(t, err)
	defer os.Remove(path)

	//data, err = os.ReadFile(path)

	cfgFlag, err = LoadConfigFromFile(cfgFlag, path)
	assert.NoError(t, err)

	// после загрузки из файла ServerAddress должен остаться из флага
	assert.Equal(t, "from-flag:3333", cfgFlag.ServerAddress, "Flag should override file value")
}

package config

import (
	"flag"
	"log"
	"os"
	"strconv"
)

// Config Структура для хранения данных о конфигурации сервера.
type Config struct {
	ServerAddress string
	BaseURL       string
	Logging       bool
	LogLevel      string
	FileDB        string
	DatabaseDSN   string
}

// ServerConfig Указатель на структуру, получаемую после инициализации структуры в InitConfig()
// и нужен для того, чтобы передавать его в handlers.
var ServerConfig *Config

// InitConfig Функция инициализации структуры, содержащей конфигурацию сервера, полученную из флагов
// (либо конфигурацию по умолчанию).
func InitConfig() *Config {
	config := &Config{}
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&config.BaseURL, "b", "http://localhost:8080", "Base URL for shortened links")
	flag.BoolVar(&config.Logging, "l", true, "Enable or disable logging by -l=true or -l=false")
	flag.StringVar(&config.LogLevel, "ll", "Info", "Log level for logging (Debug, Info, Warn, Error, DPanic, Panic, Fatal)")
	flag.StringVar(&config.FileDB, "f", "/tmp/short-url-storage.json", "Set JSON file to save short URLs. To disable this feature - set empty key")
	flag.StringVar(&config.DatabaseDSN, "d", "", "Data source name for DB. Example for PostgreSQL: -d 'host=localhost port=5432 user=db_user password=xxxxx dbname=db_name sslmode=disable'")

	flag.Parse()

	OverrideFromEnv(&config.ServerAddress, "SERVER_ADDRESS")
	OverrideFromEnv(&config.BaseURL, "BASE_URL")
	OverrideFromEnv(&config.Logging, "LOGGING")
	OverrideFromEnv(&config.LogLevel, "LOG_LEVEL")
	OverrideFromEnv(&config.FileDB, "FILE_STORAGE_PATH")
	OverrideFromEnv(&config.DatabaseDSN, "DATABASE_DSN")

	ServerConfig = config

	return config
}

// OverrideFromEnv Если заданы переменные окружения и они не пустые - функция пытается
// переписать поля структуры данными, полученными из переменных окружения.
func OverrideFromEnv(structFiled interface{}, envKey string) {
	if value, ok := os.LookupEnv(envKey); ok && value != "" {
		switch structField := structFiled.(type) {
		// если тип structField - string
		case *string:
			*structField = value
		// если тип structField - bool
		case *bool:
			if parsedValue, err := strconv.ParseBool(value); err == nil {
				*structField = parsedValue
			}
		default:
			return
		}
	}
}

// GetServerConfig Функция получения инициализированной конфигурации.
// Если структура не инициализирована данными - отдает дефолтную конфигурацию сервера с записью в лог.
func GetServerConfig() *Config {
	if ServerConfig == nil {
		log.Println("Конфигурация сервера не инициализирована, используется конфигурация по умолчанию")
		defaultConfig := &Config{
			ServerAddress: "localhost:8080",
			BaseURL:       "http://localhost:8080",
			Logging:       true,
			LogLevel:      "Info",
			FileDB:        "/tmp/short-url-storage.json",
			DatabaseDSN:   "",
		}
		return defaultConfig
	}

	return ServerConfig
}

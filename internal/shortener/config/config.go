package config

import (
	"flag"
	"log"
)

type Config struct {
	ServerAddress string
	BaseURL       string
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

	flag.Parse()

	ServerConfig = config

	return config
}

// GetServerConfig Функция получения инициализированной конфигурации.
// Если структура не инициализирована данными - отдает дефолтную конфигурацию сервера с записью в лог.
func GetServerConfig() *Config {
	if ServerConfig == nil {
		log.Println("Конфигурация сервера не инициализирована, используется конфигурация по умолчанию")
		defaultConfig := &Config{ServerAddress: "localhost:8080", BaseURL: "http://localhost:8080"}
		return defaultConfig
	}

	return ServerConfig
}

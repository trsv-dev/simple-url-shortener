package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"time"
)

// Config Структура для хранения данных о конфигурации сервера.
type Config struct {
	ServerAddress string        `json:"server_address"`
	BaseURL       string        `json:"base_url"`
	Logging       bool          `json:"logging"`
	LogLevel      string        `json:"log_level"`
	FileDB        string        `json:"file_db"`
	DatabaseDSN   string        `json:"database_dsn"`
	SecretKey     string        `json:"secret_key"`
	TokenExp      time.Duration `json:"token_exp"`
	EnableHTTPS   bool          `json:"enable_https"`
	ConfigFile    string        `json:"-"`
	TrustedSubnet string        `json:"trusted_subnet"`
	EnableGRPC    bool          `json:"enable_grpc"`
	GRPCPort      string        `json:"grpc_port"`
}

// ServerConfig Указатель на структуру, получаемую после инициализации структуры в InitConfig()
// и нужен для того, чтобы передавать его в handlers.
var ServerConfig *Config

// InitConfig Функция инициализации структуры, содержащей конфигурацию сервера, полученную из флагов
// (либо конфигурацию по умолчанию).
func InitConfig() *Config {
	config := &Config{}

	// получаем данные из флагов
	flag.StringVar(&config.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&config.BaseURL, "b", "http://localhost:8080", "Base URL for shortened links")
	flag.BoolVar(&config.Logging, "l", true, "Enable or disable logging by -l=true or -l=false")
	flag.StringVar(&config.LogLevel, "ll", "Info", "Log level for logging (Debug, Info, Warn, Error, DPanic, Panic, Fatal)")
	flag.StringVar(&config.FileDB, "f", "/tmp/short-url-storage.json", "Set JSON file to save short URLs. To disable this feature - set empty key")
	flag.StringVar(&config.DatabaseDSN, "d", "", "Data source name for DB. Example for PostgreSQL: -d 'host=localhost port=5432 user=db_user password=xxxxx dbname=db_name sslmode=disable'")
	flag.StringVar(&config.SecretKey, "k", "Change_Me_On_Production", "Secret key for signing JWT token. Example: UIfuBer431qY1crEUgzIem9")
	flag.DurationVar(&config.TokenExp, "e", 24*time.Hour, "JWT token expiration time. Example: 24h, 45m, 60s, etc")
	flag.BoolVar(&config.EnableHTTPS, "s", false, "Enable or disable HTTPS by -s=true or -s=false. Default: false")
	flag.StringVar(&config.ConfigFile, "c", "", "Set JSON file to load system configuration. Default: file not set")
	flag.StringVar(&config.TrustedSubnet, "t", "", "trusted subnet in CIDR notation. Example: 127.0.0.0/8. Default: 127.0.0.1/8")
	flag.BoolVar(&config.EnableGRPC, "g", false, "Enable or disable running the gRPC server. Default: false")
	flag.StringVar(&config.GRPCPort, "gp", "9090", "Port for the gRPC server. Default: 9090")
	flag.Parse()

	// переопределяем имя файла конфигурации из ENV (приоритет выше флага)
	OverrideFromEnv(&config.ConfigFile, "CONFIG")

	// если файл конфигурации задан (через флаг или ENV), загружаем его;
	// накладываем на текущий конфиг данные из файла конфигурации;
	// меняем только те значения, которые были установлены, а не являлись значениями по умолчанию.
	var fileErr error
	if config.ConfigFile != "" {
		config, fileErr = LoadConfigFromFile(config, config.ConfigFile)
		if fileErr != nil {
			log.Fatal(fileErr)
		}
	}

	// переопределяем все остальные параметры из ENV (наивысший приоритет)
	OverrideFromEnv(&config.ServerAddress, "SERVER_ADDRESS")
	OverrideFromEnv(&config.BaseURL, "BASE_URL")
	OverrideFromEnv(&config.Logging, "LOGGING")
	OverrideFromEnv(&config.LogLevel, "LOG_LEVEL")
	OverrideFromEnv(&config.FileDB, "FILE_STORAGE_PATH")
	OverrideFromEnv(&config.DatabaseDSN, "DATABASE_DSN")
	OverrideFromEnv(&config.SecretKey, "SECRET_KEY")
	OverrideFromEnv(&config.TokenExp, "TOKEN_EXP")
	OverrideFromEnv(&config.EnableHTTPS, "ENABLE_HTTPS")
	OverrideFromEnv(&config.TrustedSubnet, "TRUSTED_SUBNET")
	OverrideFromEnv(&config.EnableGRPC, "ENABLE_GRPC")
	OverrideFromEnv(&config.GRPCPort, "GRPC_PORT")

	ServerConfig = config

	return config
}

// GetConfigFile Возвращает конфигурацию сервера на основе переданного файла конфигурации.
func GetConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadConfigFromFile(config *Config, path string) (*Config, error) {
	fileConfig, err := GetConfigFile(path)
	if err != nil {
		return nil, err
	}

	usedFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		usedFlags[f.Name] = true
	})

	// применяем значения из файла только если:
	// - флаг не был передан явно (!usedFlags)
	// - значение в файле не пустое (!= "")

	if !usedFlags["a"] && fileConfig.ServerAddress != "" {
		config.ServerAddress = fileConfig.ServerAddress
	}

	if !usedFlags["b"] && fileConfig.BaseURL != "" {
		config.BaseURL = fileConfig.BaseURL
	}

	if !usedFlags["l"] && !fileConfig.Logging {
		config.Logging = fileConfig.Logging
	}

	if !usedFlags["ll"] && fileConfig.LogLevel != "" {
		config.LogLevel = fileConfig.LogLevel
	}

	if !usedFlags["f"] && fileConfig.FileDB != "" {
		config.FileDB = fileConfig.FileDB
	}

	if !usedFlags["d"] && fileConfig.DatabaseDSN != "" {
		config.DatabaseDSN = fileConfig.DatabaseDSN
	}

	if !usedFlags["k"] && fileConfig.SecretKey != "" {
		config.SecretKey = fileConfig.SecretKey
	}

	if !usedFlags["e"] && fileConfig.TokenExp != 0 {
		config.TokenExp = fileConfig.TokenExp
	}

	if !usedFlags["s"] && fileConfig.EnableHTTPS {
		config.EnableHTTPS = fileConfig.EnableHTTPS
	}

	if !usedFlags["t"] && fileConfig.TrustedSubnet != "" {
		config.TrustedSubnet = fileConfig.TrustedSubnet
	}

	if !usedFlags["g"] && !fileConfig.EnableGRPC {
		config.EnableGRPC = fileConfig.EnableGRPC
	}

	if !usedFlags["gp"] && fileConfig.GRPCPort != "" {
		config.GRPCPort = fileConfig.GRPCPort
	}

	return config, nil
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
		// если тип structField - time.Duration
		case *time.Duration:
			if duration, err := time.ParseDuration(value); err == nil {
				*structField = duration
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
			SecretKey:     "Change_Me_On_Production",
			TokenExp:      24 * time.Hour,
			EnableHTTPS:   false,
			TrustedSubnet: "127.0.0.1/8",
			EnableGRPC:    false,
			GRPCPort:      "9090",
		}
		return defaultConfig
	}

	return ServerConfig
}

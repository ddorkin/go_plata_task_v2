package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config содержит конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	External ExternalConfig
	Worker   WorkerConfig
	Logging  LoggingConfig
	App      AppConfig
}

// ServerConfig содержит настройки сервера
type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig содержит настройки базы данных
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// ExternalConfig содержит настройки внешнего API
type ExternalConfig struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// WorkerConfig содержит настройки фонового воркера
type WorkerConfig struct {
	Interval time.Duration
}

// LoggingConfig содержит настройки логирования
type LoggingConfig struct {
	Level  string
	Format string
}

// AppConfig содержит общие настройки приложения
type AppConfig struct {
	ShutdownTimeout     time.Duration
	SupportedCurrencies []string
}

// Load загружает конфигурацию из переменных окружения
// Сначала пытается загрузить .env файл, затем использует системные env vars
func Load() *Config {
	// Пытаемся загрузить .env файл (игнорируем ошибку, если файл не найден)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using system environment variables: %v", err)
	}
	return &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "localhost"),
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "currency_quotes"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		External: ExternalConfig{
			APIKey:  getEnv("EXTERNAL_API_KEY", ""),
			BaseURL: getEnv("EXTERNAL_API_URL", "https://api.fxratesapi.com"),
			Timeout: getDurationEnv("EXTERNAL_API_TIMEOUT", 10*time.Second),
		},
		Worker: WorkerConfig{
			Interval: getDurationEnv("WORKER_INTERVAL", 30*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		App: AppConfig{
			ShutdownTimeout:     getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
			SupportedCurrencies: getStringSliceEnv("SUPPORTED_CURRENCIES", []string{"USD", "EUR", "MXN"}),
		},
	}
}

// getEnv получает значение переменной окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDurationEnv получает значение переменной окружения как duration или возвращает значение по умолчанию
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getIntEnv получает значение переменной окружения как int или возвращает значение по умолчанию
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getStringSliceEnv получает значение переменной окружения как slice строк или возвращает значение по умолчанию
func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

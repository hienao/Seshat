package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	DatabaseURL       string
	DBDriver          string
	SQLitePath        string
	AppDataDir        string
	AppCacheDir       string
	LogDir            string
	JWTSecret         string
	APILogEnabled     bool
	APILogPath        string
	APILogQueueSize   int
	APILogBatchSize   int
	APILogExportLimit int
	ServerPort        string
	GinMode           string
}

// Load 加载配置
func Load() *Config {
	const appDataDir = "/data"
	const appCacheDir = "/cache"
	const serverPort = "8080"

	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		DBDriver:          getEnv("DB_DRIVER", "sqlite"),
		SQLitePath:        getEnv("SQLITE_PATH", filepath.Join(appDataDir, "db", "seshat.db")),
		AppDataDir:        appDataDir,
		AppCacheDir:       appCacheDir,
		LogDir:            filepath.Join(appCacheDir, "logs", "app"),
		JWTSecret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		APILogEnabled:     true,
		APILogPath:        filepath.Join(appCacheDir, "logs", "app", "api-logs.db"),
		APILogQueueSize:   getEnvInt("API_LOG_QUEUE_SIZE", 5000),
		APILogBatchSize:   getEnvInt("API_LOG_BATCH_SIZE", 100),
		APILogExportLimit: getEnvInt("API_LOG_EXPORT_LIMIT", 100000),
		ServerPort:        serverPort,
		GinMode:           "release",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

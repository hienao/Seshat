package config

import (
	"os"
)

// Config 应用配置
type Config struct {
	DatabaseURL string
	DBDriver    string
	SQLitePath  string
	DataDir     string
	CacheDir    string
	LogDir      string
	JWTSecret   string
	ServerPort  string
	GinMode     string
}

// Load 加载配置
func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DBDriver:    getEnv("DB_DRIVER", "sqlite"),
		SQLitePath:  getEnv("SQLITE_PATH", "/data/db/basegoapp.db"),
		DataDir:     getEnv("DATA_DIR", "/data"),
		CacheDir:    getEnv("CACHE_DIR", "/cache"),
		LogDir:      getEnv("LOG_DIR", "/cache/logs/app"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

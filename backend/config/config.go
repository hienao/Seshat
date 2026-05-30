package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 应用配置
type Config struct {
	DatabaseURL          string
	DBDriver             string
	SQLitePath           string
	AppDataDir           string
	AppCacheDir          string
	LogDir               string
	JWTSecret            string
	ServerPort           string
	GinMode              string
	CORSAllowedOrigins   []string
	AuthCookieName       string
	AuthCookieSecure     bool
	DefaultAdminUsername string
	DefaultAdminPassword string
}

// Load 加载配置
func Load() *Config {
	const appDataDir = "/data"
	const appCacheDir = "/cache"
	const serverPort = "8080"

	return &Config{
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		DBDriver:             getEnv("DB_DRIVER", "sqlite"),
		SQLitePath:           filepath.Join(appDataDir, "db", "basegoapp.db"),
		AppDataDir:           appDataDir,
		AppCacheDir:          appCacheDir,
		LogDir:               filepath.Join(appCacheDir, "logs", "app"),
		JWTSecret:            getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		ServerPort:           serverPort,
		GinMode:              getEnv("GIN_MODE", "debug"),
		CORSAllowedOrigins:   parseCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost,http://127.0.0.1,http://localhost:3000,http://127.0.0.1:3000")),
		AuthCookieName:       getEnv("AUTH_COOKIE_NAME", "auth_token"),
		AuthCookieSecure:     getEnvBool("AUTH_COOKIE_SECURE", false),
		DefaultAdminUsername: getEnv("DEFAULT_ADMIN_USERNAME", ""),
		DefaultAdminPassword: getEnv("DEFAULT_ADMIN_PASSWORD", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

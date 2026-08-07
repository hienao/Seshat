package database

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"seshat/config"
	"seshat/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.Config) {
	var err error

	logLevel := logger.Info
	if cfg.GinMode == "release" {
		logLevel = logger.Error
	}

	driver := strings.ToLower(strings.TrimSpace(cfg.DBDriver))
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	switch driver {
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0755); err != nil {
			log.Fatalf("Failed to create sqlite directory: %v", err)
		}
		DB, err = gorm.Open(sqlite.Open(cfg.SQLitePath), gormConfig)
	case "postgres", "postgresql":
		if cfg.DatabaseURL == "" {
			log.Fatal("DATABASE_URL is required when DB_DRIVER=postgres")
		}
		DB, err = gorm.Open(postgres.Open(cfg.DatabaseURL), gormConfig)
	default:
		log.Fatalf("Unsupported DB_DRIVER %q, expected sqlite or postgres", cfg.DBDriver)
	}
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移
	if err := DB.AutoMigrate(&model.User{}, &model.SystemSetting{}, &model.UserSetting{}, &model.NotificationChannel{}, &model.AppIntegration{}, &model.WebhookEvent{}, &model.IntegrationNotificationRule{}, &model.NotificationDelivery{}, &model.AdminAuditLog{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Printf("Database connected and migrated successfully using %s", driver)
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

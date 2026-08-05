package main

import (
	"log"
	"strings"

	"basegoapp/config"
	_ "basegoapp/docs"
	"basegoapp/internal/router"
	"basegoapp/pkg/database"
)

// @title Seshat API
// @version 1.0
// @description Seshat 模板工程 API 文档

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {token}

func main() {
	// 加载配置
	cfg := config.Load()
	validateSecurityConfig(cfg)

	// 初始化数据库
	database.Init(cfg)

	// 设置路由
	r := router.Setup(cfg)

	// 启动服务器
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func validateSecurityConfig(cfg *config.Config) {
	if strings.ToLower(cfg.GinMode) != "release" {
		return
	}

	if len(cfg.JWTSecret) < 32 || cfg.JWTSecret == "your-secret-key-change-in-production" {
		log.Fatal("In release mode, JWT_SECRET must be at least 32 chars and cannot use default value")
	}
}

package main

import (
	"log"

	"basegoapp/config"
	_ "basegoapp/docs"
	"basegoapp/internal/router"
	"basegoapp/pkg/database"
)

// @title BaseGoApp API
// @version 1.0
// @description BaseGoApp 模板工程 API 文档

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {token}

func main() {
	// 加载配置
	cfg := config.Load()

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

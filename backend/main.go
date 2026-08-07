package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"seshat/config"
	_ "seshat/docs"
	"seshat/internal/logging"
	"seshat/internal/router"
	"seshat/internal/service"
	"seshat/pkg/database"
)

// @title Seshat API
// @version 1.0
// @description Seshat Webhook 消息管理 API 文档

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

	// 设置日志管理器和路由
	logManager, err := logging.NewManager(cfg)
	if err != nil {
		log.Fatalf("Failed to init log manager: %v", err)
	}
	defer logManager.Close()
	logging.SetDefaultManager(logManager)
	defer logging.SetDefaultManager(nil)
	r := router.SetupWithLogManager(cfg, logManager)
	notificationWorker := service.NewNotificationWorker()
	notificationWorker.Start()
	defer notificationWorker.Close()

	// 启动服务器
	logging.Info("server", "Seshat 服务启动", logging.Fields{"port": cfg.ServerPort})
	server := &http.Server{Addr: ":" + cfg.ServerPort, Handler: r}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			logging.Error("server", "HTTP 服务启动失败", logging.Fields{"error": err})
			log.Fatalf("Failed to start server: %v", err)
		}
	case <-stop:
		logging.Info("server", "收到退出信号，开始优雅关闭", nil)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logging.Error("server", "HTTP 服务优雅关闭失败", logging.Fields{"error": err})
		}
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

package router

import (
	"seshat/config"
	"seshat/internal/handler"
	"seshat/internal/logging"
	"seshat/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Setup 设置路由
func Setup(cfg *config.Config) *gin.Engine {
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	logManager, err := logging.NewManager(cfg)
	if err != nil {
		panic("Failed to init API log manager: " + err.Error())
	}
	return SetupWithLogManager(cfg, logManager)
}

// SetupWithLogManager 使用调用方创建的日志管理器组装路由，便于主程序优雅退出时刷新日志队列。
func SetupWithLogManager(cfg *config.Config, logManager *logging.Manager) *gin.Engine {
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(logging.RequestIDMiddleware(), logging.AccessLogMiddleware(logManager), gin.Recovery())

	// CORS 中间件
	r.Use(corsMiddleware())

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 创建处理器
	authHandler := handler.NewAuthHandler(cfg)
	userHandler := handler.NewUserHandler(cfg)
	settingHandler := handler.NewSettingHandler(logManager)
	adminHandler := handler.NewAdminHandler(authHandler.GetAuthService())
	adminLogHandler := handler.NewAdminLogHandler(logManager)
	adminApplicationLogHandler := handler.NewAdminApplicationLogHandler(logManager)
	webhookHandler := handler.NewWebhookHandler()
	notificationHandler := handler.NewNotificationHandler()

	// 初始化一次性引导管理员和系统设置
	if err := authHandler.GetAuthService().InitBootstrapAdmin(); err != nil {
		panic("Failed to init bootstrap admin: " + err.Error())
	}
	if err := settingHandler.GetSettingService().InitDefaultSettings(); err != nil {
		panic("Failed to init default settings: " + err.Error())
	}
	// API 路由组
	api := r.Group("/api")
	{
		// 推送消息中的公开详情链接，使用随机访问标识，不需要 JWT。
		api.GET("/public/events/:token", webhookHandler.GetPublicEvent)

		// Webhook 管理接口需要认证，实际接收接口在 /hooks 下公开提供。
		webhookAPI := api.Group("/webhooks")
		webhookAPI.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete())
		{
			webhookAPI.GET("/apps", webhookHandler.Catalog)
			webhookAPI.GET("/integrations", webhookHandler.ListIntegrations)
			webhookAPI.POST("/integrations", webhookHandler.CreateIntegration)
			webhookAPI.GET("/integrations/:id/secret", webhookHandler.GetIntegrationSecret)
			webhookAPI.POST("/integrations/:id/rotate-secret", webhookHandler.RotateSecret)
			webhookAPI.GET("/integrations/:id/notification-settings", notificationHandler.GetIntegrationSettings)
			webhookAPI.PUT("/integrations/:id/notification-settings", notificationHandler.UpdateIntegrationSettings)
			webhookAPI.GET("/events", webhookHandler.ListEvents)
			webhookAPI.GET("/events/:id", webhookHandler.GetEvent)
			webhookAPI.GET("/events/:id/notification-status", notificationHandler.GetEventStatus)
		}

		notifications := api.Group("/notification-channels")
		notifications.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete())
		{
			notifications.GET("", notificationHandler.ListChannels)
			notifications.POST("", notificationHandler.CreateChannel)
			notifications.PUT("/:id", notificationHandler.UpdateChannel)
			notifications.DELETE("/:id", notificationHandler.DeleteChannel)
			notifications.POST("/:id/test", notificationHandler.TestChannel)
		}

		deliveries := api.Group("/notifications/deliveries")
		deliveries.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete())
		{
			deliveries.GET("", notificationHandler.ListDeliveries)
			deliveries.POST("/:id/retry", notificationHandler.RetryDelivery)
		}
		// 公开路由
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", middleware.JWTAuth(cfg), authHandler.Logout)
			auth.POST("/setup-admin", middleware.JWTAuth(cfg), authHandler.SetupAdmin)
		}

		// 公开设置路由
		settings := api.Group("/settings")
		{
			settings.GET("/registration-status", settingHandler.GetRegistrationStatus)
		}

		// 需要认证的路由
		user := api.Group("/user")
		user.Use(middleware.JWTAuth(cfg))
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/password", middleware.AdminSetupComplete(), userHandler.ChangePassword)
		}

		// 需要管理员权限的设置路由
		adminSettings := api.Group("/settings")
		adminSettings.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete(), middleware.AdminAuth())
		{
			adminSettings.GET("/system", settingHandler.GetSystemSettings)
			adminSettings.PUT("/system", settingHandler.UpdateSystemSettings)
		}

		// 管理员接口日志，不记录日志管理接口自身，清空操作写入审计表。
		adminLogs := api.Group("/admin/logs")
		adminLogs.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete(), middleware.AdminAuth())
		{
			adminLogs.GET("", adminLogHandler.List)
			adminLogs.GET("/summary", adminLogHandler.Summary)
			adminLogs.GET("/export", adminLogHandler.Export)
			adminLogs.POST("/clear", adminLogHandler.Clear)
			adminLogs.GET("/:id", adminLogHandler.Get)
		}

		applicationLogs := api.Group("/admin/application-logs")
		applicationLogs.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete(), middleware.AdminAuth())
		{
			applicationLogs.GET("", adminApplicationLogHandler.List)
			applicationLogs.GET("/summary", adminApplicationLogHandler.Summary)
			applicationLogs.GET("/export", adminApplicationLogHandler.Export)
			applicationLogs.POST("/clear", adminApplicationLogHandler.Clear)
			applicationLogs.GET("/:id", adminApplicationLogHandler.Get)
		}

		// 管理员路由
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(cfg), middleware.AdminSetupComplete(), middleware.AdminAuth())
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.PUT("/users/:id/role", adminHandler.SetUserRole)
		}
	}

	// 外部 App 调用的公开 Webhook 接收接口，不使用 JWT。
	r.POST("/hooks/v1/:endpointKey", webhookHandler.Receive)

	return r
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

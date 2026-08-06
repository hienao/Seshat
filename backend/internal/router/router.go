package router

import (
	"basegoapp/config"
	"basegoapp/internal/handler"
	"basegoapp/internal/logging"
	"basegoapp/internal/middleware"
	"net/http"

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
	r.Use(corsMiddleware(cfg))

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 创建处理器
	authHandler := handler.NewAuthHandler(cfg)
	userHandler := handler.NewUserHandler(cfg)
	settingHandler := handler.NewSettingHandler()
	adminHandler := handler.NewAdminHandler(authHandler.GetAuthService())
	webhookHandler := handler.NewWebhookHandler(cfg)
	adminLogHandler := handler.NewAdminLogHandler(logManager)

	// 初始化默认管理员和系统设置
	if err := authHandler.GetAuthService().InitDefaultAdmin(); err != nil {
		panic("Failed to init default admin: " + err.Error())
	}
	if err := settingHandler.GetSettingService().InitDefaultSettings(); err != nil {
		panic("Failed to init default settings: " + err.Error())
	}

	// API 路由组
	api := r.Group("/api")
	{
		// Webhook 管理接口需要认证，实际接收接口在 /hooks 下公开提供。
		webhookAPI := api.Group("/webhooks")
		webhookAPI.Use(middleware.JWTAuth(cfg))
		{
			webhookAPI.GET("/apps", webhookHandler.Catalog)
			webhookAPI.GET("/integrations", webhookHandler.ListIntegrations)
			webhookAPI.POST("/integrations", webhookHandler.CreateIntegration)
			webhookAPI.POST("/integrations/:id/rotate-secret", webhookHandler.RotateSecret)
			webhookAPI.GET("/events", webhookHandler.ListEvents)
			webhookAPI.GET("/events/:id", webhookHandler.GetEvent)
		}
		// 公开路由
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", middleware.JWTAuth(cfg), authHandler.Logout)
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
			user.PUT("/password", userHandler.ChangePassword)
		}

		// 需要管理员权限的设置路由
		adminSettings := api.Group("/settings")
		adminSettings.Use(middleware.JWTAuth(cfg), middleware.AdminAuth())
		{
			adminSettings.GET("/system", settingHandler.GetSystemSettings)
			adminSettings.PUT("/system", settingHandler.UpdateSystemSettings)
		}

		// 管理员接口日志，不记录日志管理接口自身，清空操作写入审计表。
		adminLogs := api.Group("/admin/logs")
		adminLogs.Use(middleware.JWTAuth(cfg), middleware.AdminAuth())
		{
			adminLogs.GET("", adminLogHandler.List)
			adminLogs.GET("/summary", adminLogHandler.Summary)
			adminLogs.GET("/export", adminLogHandler.Export)
			adminLogs.POST("/clear", adminLogHandler.Clear)
			adminLogs.GET("/:id", adminLogHandler.Get)
		}

		// 管理员路由
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(cfg), middleware.AdminAuth())
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
func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, origin := range cfg.CORSAllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowedOrigins[origin]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code":    -1,
					"message": "Origin 不在 CORS 白名单中",
				})
				return
			}
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
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

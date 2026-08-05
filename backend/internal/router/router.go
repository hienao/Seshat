package router

import (
	"basegoapp/config"
	"basegoapp/internal/handler"
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

	r := gin.Default()

	// CORS 中间件
	r.Use(corsMiddleware(cfg))

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 创建处理器
	authHandler := handler.NewAuthHandler(cfg)
	userHandler := handler.NewUserHandler(cfg)
	settingHandler := handler.NewSettingHandler()
	adminHandler := handler.NewAdminHandler(authHandler.GetAuthService())

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

		// 管理员路由
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(cfg), middleware.AdminAuth())
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.PUT("/users/:id/role", adminHandler.SetUserRole)
		}
	}

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

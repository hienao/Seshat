package router

import (
	"basegoapp/config"
	"basegoapp/internal/handler"
	"basegoapp/internal/middleware"

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
	r.Use(corsMiddleware())

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 创建处理器
	authHandler := handler.NewAuthHandler(cfg)
	userHandler := handler.NewUserHandler(cfg)

	// 初始化默认管理员
	if err := authHandler.GetAuthService().InitDefaultAdmin(); err != nil {
		panic("Failed to init default admin: " + err.Error())
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

		// 需要认证的路由
		user := api.Group("/user")
		user.Use(middleware.JWTAuth(cfg))
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/password", userHandler.ChangePassword)
		}
	}

	return r
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

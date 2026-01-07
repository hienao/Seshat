package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员权限中间件
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    -1,
				"message": "需要管理员权限",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

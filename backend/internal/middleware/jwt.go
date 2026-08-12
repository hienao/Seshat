package middleware

import (
	"net/http"
	"strings"

	"seshat/config"
	"seshat/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth JWT 认证中间件
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	userRepo := repository.NewUserRepository()

	return func(c *gin.Context) {
		// 仅接受显式的 Bearer Token，不从 Cookie 读取认证信息。
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    -1,
				"message": "未提供认证令牌",
			})
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    -1,
				"message": "认证令牌格式错误",
			})
			c.Abort()
			return
		}
		tokenString := strings.TrimSpace(parts[1])

		// 解析 JWT Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    -1,
				"message": "认证令牌无效或已过期",
			})
			c.Abort()
			return
		}

		// 提取用户信息并与数据库中的认证版本比对，实现旧 token 失效。
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userIDValue, ok := claims["user_id"].(float64)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "认证令牌缺少用户信息"})
				c.Abort()
				return
			}

			tokenVersionValue, ok := claims["token_version"].(float64)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "认证令牌版本无效"})
				c.Abort()
				return
			}

			userID := uint(userIDValue)
			user, dbErr := userRepo.FindAuthVersionByID(userID)
			if dbErr != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "用户不存在或认证已失效"})
				c.Abort()
				return
			}

			if int(tokenVersionValue) != user.TokenVersion {
				c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "认证令牌已失效，请重新登录"})
				c.Abort()
				return
			}

			c.Set("user_id", userID)
			c.Set("username", user.Username)
			c.Set("is_admin", user.IsAdmin)
			c.Set("requires_admin_setup", user.RequiresAdminSetup)
			c.Set("token_version", user.TokenVersion)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "认证令牌无效"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminSetupComplete 禁止一次性引导管理员访问初始化以外的受保护接口。
func AdminSetupComplete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetBool("requires_admin_setup") {
			c.JSON(http.StatusForbidden, gin.H{"code": -1, "message": "请先完成管理员初始化"})
			c.Abort()
			return
		}
		c.Next()
	}
}

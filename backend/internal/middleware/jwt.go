package middleware

import (
	"net/http"
	"strings"

	"basegoapp/config"
	"basegoapp/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuth JWT 认证中间件
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	userRepo := repository.NewUserRepository()

	return func(c *gin.Context) {
		// 优先从 Authorization Header 获取 Token，兼容 Cookie 模式。
		tokenString := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    -1,
					"message": "认证令牌格式错误",
				})
				c.Abort()
				return
			}
			tokenString = parts[1]
		}

		if tokenString == "" {
			if cookieValue, err := c.Cookie(cfg.AuthCookieName); err == nil {
				tokenString = cookieValue
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    -1,
				"message": "未提供认证令牌",
			})
			c.Abort()
			return
		}

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
			c.Set("token_version", user.TokenVersion)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "message": "认证令牌无效"})
			c.Abort()
			return
		}

		c.Next()
	}
}

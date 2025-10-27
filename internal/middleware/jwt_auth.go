package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tempmail/backend/internal/auth/jwt"
)

// JWTAuth JWT认证中间件
type JWTAuth struct {
	jwtManager *jwt.Manager
	log        *zap.Logger
}

// NewJWTAuth 创建JWT认证中间件
func NewJWTAuth(jwtManager *jwt.Manager) *JWTAuth {
	return &JWTAuth{
		jwtManager: jwtManager,
		log:        zap.NewNop(), // 临时使用空日志
	}
}

// RequireAuth 要求JWT认证
func (ja *JWTAuth) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ja.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		claims, err := ja.jwtManager.ValidateToken(token)
		if err != nil {
			ja.log.Warn("invalid token",
				zap.String("error", err.Error()),
				zap.String("ip", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("tier", claims.Tier)

		c.Next()
	}
}

// OptionalAuth 可选的JWT认证
func (ja *JWTAuth) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ja.extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := ja.jwtManager.ValidateToken(token)
		if err == nil {
			c.Set("userID", claims.UserID)
			c.Set("email", claims.Email)
			c.Set("tier", claims.Tier)
			c.Set("authenticated", true)
		}

		c.Next()
	}
}

// extractToken 从请求中提取JWT token
func (ja *JWTAuth) extractToken(c *gin.Context) string {
	// 1. 从 Authorization header 提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// 2. 从 cookie 提取
	token, err := c.Cookie("access_token")
	if err == nil && token != "" {
		return token
	}

	return ""
}

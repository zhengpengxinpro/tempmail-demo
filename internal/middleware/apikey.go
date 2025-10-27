package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tempmail/backend/internal/service"
)

// APIKeyAuth API Key认证中间件
type APIKeyAuth struct {
	apiKeyService *service.APIKeyService
}

// NewAPIKeyAuth 创建API Key认证中间件
func NewAPIKeyAuth(apiKeyService *service.APIKeyService) *APIKeyAuth {
	return &APIKeyAuth{
		apiKeyService: apiKeyService,
	}
}

// RequireAPIKey 要求API Key认证
func (m *APIKeyAuth) RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing API key",
			})
			c.Abort()
			return
		}

		// 验证API Key并自动更新最后使用时间
		user, err := m.apiKeyService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			c.Abort()
			return
		}

		// 将用户ID存入上下文
		c.Set("userID", user.ID)
		c.Set("user", user)

		c.Next()
	}
}

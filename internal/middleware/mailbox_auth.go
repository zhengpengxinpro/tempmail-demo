package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tempmail/backend/internal/service"
)

// MailboxAuth 邮箱Token认证中间件
type MailboxAuth struct {
	mailboxService *service.MailboxService
	log            *zap.Logger
}

// NewMailboxAuth 创建邮箱认证中间件
func NewMailboxAuth(mailboxService *service.MailboxService) *MailboxAuth {
	return &MailboxAuth{
		mailboxService: mailboxService,
		log:            zap.NewNop(), // 临时使用空日志
	}
}

// RequireMailboxToken 要求邮箱Token验证
func (ma *MailboxAuth) RequireMailboxToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		mailboxID := c.Param("id")
		if mailboxID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "mailbox ID required",
			})
			c.Abort()
			return
		}

		// 从多个来源提取Token
		token := ma.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "mailbox token required",
			})
			c.Abort()
			return
		}

		// 获取邮箱并验证Token
		mailbox, err := ma.mailboxService.Get(mailboxID)
		if err != nil {
			ma.log.Warn("mailbox not found",
				zap.String("mailbox_id", mailboxID),
				zap.Error(err),
			)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "mailbox not found",
			})
			c.Abort()
			return
		}

		// 验证Token
		if mailbox.Token != token {
			ma.log.Warn("invalid mailbox token",
				zap.String("mailbox_id", mailboxID),
				zap.String("ip", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid mailbox token",
			})
			c.Abort()
			return
		}

		// 将邮箱信息存储到上下文中
		c.Set("mailbox", mailbox)
		c.Next()
	}
}

// extractToken 从多个来源提取Token
func (ma *MailboxAuth) extractToken(c *gin.Context) string {
	// 1. 尝试从 Authorization header 提取 (Bearer token格式)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// 2. 尝试从 X-Mailbox-Token header 提取
	token := c.GetHeader("X-Mailbox-Token")
	if token != "" {
		return token
	}

	// 3. 尝试从 query parameter 提取
	token = c.Query("token")
	if token != "" {
		return token
	}

	return ""
}

// OptionalMailboxToken 可选的邮箱Token验证（如果提供则验证，不提供则跳过）
func (ma *MailboxAuth) OptionalMailboxToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		mailboxID := c.Param("id")
		token := ma.extractToken(c)

		// 如果没有提供Token，直接放行
		if token == "" {
			c.Next()
			return
		}

		// 如果提供了Token，则必须验证通过
		if mailboxID != "" {
			mailbox, err := ma.mailboxService.Get(mailboxID)
			if err == nil && mailbox.Token == token {
				c.Set("mailbox", mailbox)
				c.Set("authenticated", true)
			}
		}

		c.Next()
	}
}

package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecurityHeaders 添加安全响应头
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止 XSS 攻击
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")

		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'")

		// HTTPS 严格传输安全
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 引荐来源策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	log := zap.NewNop() // 临时使用空日志

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		// 记录请求日志
		duration := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("duration", duration),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// 如果有用户信息，添加到日志
		if userID, exists := c.Get("userID"); exists {
			fields = append(fields, zap.String("user_id", userID.(string)))
		}

		// 根据状态码选择日志级别
		switch {
		case status >= 500:
			log.Error("server error", fields...)
		case status >= 400:
			log.Warn("client error", fields...)
		case status >= 300:
			log.Info("redirect", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// RequestSizeLimit 请求大小限制中间件
func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// Timeout 请求超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 使用 channel 等待处理完成或超时
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "request timeout",
			})
			c.Abort()
		}
	}
}

// IPWhitelist IP 白名单中间件
func IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	allowedMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowedMap[ip] = true
	}

	log := zap.NewNop() // 临时使用空日志

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !allowedMap[clientIP] {
			log.Warn("IP not in whitelist", zap.String("ip", clientIP))
			c.JSON(http.StatusForbidden, gin.H{
				"error": "access denied",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateContentType 验证 Content-Type 中间件
func ValidateContentType(allowedTypes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅对 POST/PUT/PATCH 请求验证
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if contentType == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "missing Content-Type header",
			})
			c.Abort()
			return
		}

		// 检查 Content-Type 是否在允许列表中
		allowed := false
		for _, allowedType := range allowedTypes {
			if strings.HasPrefix(contentType, allowedType) {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error": "unsupported Content-Type",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	log := zap.NewNop() // 临时使用空日志

	return func(c *gin.Context) {
		c.Next()

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			log.Error("request error",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(err.Err),
			)

			// 如果还没有响应，返回 500
			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}
	}
}

// RecoveryHandler 恢复 panic 的中间件
func RecoveryHandler() gin.HandlerFunc {
	log := zap.NewNop() // 临时使用空日志

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Any("error", err),
					zap.Stack("stack"),
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

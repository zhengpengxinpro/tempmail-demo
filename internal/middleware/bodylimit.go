package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// 默认请求体大小限制
	DefaultBodyLimit = 10 * 1024 * 1024 // 10MB

	// 不同类型请求的限制
	SmallBodyLimit  = 1 * 1024 * 1024  // 1MB - 用于普通API请求
	MediumBodyLimit = 5 * 1024 * 1024  // 5MB - 用于文件上传
	LargeBodyLimit  = 20 * 1024 * 1024 // 20MB - 用于大文件上传
)

// BodySizeLimit 限制请求体大小的中间件
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查 Content-Length 头
		if c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request body too large",
				"message": fmt.Sprintf("Request body exceeds maximum size of %d bytes", maxBytes),
				"limit":   maxBytes,
				"size":    c.Request.ContentLength,
			})
			c.Abort()
			return
		}

		// 限制请求体读取大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// 设置响应头，告知客户端最大允许的请求体大小
		c.Header("X-Max-Body-Size", strconv.FormatInt(maxBytes, 10))

		c.Next()

		// 检查是否因为请求体过大而产生错误
		if c.Errors != nil {
			for _, err := range c.Errors {
				if err.Err != nil && err.Err.Error() == "http: request body too large" {
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{
						"error":   "Request body too large",
						"message": fmt.Sprintf("Request body exceeds maximum size of %d bytes", maxBytes),
						"limit":   maxBytes,
					})
					return
				}
			}
		}
	}
}

// DynamicBodySizeLimit 根据路由动态设置请求体大小限制
func DynamicBodySizeLimit(limits map[string]int64, defaultLimit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前路由的限制
		path := c.FullPath()
		limit, exists := limits[path]
		if !exists {
			limit = defaultLimit
		}

		// 应用限制
		if c.Request.ContentLength > limit {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request body too large",
				"message": fmt.Sprintf("Request body exceeds maximum size of %d bytes for this endpoint", limit),
				"limit":   limit,
				"size":    c.Request.ContentLength,
				"path":    path,
			})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Header("X-Max-Body-Size", strconv.FormatInt(limit, 10))

		c.Next()
	}
}

// ContentTypeBasedLimit 根据内容类型设置不同的大小限制
func ContentTypeBasedLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentType := c.GetHeader("Content-Type")
		var limit int64

		// 根据内容类型设置限制
		switch {
		case contentType == "application/json":
			limit = SmallBodyLimit // JSON 请求通常较小
		case contentType == "multipart/form-data":
			limit = MediumBodyLimit // 文件上传
		case contentType == "application/octet-stream":
			limit = LargeBodyLimit // 二进制数据
		default:
			limit = DefaultBodyLimit
		}

		// 应用限制
		if c.Request.ContentLength > limit {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":        "Request body too large",
				"message":      fmt.Sprintf("Request body exceeds maximum size of %d bytes for content type %s", limit, contentType),
				"limit":        limit,
				"size":         c.Request.ContentLength,
				"content_type": contentType,
			})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Header("X-Max-Body-Size", strconv.FormatInt(limit, 10))
		c.Header("X-Content-Type-Limit", contentType)

		c.Next()
	}
}

// EmailBodyLimit 专门用于邮件相关端点的限制
func EmailBodyLimit() gin.HandlerFunc {
	// 邮件内容可能比较大，但需要限制以防止滥用
	const emailMaxSize = 25 * 1024 * 1024 // 25MB - 符合大多数邮件服务器的限制

	return func(c *gin.Context) {
		// 检查是否是邮件相关的端点
		path := c.FullPath()
		isEmailEndpoint := false

		// 判断是否是邮件相关端点
		switch path {
		case "/v1/mailboxes/:id/messages",
			"/api/emails/:emailId/:messageId",
			"/smtp/receive":
			isEmailEndpoint = true
		}

		var limit int64
		if isEmailEndpoint {
			limit = emailMaxSize
		} else {
			limit = SmallBodyLimit
		}

		// 应用限制
		if c.Request.ContentLength > limit {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":   "Request body too large",
				"message": fmt.Sprintf("Request body exceeds maximum size of %d bytes", limit),
				"limit":   limit,
				"size":    c.Request.ContentLength,
			})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Header("X-Max-Body-Size", strconv.FormatInt(limit, 10))

		c.Next()
	}
}

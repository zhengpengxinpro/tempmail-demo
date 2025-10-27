package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tempmail/backend/internal/security"
)

// EmailSecurityMiddleware 邮件安全中间件
type EmailSecurityMiddleware struct {
	contentFilter      *security.ContentFilter
	attachmentSecurity *security.AttachmentSecurity
	logger             *zap.Logger
}

// NewEmailSecurityMiddleware 创建邮件安全中间件
func NewEmailSecurityMiddleware(logger *zap.Logger) *EmailSecurityMiddleware {
	return &EmailSecurityMiddleware{
		contentFilter:      security.NewContentFilter(),
		attachmentSecurity: security.NewAttachmentSecurity(),
		logger:             logger,
	}
}

// ContentFilter 内容过滤中间件
func (esm *EmailSecurityMiddleware) ContentFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查请求体内容
		if c.Request.Method == "POST" {
			// 读取请求体
			body, err := c.GetRawData()
			if err != nil {
				esm.logger.Error("Failed to read request body", zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Failed to read request body",
				})
				c.Abort()
				return
			}

			// 过滤内容
			allowed, reason := esm.contentFilter.FilterEmail(string(body))
			if !allowed {
				esm.logger.Warn("Content filtered",
					zap.String("reason", reason),
					zap.String("ip", c.ClientIP()),
				)

				c.JSON(http.StatusBadRequest, gin.H{
					"error":  "Content not allowed",
					"reason": reason,
				})
				c.Abort()
				return
			}

			// 将请求体写回
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		c.Next()
	}
}

// AttachmentSecurity 附件安全中间件
func (esm *EmailSecurityMiddleware) AttachmentSecurity() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查附件上传
		if c.Request.Method == "POST" && strings.Contains(c.Request.URL.Path, "/attachments") {
			// 获取上传的文件
			file, header, err := c.Request.FormFile("file")
			if err != nil {
				c.Next()
				return
			}
			defer file.Close()

			// 检查附件安全性
			allowed, reason := esm.attachmentSecurity.CheckAttachment(
				header.Filename,
				file,
				header.Header.Get("Content-Type"),
			)

			if !allowed {
				esm.logger.Warn("Attachment blocked",
					zap.String("filename", header.Filename),
					zap.String("reason", reason),
					zap.String("ip", c.ClientIP()),
				)

				c.JSON(http.StatusBadRequest, gin.H{
					"error":  "Attachment not allowed",
					"reason": reason,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// SpamDetection 垃圾邮件检测中间件
func (esm *EmailSecurityMiddleware) SpamDetection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查邮件内容
		if c.Request.Method == "POST" && strings.Contains(c.Request.URL.Path, "/messages") {
			// 这里可以添加更复杂的垃圾邮件检测逻辑
			// 例如：检查发件人、主题、内容等

			// 检查发件人域名
			from := c.GetHeader("From")
			if esm.isSuspiciousSender(from) {
				esm.logger.Warn("Suspicious sender detected",
					zap.String("from", from),
					zap.String("ip", c.ClientIP()),
				)

				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Suspicious sender",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// isSuspiciousSender 检查是否为可疑发件人
func (esm *EmailSecurityMiddleware) isSuspiciousSender(from string) bool {
	// 检查常见的垃圾邮件域名
	suspiciousDomains := []string{
		"spam.com",
		"junk.com",
		"trash.com",
		"fake.com",
	}

	fromLower := strings.ToLower(from)
	for _, domain := range suspiciousDomains {
		if strings.Contains(fromLower, domain) {
			return true
		}
	}

	return false
}

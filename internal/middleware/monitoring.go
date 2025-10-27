package middleware

import (
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tempmail/backend/internal/monitoring"
)

// MonitoringMiddleware 监控中间件
type MonitoringMiddleware struct {
	metrics *monitoring.Metrics
	logger  *zap.Logger
}

// NewMonitoringMiddleware 创建监控中间件
func NewMonitoringMiddleware(metrics *monitoring.Metrics, logger *zap.Logger) *MonitoringMiddleware {
	return &MonitoringMiddleware{
		metrics: metrics,
		logger:  logger,
	}
}

// HTTPMetrics HTTP 指标中间件
func (mm *MonitoringMiddleware) HTTPMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestSize := c.Request.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}

		// 处理请求
		c.Next()

		// 计算指标
		duration := time.Since(start)
		statusCode := strconv.Itoa(c.Writer.Status())
		responseSize := int64(c.Writer.Size())

		// 记录指标
		mm.metrics.RecordHTTPRequest(
			c.Request.Method,
			c.FullPath(),
			statusCode,
			duration,
			requestSize,
			responseSize,
		)

		// 记录错误
		if c.Writer.Status() >= 400 {
			mm.metrics.RecordError("http_error", "http")
		}
	}
}

// PanicRecovery Panic 恢复中间件
func (mm *MonitoringMiddleware) PanicRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 指标
				mm.metrics.RecordPanic()

				// 记录错误日志
				mm.logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
				)

				// 返回错误响应
				c.JSON(500, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// BusinessMetrics 业务指标中间件
func (mm *MonitoringMiddleware) BusinessMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求
		c.Next()

		// 根据路径记录业务指标
		switch c.FullPath() {
		case "/v1/mailboxes":
			if c.Request.Method == "POST" {
				mm.metrics.RecordMailboxCreated()
			}
		case "/v1/mailboxes/:id":
			if c.Request.Method == "DELETE" {
				mm.metrics.RecordMailboxDeleted()
			}
		case "/v1/mailboxes/:id/messages":
			if c.Request.Method == "POST" {
				mm.metrics.RecordMessageReceived()
			}
		case "/v1/mailboxes/:id/messages/:messageId/read":
			if c.Request.Method == "POST" {
				mm.metrics.RecordMessageRead()
			}
		case "/v1/auth/register":
			if c.Request.Method == "POST" {
				mm.metrics.RecordUserRegistered()
			}
		}
	}
}

// RateLimitMetrics 限流指标中间件
func (mm *MonitoringMiddleware) RateLimitMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求
		c.Next()

		// 检查是否有限流响应头
		if c.Writer.Status() == 429 {
			// 记录限流阻止
			mm.metrics.RecordRateLimitBlock("http", c.ClientIP())
		}
	}
}

// SystemMetrics 系统指标中间件
func (mm *MonitoringMiddleware) SystemMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求
		c.Next()

		// 更新系统指标
		mm.updateSystemMetrics()
	}
}

// updateSystemMetrics 更新系统指标
func (mm *MonitoringMiddleware) updateSystemMetrics() {
	// 更新内存使用
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	mm.metrics.UpdateMemoryUsage(int64(m.Alloc))

	// 更新系统运行时间
	// 这里需要从应用启动时间计算
	// 简化处理，使用当前时间
	mm.metrics.UpdateSystemUptime(time.Since(time.Now().Add(-time.Hour)))
}

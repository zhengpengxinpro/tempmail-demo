package monitoring

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 监控指标
type Metrics struct {
	// HTTP 请求指标
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// 邮箱指标
	MailboxesCreated prometheus.Counter
	MailboxesDeleted prometheus.Counter
	MailboxesActive  prometheus.Gauge
	MailboxesExpired prometheus.Counter

	// 邮件指标
	MessagesReceived prometheus.Counter
	MessagesRead     prometheus.Counter
	MessagesDeleted  prometheus.Counter
	MessagesTotal    prometheus.Gauge

	// 用户指标
	UsersRegistered prometheus.Counter
	UsersActive     prometheus.Gauge
	UsersOnline     prometheus.Gauge

	// 系统指标
	SystemUptime        prometheus.Gauge
	DatabaseConnections prometheus.Gauge
	RedisConnections    prometheus.Gauge
	MemoryUsage         prometheus.Gauge
	CPUUsage            prometheus.Gauge

	// 错误指标
	ErrorsTotal *prometheus.CounterVec
	PanicsTotal prometheus.Counter

	// 限流指标
	RateLimitHits   *prometheus.CounterVec
	RateLimitBlocks *prometheus.CounterVec

	// 业务指标
	DomainUsage         *prometheus.GaugeVec
	AttachmentSize      *prometheus.HistogramVec
	EmailProcessingTime *prometheus.HistogramVec
}

// NewMetrics 创建监控指标
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP 请求指标
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tempmail_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tempmail_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		HTTPRequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tempmail_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tempmail_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "endpoint"},
		),

		// 邮箱指标
		MailboxesCreated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_mailboxes_created_total",
				Help: "Total number of mailboxes created",
			},
		),

		MailboxesDeleted: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_mailboxes_deleted_total",
				Help: "Total number of mailboxes deleted",
			},
		),

		MailboxesActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_mailboxes_active",
				Help: "Number of active mailboxes",
			},
		),

		MailboxesExpired: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_mailboxes_expired_total",
				Help: "Total number of expired mailboxes",
			},
		),

		// 邮件指标
		MessagesReceived: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_messages_received_total",
				Help: "Total number of messages received",
			},
		),

		MessagesRead: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_messages_read_total",
				Help: "Total number of messages read",
			},
		),

		MessagesDeleted: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_messages_deleted_total",
				Help: "Total number of messages deleted",
			},
		),

		MessagesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_messages_total",
				Help: "Total number of messages",
			},
		),

		// 用户指标
		UsersRegistered: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_users_registered_total",
				Help: "Total number of users registered",
			},
		),

		UsersActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_users_active",
				Help: "Number of active users",
			},
		),

		UsersOnline: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_users_online",
				Help: "Number of online users",
			},
		),

		// 系统指标
		SystemUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_system_uptime_seconds",
				Help: "System uptime in seconds",
			},
		),

		DatabaseConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_database_connections",
				Help: "Number of database connections",
			},
		),

		RedisConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_redis_connections",
				Help: "Number of Redis connections",
			},
		),

		MemoryUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_memory_usage_bytes",
				Help: "Memory usage in bytes",
			},
		),

		CPUUsage: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "tempmail_cpu_usage_percent",
				Help: "CPU usage percentage",
			},
		),

		// 错误指标
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tempmail_errors_total",
				Help: "Total number of errors",
			},
			[]string{"type", "component"},
		),

		PanicsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "tempmail_panics_total",
				Help: "Total number of panics",
			},
		),

		// 限流指标
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tempmail_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"type", "key"},
		),

		RateLimitBlocks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tempmail_rate_limit_blocks_total",
				Help: "Total number of rate limit blocks",
			},
			[]string{"type", "key"},
		),

		// 业务指标
		DomainUsage: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tempmail_domain_usage",
				Help: "Usage by domain",
			},
			[]string{"domain"},
		),

		AttachmentSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tempmail_attachment_size_bytes",
				Help:    "Attachment size in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 20),
			},
			[]string{"type"},
		),

		EmailProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tempmail_email_processing_duration_seconds",
				Help:    "Email processing duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"type"},
		),
	}
}

// RecordHTTPRequest 记录 HTTP 请求指标
func (m *Metrics) RecordHTTPRequest(method, endpoint, statusCode string, duration time.Duration, requestSize, responseSize int64) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordMailboxCreated 记录邮箱创建
func (m *Metrics) RecordMailboxCreated() {
	m.MailboxesCreated.Inc()
}

// RecordMailboxDeleted 记录邮箱删除
func (m *Metrics) RecordMailboxDeleted() {
	m.MailboxesDeleted.Inc()
}

// RecordMailboxExpired 记录邮箱过期
func (m *Metrics) RecordMailboxExpired() {
	m.MailboxesExpired.Inc()
}

// RecordMessageReceived 记录邮件接收
func (m *Metrics) RecordMessageReceived() {
	m.MessagesReceived.Inc()
}

// RecordMessageRead 记录邮件阅读
func (m *Metrics) RecordMessageRead() {
	m.MessagesRead.Inc()
}

// RecordMessageDeleted 记录邮件删除
func (m *Metrics) RecordMessageDeleted() {
	m.MessagesDeleted.Inc()
}

// RecordUserRegistered 记录用户注册
func (m *Metrics) RecordUserRegistered() {
	m.UsersRegistered.Inc()
}

// RecordError 记录错误
func (m *Metrics) RecordError(errorType, component string) {
	m.ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

// RecordPanic 记录 panic
func (m *Metrics) RecordPanic() {
	m.PanicsTotal.Inc()
}

// RecordRateLimitHit 记录限流命中
func (m *Metrics) RecordRateLimitHit(limitType, key string) {
	m.RateLimitHits.WithLabelValues(limitType, key).Inc()
}

// RecordRateLimitBlock 记录限流阻止
func (m *Metrics) RecordRateLimitBlock(limitType, key string) {
	m.RateLimitBlocks.WithLabelValues(limitType, key).Inc()
}

// UpdateMailboxesActive 更新活跃邮箱数
func (m *Metrics) UpdateMailboxesActive(count int) {
	m.MailboxesActive.Set(float64(count))
}

// UpdateMessagesTotal 更新总邮件数
func (m *Metrics) UpdateMessagesTotal(count int) {
	m.MessagesTotal.Set(float64(count))
}

// UpdateUsersActive 更新活跃用户数
func (m *Metrics) UpdateUsersActive(count int) {
	m.UsersActive.Set(float64(count))
}

// UpdateUsersOnline 更新在线用户数
func (m *Metrics) UpdateUsersOnline(count int) {
	m.UsersOnline.Set(float64(count))
}

// UpdateSystemUptime 更新系统运行时间
func (m *Metrics) UpdateSystemUptime(uptime time.Duration) {
	m.SystemUptime.Set(uptime.Seconds())
}

// UpdateDatabaseConnections 更新数据库连接数
func (m *Metrics) UpdateDatabaseConnections(count int) {
	m.DatabaseConnections.Set(float64(count))
}

// UpdateRedisConnections 更新 Redis 连接数
func (m *Metrics) UpdateRedisConnections(count int) {
	m.RedisConnections.Set(float64(count))
}

// UpdateMemoryUsage 更新内存使用量
func (m *Metrics) UpdateMemoryUsage(bytes int64) {
	m.MemoryUsage.Set(float64(bytes))
}

// UpdateCPUUsage 更新 CPU 使用率
func (m *Metrics) UpdateCPUUsage(percent float64) {
	m.CPUUsage.Set(percent)
}

// UpdateDomainUsage 更新域名使用量
func (m *Metrics) UpdateDomainUsage(domain string, count int) {
	m.DomainUsage.WithLabelValues(domain).Set(float64(count))
}

// RecordAttachmentSize 记录附件大小
func (m *Metrics) RecordAttachmentSize(attachmentType string, size int64) {
	m.AttachmentSize.WithLabelValues(attachmentType).Observe(float64(size))
}

// RecordEmailProcessingTime 记录邮件处理时间
func (m *Metrics) RecordEmailProcessingTime(processingType string, duration time.Duration) {
	m.EmailProcessingTime.WithLabelValues(processingType).Observe(duration.Seconds())
}

// HTTPHandler 返回 Prometheus HTTP 处理器
func (m *Metrics) HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// RegisterCustomMetrics 注册自定义指标
func (m *Metrics) RegisterCustomMetrics() {
	// 注册所有指标到默认注册表
	prometheus.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.MailboxesCreated,
		m.MailboxesDeleted,
		m.MailboxesActive,
		m.MailboxesExpired,
		m.MessagesReceived,
		m.MessagesRead,
		m.MessagesDeleted,
		m.MessagesTotal,
		m.UsersRegistered,
		m.UsersActive,
		m.UsersOnline,
		m.SystemUptime,
		m.DatabaseConnections,
		m.RedisConnections,
		m.MemoryUsage,
		m.CPUUsage,
		m.ErrorsTotal,
		m.PanicsTotal,
		m.RateLimitHits,
		m.RateLimitBlocks,
		m.DomainUsage,
		m.AttachmentSize,
		m.EmailProcessingTime,
	)
}

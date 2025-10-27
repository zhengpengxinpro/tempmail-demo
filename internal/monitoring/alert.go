package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"tempmail/backend/internal/storage"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert 告警
type Alert struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Level      AlertLevel             `json:"level"`
	Component  string                 `json:"component"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID            string
	Name          string
	Condition     func() bool
	Level         AlertLevel
	Component     string
	Message       string
	Cooldown      time.Duration
	LastTriggered time.Time
}

// AlertManager 告警管理器
type AlertManager struct {
	alerts    map[string]*Alert
	rules     []AlertRule
	receivers []AlertReceiver
	logger    *zap.Logger
	mu        sync.RWMutex
}

// AlertReceiver 告警接收器接口
type AlertReceiver interface {
	SendAlert(alert *Alert) error
}

// NewAlertManager 创建告警管理器
func NewAlertManager(logger *zap.Logger) *AlertManager {
	return &AlertManager{
		alerts:    make(map[string]*Alert),
		rules:     make([]AlertRule, 0),
		receivers: make([]AlertReceiver, 0),
		logger:    logger,
	}
}

// AddReceiver 添加告警接收器
func (am *AlertManager) AddReceiver(receiver AlertReceiver) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.receivers = append(am.receivers, receiver)
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = append(am.rules, rule)
}

// TriggerAlert 触发告警
func (am *AlertManager) TriggerAlert(alert *Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// 检查是否已存在相同的告警
	if existing, exists := am.alerts[alert.ID]; exists && !existing.Resolved {
		am.logger.Debug("Alert already exists and not resolved",
			zap.String("alert_id", alert.ID),
		)
		return
	}

	// 添加或更新告警
	am.alerts[alert.ID] = alert

	// 发送告警
	for _, receiver := range am.receivers {
		if err := receiver.SendAlert(alert); err != nil {
			am.logger.Error("Failed to send alert",
				zap.String("alert_id", alert.ID),
				zap.Error(err),
			)
		}
	}

	am.logger.Info("Alert triggered",
		zap.String("alert_id", alert.ID),
		zap.String("level", string(alert.Level)),
		zap.String("component", alert.Component),
	)
}

// ResolveAlert 解决告警
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if alert, exists := am.alerts[alertID]; exists && !alert.Resolved {
		now := time.Now()
		alert.Resolved = true
		alert.ResolvedAt = &now

		am.logger.Info("Alert resolved",
			zap.String("alert_id", alertID),
		)
	}
}

// GetAlerts 获取告警列表
func (am *AlertManager) GetAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, *alert)
	}

	return alerts
}

// GetActiveAlerts 获取活跃告警
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]Alert, 0)
	for _, alert := range am.alerts {
		if !alert.Resolved {
			alerts = append(alerts, *alert)
		}
	}

	return alerts
}

// CheckRules 检查告警规则
func (am *AlertManager) CheckRules() {
	am.mu.RLock()
	rules := make([]AlertRule, len(am.rules))
	copy(rules, am.rules)
	am.mu.RUnlock()

	for _, rule := range rules {
		// 检查冷却时间
		if time.Since(rule.LastTriggered) < rule.Cooldown {
			continue
		}

		// 检查条件
		if rule.Condition() {
			alert := &Alert{
				ID:        fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix()),
				Title:     rule.Name,
				Message:   rule.Message,
				Level:     rule.Level,
				Component: rule.Component,
				Timestamp: time.Now(),
				Resolved:  false,
			}

			am.TriggerAlert(alert)

			// 更新最后触发时间
			am.mu.Lock()
			for i, r := range am.rules {
				if r.ID == rule.ID {
					am.rules[i].LastTriggered = time.Now()
					break
				}
			}
			am.mu.Unlock()
		}
	}
}

// StartMonitoring 启动监控
func (am *AlertManager) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.CheckRules()
		}
	}
}

// ========== 内置告警规则 ==========

// HighMemoryUsageRule 高内存使用告警规则
func HighMemoryUsageRule(thresholdMB float64) AlertRule {
	return AlertRule{
		ID:   "high_memory_usage",
		Name: "High Memory Usage",
		Condition: func() bool {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memoryUsageMB := float64(m.Alloc) / 1024 / 1024
			return memoryUsageMB > thresholdMB
		},
		Level:     AlertLevelWarning,
		Component: "memory",
		Message:   fmt.Sprintf("Memory usage exceeds %f MB", thresholdMB),
		Cooldown:  5 * time.Minute,
	}
}

// HighCPUUsageRule 高 CPU 使用告警规则
func HighCPUUsageRule(thresholdPercent float64) AlertRule {
	return AlertRule{
		ID:   "high_cpu_usage",
		Name: "High CPU Usage",
		Condition: func() bool {
			// 这里需要实现 CPU 使用率检查
			// 简化处理，总是返回 false
			return false
		},
		Level:     AlertLevelWarning,
		Component: "cpu",
		Message:   fmt.Sprintf("CPU usage exceeds %.1f%%", thresholdPercent),
		Cooldown:  2 * time.Minute,
	}
}

// DatabaseConnectionRule 数据库连接告警规则
func DatabaseConnectionRule(store storage.Store) AlertRule {
	return AlertRule{
		ID:   "database_connection",
		Name: "Database Connection",
		Condition: func() bool {
			return store.Health() != nil
		},
		Level:     AlertLevelCritical,
		Component: "database",
		Message:   "Database connection failed",
		Cooldown:  1 * time.Minute,
	}
}

// HighErrorRateRule 高错误率告警规则
func HighErrorRateRule(errorCounter *prometheus.CounterVec, threshold float64) AlertRule {
	return AlertRule{
		ID:   "high_error_rate",
		Name: "High Error Rate",
		Condition: func() bool {
			// 这里需要实现错误率检查
			// 简化处理，总是返回 false
			return false
		},
		Level:     AlertLevelWarning,
		Component: "system",
		Message:   fmt.Sprintf("Error rate exceeds %.1f%%", threshold),
		Cooldown:  5 * time.Minute,
	}
}

// ========== 告警接收器实现 ==========

// LogAlertReceiver 日志告警接收器
type LogAlertReceiver struct {
	logger *zap.Logger
}

// NewLogAlertReceiver 创建日志告警接收器
func NewLogAlertReceiver(logger *zap.Logger) *LogAlertReceiver {
	return &LogAlertReceiver{logger: logger}
}

// SendAlert 发送告警到日志
func (lar *LogAlertReceiver) SendAlert(alert *Alert) error {
	switch alert.Level {
	case AlertLevelCritical:
		lar.logger.Error("CRITICAL ALERT",
			zap.String("alert_id", alert.ID),
			zap.String("title", alert.Title),
			zap.String("message", alert.Message),
			zap.String("component", alert.Component),
			zap.Time("timestamp", alert.Timestamp),
		)
	case AlertLevelWarning:
		lar.logger.Warn("WARNING ALERT",
			zap.String("alert_id", alert.ID),
			zap.String("title", alert.Title),
			zap.String("message", alert.Message),
			zap.String("component", alert.Component),
			zap.Time("timestamp", alert.Timestamp),
		)
	case AlertLevelInfo:
		lar.logger.Info("INFO ALERT",
			zap.String("alert_id", alert.ID),
			zap.String("title", alert.Title),
			zap.String("message", alert.Message),
			zap.String("component", alert.Component),
			zap.Time("timestamp", alert.Timestamp),
		)
	}

	return nil
}

// WebhookAlertReceiver Webhook 告警接收器
type WebhookAlertReceiver struct {
	url    string
	client *http.Client
	logger *zap.Logger
}

// NewWebhookAlertReceiver 创建 Webhook 告警接收器
func NewWebhookAlertReceiver(url string, logger *zap.Logger) *WebhookAlertReceiver {
	return &WebhookAlertReceiver{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

// SendAlert 发送告警到 Webhook
func (war *WebhookAlertReceiver) SendAlert(alert *Alert) error {
	// 这里需要实现 HTTP 请求发送告警
	// 简化处理，只记录日志
	war.logger.Info("Sending alert to webhook",
		zap.String("url", war.url),
		zap.String("alert_id", alert.ID),
		zap.String("level", string(alert.Level)),
	)

	return nil
}

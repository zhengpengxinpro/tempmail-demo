package domain

import "time"

// WebhookEventType Webhook 事件类型
type WebhookEventType string

const (
	WebhookEventMailReceived   WebhookEventType = "mail.received"   // 新邮件到达
	WebhookEventMailRead       WebhookEventType = "mail.read"       // 邮件已读
	WebhookEventMailboxCreated WebhookEventType = "mailbox.created" // 邮箱创建
	WebhookEventMailboxDeleted WebhookEventType = "mailbox.deleted" // 邮箱删除
	WebhookEventTagCreated     WebhookEventType = "tag.created"     // 标签创建
	WebhookEventTagUpdated     WebhookEventType = "tag.updated"     // 标签更新
	WebhookEventTagDeleted     WebhookEventType = "tag.deleted"     // 标签删除
	WebhookEventMessageTagged  WebhookEventType = "message.tagged"  // 邮件添加标签
)

// Webhook Webhook 配置
type Webhook struct {
	ID          string           `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID      string           `json:"userId" gorm:"type:varchar(36);index;not null"`
	URL         string           `json:"url" gorm:"type:varchar(500);not null"`
	Events      []string         `json:"events" gorm:"serializer:json;type:json"`
	Secret      string           `json:"secret" gorm:"type:varchar(255)"`
	IsActive    bool             `json:"isActive" gorm:"default:true"`
	RetryCount  int              `json:"retryCount" gorm:"default:0"`
	LastError   string           `json:"lastError" gorm:"type:text"`
	LastSuccess *time.Time       `json:"lastSuccess"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// WebhookEvent Webhook 事件数据
type WebhookEvent struct {
	ID        string           `json:"id"`
	Event     WebhookEventType `json:"event"`     // 事件类型
	Timestamp time.Time        `json:"timestamp"` // 事件时间
	Data      interface{}      `json:"data"`      // 事件数据
}

// WebhookDelivery Webhook 投递记录
type WebhookDelivery struct {
	ID          string           `json:"id"`
	WebhookID   string           `json:"webhookId"`
	Event       WebhookEventType `json:"event"`
	Payload     string           `json:"payload"`      // JSON payload
	StatusCode  int              `json:"statusCode"`   // HTTP 状态码
	Response    string           `json:"response"`     // 响应内容
	Duration    int64            `json:"duration"`     // 请求耗时（毫秒）
	Success     bool             `json:"success"`      // 是否成功
	Error       string           `json:"error"`        // 错误信息
	Attempts    int              `json:"attempts"`     // 尝试次数
	NextRetry   *time.Time       `json:"nextRetry"`    // 下次重试时间
	CreatedAt   time.Time        `json:"createdAt"`
}

// WebhookRepository Webhook 仓储接口
type WebhookRepository interface {
	// CreateWebhook 创建 Webhook
	CreateWebhook(webhook *Webhook) error
	
	// GetWebhook 获取 Webhook
	GetWebhook(id string) (*Webhook, error)
	
	// ListWebhooks 列出用户的 Webhooks
	ListWebhooks(userID string) ([]Webhook, error)
	
	// UpdateWebhook 更新 Webhook
	UpdateWebhook(webhook *Webhook) error
	
	// DeleteWebhook 删除 Webhook
	DeleteWebhook(id string) error
	
	// RecordDelivery 记录投递结果
	RecordDelivery(delivery *WebhookDelivery) error
	
	// GetDeliveries 获取投递记录
	GetDeliveries(webhookID string, limit int) ([]WebhookDelivery, error)
	
	// GetPendingDeliveries 获取待重试的投递
	GetPendingDeliveries(limit int) ([]WebhookDelivery, error)
}

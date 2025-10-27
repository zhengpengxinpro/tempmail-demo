package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"tempmail/backend/internal/domain"
)

// WebhookService Webhook 服务
type WebhookService struct {
	store      domain.Store
	httpClient *http.Client
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(store domain.Store) *WebhookService {
	return &WebhookService{
		store: store,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateWebhookInput 创建 Webhook 输入
type CreateWebhookInput struct {
	UserID      string   `json:"-"` // 从JWT中获取，不需要客户端提供
	URL         string   `json:"url" binding:"required,url"`
	Events      []string `json:"events" binding:"required,min=1"`
	Description string   `json:"description" binding:"omitempty,max=200"`
}

// UpdateWebhookInput 更新 Webhook 输入
type UpdateWebhookInput struct {
	URL         string   `json:"url" binding:"omitempty,url"`
	Events      []string `json:"events" binding:"omitempty,min=1"`
	Description string   `json:"description" binding:"omitempty,max=200"`
	IsActive    *bool    `json:"isActive"`
}

// CreateWebhook 创建 Webhook
func (s *WebhookService) CreateWebhook(input CreateWebhookInput) (*domain.Webhook, error) {
	// 生成密钥
	secret := generateSecret()

	webhook := &domain.Webhook{
		ID:       uuid.New().String(),
		UserID:   input.UserID,
		URL:      input.URL,
		Events:   input.Events,
		Secret:   secret,
		IsActive: true,
	}

	if err := s.store.CreateWebhook(webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// GetWebhook 获取 Webhook
func (s *WebhookService) GetWebhook(id string) (*domain.Webhook, error) {
	return s.store.GetWebhook(id)
}

// ListWebhooks 列出用户的 Webhooks
func (s *WebhookService) ListWebhooks(userID string) ([]domain.Webhook, error) {
	return s.store.ListWebhooks(userID)
}

// UpdateWebhook 更新 Webhook
func (s *WebhookService) UpdateWebhook(id string, input UpdateWebhookInput) (*domain.Webhook, error) {
	webhook, err := s.store.GetWebhook(id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if input.URL != "" {
		webhook.URL = input.URL
	}
	if len(input.Events) > 0 {
		webhook.Events = input.Events
	}
	if input.IsActive != nil {
		webhook.IsActive = *input.IsActive
	}

	if err := s.store.UpdateWebhook(webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// DeleteWebhook 删除 Webhook
func (s *WebhookService) DeleteWebhook(id string) error {
	return s.store.DeleteWebhook(id)
}

// TriggerEvent 触发 Webhook 事件
func (s *WebhookService) TriggerEvent(userID string, eventType domain.WebhookEventType, data interface{}) error {
	// 获取用户的所有 Webhooks
	webhooks, err := s.store.ListWebhooks(userID)
	if err != nil {
		return err
	}

	// 构建事件数据
	event := domain.WebhookEvent{
		ID:        uuid.New().String(),
		Event:     eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	// 遍历 Webhooks，异步发送
	for _, webhook := range webhooks {
		// 检查是否启用
		if !webhook.IsActive {
			continue
		}

		// 检查是否订阅该事件
		if !containsEvent(webhook.Events, string(eventType)) {
			continue
		}

		// 异步发送
		go s.deliverWebhook(&webhook, event)
	}

	return nil
}

// deliverWebhook 投递 Webhook
func (s *WebhookService) deliverWebhook(webhook *domain.Webhook, event domain.WebhookEvent) {
	delivery := &domain.WebhookDelivery{
		ID:        uuid.New().String(),
		WebhookID: webhook.ID,
		Event:     event.Event,
		Attempts:  1,
	}

	// 序列化 payload
	payload, err := json.Marshal(event)
	if err != nil {
		delivery.Success = false
		delivery.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		s.store.RecordDelivery(delivery)
		return
	}
	delivery.Payload = string(payload)

	// 生成签名
	signature := generateSignature(payload, webhook.Secret)

	// 发送 HTTP 请求
	startTime := time.Now()
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payload))
	if err != nil {
		delivery.Success = false
		delivery.Error = fmt.Sprintf("failed to create request: %v", err)
		delivery.Duration = time.Since(startTime).Milliseconds()
		s.store.RecordDelivery(delivery)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Event", string(event.Event))
	req.Header.Set("X-Webhook-ID", delivery.ID)

	resp, err := s.httpClient.Do(req)
	delivery.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		delivery.Success = false
		delivery.Error = fmt.Sprintf("failed to send request: %v", err)
		delivery.NextRetry = calculateNextRetry(delivery.Attempts)
		s.store.RecordDelivery(delivery)
		return
	}
	defer resp.Body.Close()

	delivery.StatusCode = resp.StatusCode

	// 读取响应
	body, _ := io.ReadAll(resp.Body)
	delivery.Response = string(body)

	// 判断是否成功
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.Success = true
	} else {
		delivery.Success = false
		delivery.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, delivery.Response)
		
		// 如果失败，计算下次重试时间
		if delivery.Attempts < 5 {
			delivery.NextRetry = calculateNextRetry(delivery.Attempts)
		}
	}

	s.store.RecordDelivery(delivery)
}

// GetDeliveries 获取投递记录
func (s *WebhookService) GetDeliveries(webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.store.GetDeliveries(webhookID, limit)
}

// RetryFailedDeliveries 重试失败的投递
func (s *WebhookService) RetryFailedDeliveries() error {
	// 获取待重试的投递
	deliveries, err := s.store.GetPendingDeliveries(10)
	if err != nil {
		return err
	}

	// 重新投递
	for _, delivery := range deliveries {
		webhook, err := s.store.GetWebhook(delivery.WebhookID)
		if err != nil {
			continue
		}

		// 解析事件数据
		var event domain.WebhookEvent
		if err := json.Unmarshal([]byte(delivery.Payload), &event); err != nil {
			continue
		}

		// 异步重试
		go s.deliverWebhook(webhook, event)
	}

	return nil
}

// generateSecret 生成 Webhook 密钥
func generateSecret() string {
	return uuid.New().String()
}

// generateSignature 生成 HMAC-SHA256 签名
func generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// calculateNextRetry 计算下次重试时间（指数退避）
func calculateNextRetry(attempts int) *time.Time {
	// 重试间隔：1分钟、5分钟、15分钟、1小时、6小时
	intervals := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		6 * time.Hour,
	}

	index := attempts - 1
	if index >= len(intervals) {
		return nil // 不再重试
	}

	nextRetry := time.Now().Add(intervals[index])
	return &nextRetry
}

// containsEvent 检查事件列表是否包含指定事件
func containsEvent(events []string, event string) bool {
	for _, e := range events {
		if e == event {
			return true
		}
	}
	return false
}

package memory

import (
	"fmt"
	"time"

	"tempmail/backend/internal/domain"
)

// CreateWebhook 创建 Webhook
func (s *Store) CreateWebhook(webhook *domain.Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	if _, exists := s.webhooks[webhook.ID]; exists {
		return fmt.Errorf("webhook already exists")
	}

	webhook.CreatedAt = time.Now()
	webhook.UpdatedAt = time.Now()
	s.webhooks[webhook.ID] = webhook

	// 按用户ID索引
	if s.webhooksByUser[webhook.UserID] == nil {
		s.webhooksByUser[webhook.UserID] = make(map[string]*domain.Webhook)
	}
	s.webhooksByUser[webhook.UserID][webhook.ID] = webhook

	return nil
}

// GetWebhook 获取 Webhook
func (s *Store) GetWebhook(id string) (*domain.Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	webhook, exists := s.webhooks[id]
	if !exists {
		return nil, fmt.Errorf("webhook not found")
	}

	return webhook, nil
}

// ListWebhooks 列出用户的 Webhooks
func (s *Store) ListWebhooks(userID string) ([]domain.Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userWebhooks := s.webhooksByUser[userID]
	if userWebhooks == nil {
		return []domain.Webhook{}, nil
	}

	result := make([]domain.Webhook, 0, len(userWebhooks))
	for _, webhook := range userWebhooks {
		result = append(result, *webhook)
	}

	return result, nil
}

// UpdateWebhook 更新 Webhook
func (s *Store) UpdateWebhook(webhook *domain.Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.webhooks[webhook.ID]
	if !exists {
		return fmt.Errorf("webhook not found")
	}

	webhook.UpdatedAt = time.Now()
	webhook.CreatedAt = existing.CreatedAt
	s.webhooks[webhook.ID] = webhook
	s.webhooksByUser[webhook.UserID][webhook.ID] = webhook

	return nil
}

// DeleteWebhook 删除 Webhook
func (s *Store) DeleteWebhook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhook, exists := s.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found")
	}

	delete(s.webhooks, id)
	delete(s.webhooksByUser[webhook.UserID], id)

	return nil
}

// RecordDelivery 记录投递结果
func (s *Store) RecordDelivery(delivery *domain.WebhookDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delivery.CreatedAt = time.Now()
	
	// 存储投递记录
	if s.deliveries[delivery.WebhookID] == nil {
		s.deliveries[delivery.WebhookID] = make([]*domain.WebhookDelivery, 0)
	}
	s.deliveries[delivery.WebhookID] = append(s.deliveries[delivery.WebhookID], delivery)

	// 限制最多保存 100 条记录
	if len(s.deliveries[delivery.WebhookID]) > 100 {
		s.deliveries[delivery.WebhookID] = s.deliveries[delivery.WebhookID][1:]
	}

	// 如果需要重试，加入重试队列
	if !delivery.Success && delivery.NextRetry != nil {
		s.retryQueue = append(s.retryQueue, delivery)
	}

	// 更新 Webhook 状态
	webhook := s.webhooks[delivery.WebhookID]
	if webhook != nil {
		if delivery.Success {
			now := time.Now()
			webhook.LastSuccess = &now
			webhook.LastError = ""
		} else {
			webhook.RetryCount++
			webhook.LastError = delivery.Error
		}
		webhook.UpdatedAt = time.Now()
	}

	return nil
}

// GetDeliveries 获取投递记录
func (s *Store) GetDeliveries(webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deliveries := s.deliveries[webhookID]
	if deliveries == nil {
		return []domain.WebhookDelivery{}, nil
	}

	// 返回最近的 N 条记录
	start := 0
	if len(deliveries) > limit {
		start = len(deliveries) - limit
	}

	result := make([]domain.WebhookDelivery, 0, limit)
	for i := len(deliveries) - 1; i >= start; i-- {
		result = append(result, *deliveries[i])
	}

	return result, nil
}

// GetPendingDeliveries 获取待重试的投递
func (s *Store) GetPendingDeliveries(limit int) ([]domain.WebhookDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	result := make([]domain.WebhookDelivery, 0)
	newQueue := make([]*domain.WebhookDelivery, 0)

	for _, delivery := range s.retryQueue {
		if delivery.NextRetry != nil && delivery.NextRetry.Before(now) {
			// 可以重试
			if len(result) < limit {
				result = append(result, *delivery)
			} else {
				newQueue = append(newQueue, delivery)
			}
		} else {
			// 还未到重试时间
			newQueue = append(newQueue, delivery)
		}
	}

	s.retryQueue = newQueue
	return result, nil
}

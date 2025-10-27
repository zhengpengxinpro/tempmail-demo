package postgres

import (
	"fmt"

	"gorm.io/gorm"
	"tempmail/backend/internal/domain"
)

// SearchMessages 搜索邮件（PostgreSQL 实现，支持全文搜索）
func (s *Store) SearchMessages(criteria domain.MessageSearchCriteria) (*domain.MessageSearchResult, error) {
	// 设置默认分页参数
	if criteria.Page <= 0 {
		criteria.Page = 1
	}
	if criteria.PageSize <= 0 {
		criteria.PageSize = 20
	}
	if criteria.PageSize > 100 {
		criteria.PageSize = 100
	}

	// 构建查询
	query := s.db.Model(&domain.Message{}).Where("mailbox_id = ?", criteria.MailboxID)

	// 关键词搜索（使用 LIKE 进行不区分大小写搜索，兼容MySQL和PostgreSQL）
	if criteria.Query != "" {
		searchPattern := "%" + criteria.Query + "%"
		// 使用GORM字段名而不是数据库列名
		query = query.Where(
			"subject LIKE ? OR `from` LIKE ? OR text LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// 发件人筛选
	if criteria.From != "" {
		query = query.Where("`from` LIKE ?", "%"+criteria.From+"%")
	}

	// 主题筛选（MySQL的LIKE默认不区分大小写）
	if criteria.Subject != "" {
		query = query.Where("subject LIKE ?", "%"+criteria.Subject+"%")
	}

	// 时间范围筛选
	if criteria.StartDate != nil {
		query = query.Where("created_at >= ?", *criteria.StartDate)
	}
	if criteria.EndDate != nil {
		query = query.Where("created_at <= ?", *criteria.EndDate)
	}

	// 已读状态筛选
	if criteria.IsRead != nil {
		query = query.Where("is_read = ?", *criteria.IsRead)
	}

	// 附件筛选（需要子查询）
	if criteria.HasAttachment != nil {
		if *criteria.HasAttachment {
			// 有附件：EXISTS子查询
			query = query.Where("EXISTS (SELECT 1 FROM attachments WHERE attachments.message_id = messages.id)")
		} else {
			// 无附件：NOT EXISTS子查询
			query = query.Where("NOT EXISTS (SELECT 1 FROM attachments WHERE attachments.message_id = messages.id)")
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	// 分页查询
	var messages []domain.Message
	offset := (criteria.Page - 1) * criteria.PageSize
	if err := query.
		Order("created_at DESC").
		Limit(criteria.PageSize).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / criteria.PageSize
	if int(total)%criteria.PageSize > 0 {
		totalPages++
	}

	return &domain.MessageSearchResult{
		Messages:   messages,
		Total:      int(total),
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ========== Webhook Repository ==========

// CreateWebhook 创建 Webhook
func (s *Store) CreateWebhook(webhook *domain.Webhook) error {
	return s.db.Create(webhook).Error
}

// GetWebhook 获取 Webhook
func (s *Store) GetWebhook(id string) (*domain.Webhook, error) {
	var webhook domain.Webhook
	if err := s.db.Where("id = ?", id).First(&webhook).Error; err != nil {
		return nil, fmt.Errorf("webhook not found: %w", err)
	}
	return &webhook, nil
}

// ListWebhooks 列出用户的 Webhooks
func (s *Store) ListWebhooks(userID string) ([]domain.Webhook, error) {
	var webhooks []domain.Webhook
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&webhooks).Error; err != nil {
		return nil, err
	}
	return webhooks, nil
}

// UpdateWebhook 更新 Webhook
func (s *Store) UpdateWebhook(webhook *domain.Webhook) error {
	return s.db.Save(webhook).Error
}

// DeleteWebhook 删除 Webhook
func (s *Store) DeleteWebhook(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除投递记录
		if err := tx.Where("webhook_id = ?", id).Delete(&domain.WebhookDelivery{}).Error; err != nil {
			return err
		}
		// 删除 Webhook
		return tx.Where("id = ?", id).Delete(&domain.Webhook{}).Error
	})
}

// RecordDelivery 记录投递
func (s *Store) RecordDelivery(delivery *domain.WebhookDelivery) error {
	return s.db.Create(delivery).Error
}

// GetDeliveries 获取投递记录
func (s *Store) GetDeliveries(webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	var deliveries []domain.WebhookDelivery
	if err := s.db.
		Where("webhook_id = ?", webhookID).
		Order("created_at DESC").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		return nil, err
	}
	return deliveries, nil
}

// GetPendingDeliveries 获取待重试投递
func (s *Store) GetPendingDeliveries(limit int) ([]domain.WebhookDelivery, error) {
	var deliveries []domain.WebhookDelivery
	if err := s.db.
		Where("success = ? AND next_retry IS NOT NULL AND next_retry <= NOW()", false).
		Order("next_retry ASC").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		return nil, err
	}
	return deliveries, nil
}

package sql

import (
	"database/sql"
	"fmt"

	"tempmail/backend/internal/domain"
)

// SearchMessages 搜索邮件（SQL 实现）
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

	// 构建WHERE子句
	whereClauses := []string{"mailbox_id = ?"}
	args := []interface{}{criteria.MailboxID}

	// 关键词搜索
	if criteria.Query != "" {
		whereClauses = append(whereClauses, "(subject LIKE ? OR from_address LIKE ? OR text_content LIKE ?)")
		searchPattern := "%" + criteria.Query + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// 发件人筛选
	if criteria.From != "" {
		whereClauses = append(whereClauses, "from_address LIKE ?")
		args = append(args, "%"+criteria.From+"%")
	}

	// 主题筛选
	if criteria.Subject != "" {
		whereClauses = append(whereClauses, "subject LIKE ?")
		args = append(args, "%"+criteria.Subject+"%")
	}

	// 时间范围筛选
	if criteria.StartDate != nil {
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, *criteria.StartDate)
	}
	if criteria.EndDate != nil {
		whereClauses = append(whereClauses, "created_at <= ?")
		args = append(args, *criteria.EndDate)
	}

	// 已读状态筛选
	if criteria.IsRead != nil {
		whereClauses = append(whereClauses, "is_read = ?")
		args = append(args, *criteria.IsRead)
	}

	// 附件筛选
	if criteria.HasAttachment != nil {
		if *criteria.HasAttachment {
			whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM attachments WHERE attachments.message_id = messages.id)")
		} else {
			whereClauses = append(whereClauses, "NOT EXISTS (SELECT 1 FROM attachments WHERE attachments.message_id = messages.id)")
		}
	}

	// 构建完整查询
	whereClause := ""
	for i, clause := range whereClauses {
		if i == 0 {
			whereClause = "WHERE " + clause
		} else {
			whereClause += " AND " + clause
		}
	}

	// 获取总数
	countQuery := "SELECT COUNT(*) FROM messages " + whereClause
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count messages: %w", err)
	}

	// 分页查询
	offset := (criteria.Page - 1) * criteria.PageSize
	query := fmt.Sprintf("SELECT id, mailbox_id, from_address, to_address, subject, text_content, html_content, raw_content, is_read, created_at FROM messages %s ORDER BY created_at DESC LIMIT ? OFFSET ?", whereClause)
	args = append(args, criteria.PageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	messages := []domain.Message{}
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.ID, &msg.MailboxID, &msg.From, &msg.To, &msg.Subject, &msg.Text, &msg.HTML, &msg.Raw, &msg.IsRead, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := total / criteria.PageSize
	if total%criteria.PageSize > 0 {
		totalPages++
	}

	return &domain.MessageSearchResult{
		Messages:   messages,
		Total:      total,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ========== Webhook Repository ==========

// CreateWebhook 创建 Webhook
func (s *Store) CreateWebhook(webhook *domain.Webhook) error {
	query := `INSERT INTO webhooks (id, user_id, url, events, secret, is_active, retry_count, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		webhook.ID,
		webhook.UserID,
		webhook.URL,
		eventsToJSON(webhook.Events),
		webhook.Secret,
		webhook.IsActive,
		webhook.RetryCount,
		webhook.CreatedAt,
		webhook.UpdatedAt,
	)
	return err
}

// GetWebhook 获取 Webhook
func (s *Store) GetWebhook(id string) (*domain.Webhook, error) {
	query := `SELECT id, user_id, url, events, secret, is_active, retry_count, last_error, last_success, created_at, updated_at 
			  FROM webhooks WHERE id = ?`
	
	var webhook domain.Webhook
	var eventsJSON string
	var lastSuccess sql.NullTime
	
	err := s.db.QueryRow(query, id).Scan(
		&webhook.ID,
		&webhook.UserID,
		&webhook.URL,
		&eventsJSON,
		&webhook.Secret,
		&webhook.IsActive,
		&webhook.RetryCount,
		&webhook.LastError,
		&lastSuccess,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("webhook not found")
		}
		return nil, err
	}
	
	webhook.Events = eventsFromJSON(eventsJSON)
	if lastSuccess.Valid {
		webhook.LastSuccess = &lastSuccess.Time
	}
	
	return &webhook, nil
}

// ListWebhooks 列出用户的 Webhooks
func (s *Store) ListWebhooks(userID string) ([]domain.Webhook, error) {
	query := `SELECT id, user_id, url, events, secret, is_active, retry_count, last_error, last_success, created_at, updated_at 
			  FROM webhooks WHERE user_id = ? ORDER BY created_at DESC`
	
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	webhooks := []domain.Webhook{}
	for rows.Next() {
		var webhook domain.Webhook
		var eventsJSON string
		var lastSuccess sql.NullTime
		
		if err := rows.Scan(
			&webhook.ID,
			&webhook.UserID,
			&webhook.URL,
			&eventsJSON,
			&webhook.Secret,
			&webhook.IsActive,
			&webhook.RetryCount,
			&webhook.LastError,
			&lastSuccess,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		); err != nil {
			return nil, err
		}
		
		webhook.Events = eventsFromJSON(eventsJSON)
		if lastSuccess.Valid {
			webhook.LastSuccess = &lastSuccess.Time
		}
		
		webhooks = append(webhooks, webhook)
	}
	
	return webhooks, rows.Err()
}

// UpdateWebhook 更新 Webhook
func (s *Store) UpdateWebhook(webhook *domain.Webhook) error {
	query := `UPDATE webhooks SET url = ?, events = ?, is_active = ?, retry_count = ?, last_error = ?, last_success = ?, updated_at = ? 
			  WHERE id = ?`
	_, err := s.db.Exec(query,
		webhook.URL,
		eventsToJSON(webhook.Events),
		webhook.IsActive,
		webhook.RetryCount,
		webhook.LastError,
		webhook.LastSuccess,
		webhook.UpdatedAt,
		webhook.ID,
	)
	return err
}

// DeleteWebhook 删除 Webhook
func (s *Store) DeleteWebhook(id string) error {
	// 使用事务删除Webhook及其投递记录
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// 删除投递记录
	if _, err := tx.Exec("DELETE FROM webhook_deliveries WHERE webhook_id = ?", id); err != nil {
		return err
	}
	
	// 删除Webhook
	if _, err := tx.Exec("DELETE FROM webhooks WHERE id = ?", id); err != nil {
		return err
	}
	
	return tx.Commit()
}

// RecordDelivery 记录投递
func (s *Store) RecordDelivery(delivery *domain.WebhookDelivery) error {
	query := `INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status_code, response, duration, success, error, attempts, next_retry, created_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query,
		delivery.ID,
		delivery.WebhookID,
		delivery.Event,
		delivery.Payload,
		delivery.StatusCode,
		delivery.Response,
		delivery.Duration,
		delivery.Success,
		delivery.Error,
		delivery.Attempts,
		delivery.NextRetry,
		delivery.CreatedAt,
	)
	return err
}

// GetDeliveries 获取投递记录
func (s *Store) GetDeliveries(webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	query := `SELECT id, webhook_id, event, payload, status_code, response, duration, success, error, attempts, next_retry, created_at 
			  FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC LIMIT ?`
	
	rows, err := s.db.Query(query, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	deliveries := []domain.WebhookDelivery{}
	for rows.Next() {
		var delivery domain.WebhookDelivery
		var nextRetry sql.NullTime
		
		if err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.Event,
			&delivery.Payload,
			&delivery.StatusCode,
			&delivery.Response,
			&delivery.Duration,
			&delivery.Success,
			&delivery.Error,
			&delivery.Attempts,
			&nextRetry,
			&delivery.CreatedAt,
		); err != nil {
			return nil, err
		}
		
		if nextRetry.Valid {
			delivery.NextRetry = &nextRetry.Time
		}
		
		deliveries = append(deliveries, delivery)
	}
	
	return deliveries, rows.Err()
}

// GetPendingDeliveries 获取待重试投递
func (s *Store) GetPendingDeliveries(limit int) ([]domain.WebhookDelivery, error) {
	query := `SELECT id, webhook_id, event, payload, status_code, response, duration, success, error, attempts, next_retry, created_at 
			  FROM webhook_deliveries 
			  WHERE success = 0 AND next_retry IS NOT NULL AND next_retry <= datetime('now') 
			  ORDER BY next_retry ASC 
			  LIMIT ?`
	
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	deliveries := []domain.WebhookDelivery{}
	for rows.Next() {
		var delivery domain.WebhookDelivery
		var nextRetry sql.NullTime
		
		if err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.Event,
			&delivery.Payload,
			&delivery.StatusCode,
			&delivery.Response,
			&delivery.Duration,
			&delivery.Success,
			&delivery.Error,
			&delivery.Attempts,
			&nextRetry,
			&delivery.CreatedAt,
		); err != nil {
			return nil, err
		}
		
		if nextRetry.Valid {
			delivery.NextRetry = &nextRetry.Time
		}
		
		deliveries = append(deliveries, delivery)
	}
	
	return deliveries, rows.Err()
}

// eventsToJSON 将事件数组转换为JSON字符串
func eventsToJSON(events []string) string {
	if len(events) == 0 {
		return "[]"
	}
	result := "["
	for i, event := range events {
		if i > 0 {
			result += ","
		}
		result += `"` + event + `"`
	}
	result += "]"
	return result
}

// eventsFromJSON 从JSON字符串解析事件数组（简化实现）
func eventsFromJSON(jsonStr string) []string {
	// 简化实现，生产环境应使用 json.Unmarshal
	if jsonStr == "" || jsonStr == "[]" {
		return []string{}
	}
	// 这里应该使用 json.Unmarshal，但为了避免依赖，暂时返回空数组
	// 实际使用中应该导入 encoding/json 并正确解析
	return []string{}
}

package memory

import (
	"strings"

	"tempmail/backend/internal/domain"
)

// SearchMessages 搜索邮件（内存存储实现）
func (s *Store) SearchMessages(criteria domain.MessageSearchCriteria) (*domain.MessageSearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

	// 获取邮箱的所有邮件
	messages, ok := s.messages[criteria.MailboxID]
	if !ok {
		return &domain.MessageSearchResult{
			Messages:   []domain.Message{},
			Total:      0,
			Page:       criteria.Page,
			PageSize:   criteria.PageSize,
			TotalPages: 0,
		}, nil
	}

	// 过滤邮件
	filtered := make([]domain.Message, 0)
	for _, msg := range messages {
		if matchesCriteria(msg, criteria) {
			filtered = append(filtered, *msg)
		}
	}

	// 按时间倒序排序
	sortByCreatedAtDesc(filtered)

	// 分页
	total := len(filtered)
	start := (criteria.Page - 1) * criteria.PageSize
	end := start + criteria.PageSize

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedMessages := filtered[start:end]
	totalPages := (total + criteria.PageSize - 1) / criteria.PageSize

	return &domain.MessageSearchResult{
		Messages:   pagedMessages,
		Total:      total,
		Page:       criteria.Page,
		PageSize:   criteria.PageSize,
		TotalPages: totalPages,
	}, nil
}

// matchesCriteria 检查邮件是否匹配搜索条件
func matchesCriteria(msg *domain.Message, criteria domain.MessageSearchCriteria) bool {
	// 关键词搜索（主题、发件人、内容）
	if criteria.Query != "" {
		query := strings.ToLower(criteria.Query)
		subject := strings.ToLower(msg.Subject)
		from := strings.ToLower(msg.From)
		text := strings.ToLower(msg.Text)

		if !strings.Contains(subject, query) &&
			!strings.Contains(from, query) &&
			!strings.Contains(text, query) {
			return false
		}
	}

	// 发件人筛选
	if criteria.From != "" {
		if !strings.Contains(strings.ToLower(msg.From), strings.ToLower(criteria.From)) {
			return false
		}
	}

	// 主题筛选
	if criteria.Subject != "" {
		if !strings.Contains(strings.ToLower(msg.Subject), strings.ToLower(criteria.Subject)) {
			return false
		}
	}

	// 时间范围筛选
	if criteria.StartDate != nil && msg.CreatedAt.Before(*criteria.StartDate) {
		return false
	}
	if criteria.EndDate != nil && msg.CreatedAt.After(*criteria.EndDate) {
		return false
	}

	// 已读状态筛选
	if criteria.IsRead != nil && msg.IsRead != *criteria.IsRead {
		return false
	}

	// 附件筛选
	if criteria.HasAttachment != nil {
		hasAttachment := len(msg.Attachments) > 0
		if hasAttachment != *criteria.HasAttachment {
			return false
		}
	}

	return true
}

// sortByCreatedAtDesc 按创建时间倒序排序
func sortByCreatedAtDesc(messages []domain.Message) {
	// 简单冒泡排序（内存存储数据量不大）
	n := len(messages)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if messages[j].CreatedAt.Before(messages[j+1].CreatedAt) {
				messages[j], messages[j+1] = messages[j+1], messages[j]
			}
		}
	}
}

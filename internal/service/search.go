package service

import (
	"time"

	"tempmail/backend/internal/domain"
)

// SearchService 搜索服务
type SearchService struct {
	store domain.Store
}

// NewSearchService 创建搜索服务
func NewSearchService(store domain.Store) *SearchService {
	return &SearchService{
		store: store,
	}
}

// SearchMessagesInput 搜索邮件输入
type SearchMessagesInput struct {
	MailboxID     string     // 邮箱ID（必填）
	Query         string     // 搜索关键词
	From          string     // 发件人筛选
	Subject       string     // 主题筛选
	StartDate     *time.Time // 开始日期
	EndDate       *time.Time // 结束日期
	IsRead        *bool      // 是否已读
	HasAttachment *bool      // 是否有附件
	Page          int        // 页码
	PageSize      int        // 每页数量
}

// SearchMessages 搜索邮件
//
// 参数:
//   - input: 搜索条件
//
// 返回值:
//   - *domain.MessageSearchResult: 搜索结果
//   - error: 错误信息
func (s *SearchService) SearchMessages(input SearchMessagesInput) (*domain.MessageSearchResult, error) {
	// 构建搜索条件
	criteria := domain.MessageSearchCriteria{
		MailboxID:     input.MailboxID,
		Query:         input.Query,
		From:          input.From,
		Subject:       input.Subject,
		StartDate:     input.StartDate,
		EndDate:       input.EndDate,
		IsRead:        input.IsRead,
		HasAttachment: input.HasAttachment,
		Page:          input.Page,
		PageSize:      input.PageSize,
	}

	// 执行搜索
	return s.store.SearchMessages(criteria)
}

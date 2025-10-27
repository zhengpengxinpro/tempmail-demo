package domain

import "time"

// MessageSearchCriteria 邮件搜索条件
type MessageSearchCriteria struct {
	MailboxID   string     // 邮箱ID（必填）
	Query       string     // 搜索关键词（搜索主题、发件人、内容）
	From        string     // 发件人筛选
	Subject     string     // 主题筛选
	StartDate   *time.Time // 开始日期
	EndDate     *time.Time // 结束日期
	IsRead      *bool      // 是否已读
	HasAttachment *bool    // 是否有附件
	Page        int        // 页码（默认1）
	PageSize    int        // 每页数量（默认20，最大100）
}

// MessageSearchResult 邮件搜索结果
type MessageSearchResult struct {
	Messages   []Message `json:"messages"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	TotalPages int       `json:"totalPages"`
}

// MessageSearchRepository 邮件搜索仓储接口
type MessageSearchRepository interface {
	// SearchMessages 搜索邮件
	SearchMessages(criteria MessageSearchCriteria) (*MessageSearchResult, error)
}

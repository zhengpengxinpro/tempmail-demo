package domain

import "time"

// Tag 邮件标签
type Tag struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID      string    `json:"userId" gorm:"type:varchar(36);index;not null"`      // 所属用户
	Name        string    `json:"name"`        // 标签名称
	Color       string    `json:"color"`       // 标签颜色（十六进制）
	Description string    `json:"description"` // 标签描述
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// MessageTag 邮件-标签关联
type MessageTag struct {
	MessageID string    `json:"messageId" gorm:"type:varchar(36);primaryKey"`
	TagID     string    `json:"tagId" gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `json:"createdAt"`
}

// TagWithCount 带计数的标签
type TagWithCount struct {
	Tag
	MessageCount int `json:"messageCount"` // 该标签下的邮件数量
}

// TagRepository 标签仓储接口
type TagRepository interface {
	// CreateTag 创建标签
	CreateTag(tag *Tag) error
	
	// GetTag 获取标签
	GetTag(id string) (*Tag, error)
	
	// GetTagByName 根据名称获取标签
	GetTagByName(userID, name string) (*Tag, error)
	
	// ListTags 列出用户的所有标签
	ListTags(userID string) ([]TagWithCount, error)
	
	// UpdateTag 更新标签
	UpdateTag(tag *Tag) error
	
	// DeleteTag 删除标签
	DeleteTag(id string) error
	
	// AddMessageTag 为邮件添加标签
	AddMessageTag(messageID, tagID string) error
	
	// RemoveMessageTag 移除邮件标签
	RemoveMessageTag(messageID, tagID string) error
	
	// GetMessageTags 获取邮件的所有标签
	GetMessageTags(messageID string) ([]Tag, error)
	
	// ListMessagesByTag 列出标签下的所有邮件
	ListMessagesByTag(tagID string) ([]Message, error)
	
	// DeleteMessageTags 删除邮件的所有标签
	DeleteMessageTags(messageID string) error
}

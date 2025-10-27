package domain

import (
	"time"
)

// Mailbox 表示临时邮箱的业务实体。
type Mailbox struct {
	ID         string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Address    string     `json:"address" gorm:"type:varchar(255);uniqueIndex"`
	LocalPart  string     `json:"localPart" gorm:"type:varchar(255)"`
	Domain     string     `json:"domain" gorm:"type:varchar(100);index"`
	Token      string     `json:"token" gorm:"type:varchar(255);uniqueIndex"`
	UserID     *string    `json:"userId,omitempty" gorm:"type:varchar(36);index"` // 关联的用户ID（可选，游客模式为nil）
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	IPSource   string     `json:"-"`
	TotalCount int        `json:"totalCount"`
	Unread     int        `json:"unread"`
}

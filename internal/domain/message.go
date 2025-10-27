package domain

import "time"

// Message 表示一封临时邮箱内的邮件。
type Message struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	MailboxID  string    `json:"mailboxId" gorm:"type:varchar(36);index;not null"`
	From       string    `json:"from" gorm:"type:varchar(255)"`
	To         string    `json:"to" gorm:"type:varchar(255)"`
	Subject    string    `json:"subject" gorm:"type:varchar(500)"`
	CreatedAt  time.Time `json:"createdAt"`
	IsRead     bool      `json:"isRead" gorm:"default:false;index"`
	ReceivedAt time.Time `json:"receivedAt"`
	// 文件系统存储标记
	HasRaw  bool `json:"hasRaw" gorm:"default:false"`
	HasHTML bool `json:"hasHtml" gorm:"default:false"`
	HasText bool `json:"hasText" gorm:"default:false"`
	// 内容字段（不存数据库，从文件系统加载）
	Text        string        `json:"text,omitempty" gorm:"-"`
	HTML        string        `json:"html,omitempty" gorm:"-"`
	Raw         string        `json:"raw,omitempty" gorm:"-"`
	Attachments []*Attachment `json:"attachments,omitempty" gorm:"-"` // 邮件附件列表
}

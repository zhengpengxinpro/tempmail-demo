package domain

import "time"

// MailboxAlias 表示邮箱别名。
// 别名允许一个邮箱有多个接收地址，所有发送到别名的邮件都会转发到主邮箱。
type MailboxAlias struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`        // 别名唯一标识
	MailboxID string    `json:"mailboxId" gorm:"type:varchar(36);index;not null"` // 关联的主邮箱ID
	Address   string    `json:"address" gorm:"type:varchar(255);index"`   // 别名地址
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	IsActive  bool      `json:"isActive"`  // 是否启用
}

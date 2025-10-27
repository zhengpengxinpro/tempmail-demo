package domain

import "time"

// APIKey API密钥实体
type APIKey struct {
	ID         string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID     string     `json:"userId" gorm:"type:varchar(36);index;not null"`
	Key        string     `json:"key" gorm:"column:key_hash;type:varchar(255);uniqueIndex;not null"`      // API密钥
	KeyPrefix  string     `json:"keyPrefix" gorm:"type:varchar(20);not null"` // 密钥前缀（用于快速查找）
	Name       string     `json:"name" gorm:"type:varchar(100)"`     // 密钥名称/描述
	Scopes     *string    `json:"scopes,omitempty" gorm:"type:json"` // 权限范围
	IsActive   bool       `json:"isActive"` // 是否激活
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`  // 过期时间（可选）
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"` // 最后使用时间
}

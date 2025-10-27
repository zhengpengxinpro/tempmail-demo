package domain

import "time"

// DomainMode 域名模式
type DomainMode string

const (
	// DomainModeShared 共享模式（免费）- 任何人都可以创建该域名下的邮箱
	DomainModeShared DomainMode = "shared"
	// DomainModeExclusive 独享模式（付费）- 只有所有者可以创建该域名下的邮箱
	DomainModeExclusive DomainMode = "exclusive"
	// DomainModeCatchAll 通配模式 - 捕获所有发往该域名的邮件
	DomainModeCatchAll DomainMode = "catch_all"
)

// DomainStatus 域名状态
type DomainStatus string

const (
	// DomainStatusPending 待验证
	DomainStatusPending DomainStatus = "pending"
	// DomainStatusVerified 已验证
	DomainStatusVerified DomainStatus = "verified"
	// DomainStatusFailed 验证失败
	DomainStatusFailed DomainStatus = "failed"
	// DomainStatusExpired 已过期
	DomainStatusExpired DomainStatus = "expired"
)

// UserDomain 用户自定义域名
type UserDomain struct {
	ID           string       `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID       string       `json:"userId" gorm:"type:varchar(36);index;not null"`
	Domain       string       `json:"domain" gorm:"uniqueIndex;type:varchar(100);not null"`
	Mode         DomainMode   `json:"mode" gorm:"type:varchar(20);default:'shared'"`
	Status       DomainStatus `json:"status" gorm:"type:varchar(20);default:'pending';index"`
	VerifyToken  string       `json:"verifyToken" gorm:"type:varchar(255)"`
	VerifyMethod string       `json:"verifyMethod" gorm:"type:varchar(20);default:'dns_txt'"`
	VerifiedAt   *time.Time   `json:"verifiedAt"`
	LastCheckAt  *time.Time   `json:"lastCheckAt"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt" gorm:"autoUpdateTime"`
	ExpiresAt    *time.Time   `json:"expiresAt"`
	MXRecords    []string     `json:"mxRecords" gorm:"serializer:json;type:json"`
	IsActive     bool         `json:"isActive" gorm:"default:false;index"`
	MailboxCount int          `json:"mailboxCount" gorm:"default:0"`
	MonthlyFee   float64      `json:"monthlyFee" gorm:"type:decimal(10,2);default:0.00"`
	Notes        string       `json:"notes,omitempty" gorm:"type:text"`
}

// UserDomainRepository 用户域名仓储接口
type UserDomainRepository interface {
	// SaveUserDomain 保存用户域名
	SaveUserDomain(domain *UserDomain) error

	// GetUserDomain 根据 ID 获取用户域名
	GetUserDomain(id string) (*UserDomain, error)

	// GetUserDomainByDomain 根据域名获取
	GetUserDomainByDomain(domain string) (*UserDomain, error)

	// ListUserDomainsByUserID 获取用户的所有域名
	ListUserDomainsByUserID(userID string) ([]*UserDomain, error)

	// ListAllUserDomains 获取所有用户域名
	ListAllUserDomains() ([]*UserDomain, error)

	// DeleteUserDomain 删除用户域名
	DeleteUserDomain(id string) error

	// IncrementMailboxCount 增加邮箱计数
	IncrementMailboxCount(domain string) error

	// DecrementMailboxCount 减少邮箱计数
	DecrementMailboxCount(domain string) error
}

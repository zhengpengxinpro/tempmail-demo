package domain

import "time"

// SystemDomainStatus 系统域名状态
type SystemDomainStatus string

const (
	// SystemDomainStatusPending 待验证
	SystemDomainStatusPending SystemDomainStatus = "pending"
	// SystemDomainStatusVerified 已验证
	SystemDomainStatusVerified SystemDomainStatus = "verified"
	// SystemDomainStatusFailed 验证失败
	SystemDomainStatusFailed SystemDomainStatus = "failed"
)

// SystemDomain 系统域名（公共域名，所有用户可用）
type SystemDomain struct {
	ID           string               `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Domain       string               `json:"domain" gorm:"uniqueIndex;type:varchar(100);not null"`
	Status       SystemDomainStatus   `json:"status" gorm:"type:varchar(20);default:'pending';index"`
	VerifyToken  string               `json:"verifyToken" gorm:"type:varchar(255)"`
	VerifyMethod string               `json:"verifyMethod" gorm:"type:varchar(20);default:'dns_txt'"`
	VerifiedAt   *time.Time           `json:"verifiedAt"`
	LastCheckAt  *time.Time           `json:"lastCheckAt"`
	CreatedAt    time.Time            `json:"createdAt"`
	CreatedBy    string               `json:"createdBy" gorm:"type:varchar(36)"`
	IsActive     bool                 `json:"isActive" gorm:"default:false;index"`
	IsDefault    bool                 `json:"isDefault" gorm:"default:false;index"`
	MXRecords    []string             `json:"mxRecords" gorm:"serializer:json;type:json"`
	MailboxCount int                  `json:"mailboxCount" gorm:"default:0"`
	Notes        string               `json:"notes" gorm:"type:text"`
}

// SystemDomainRepository 系统域名仓储接口
type SystemDomainRepository interface {
	// SaveSystemDomain 保存系统域名
	SaveSystemDomain(domain *SystemDomain) error

	// GetSystemDomain 根据 ID 获取系统域名
	GetSystemDomain(id string) (*SystemDomain, error)

	// GetSystemDomainByDomain 根据域名获取
	GetSystemDomainByDomain(domain string) (*SystemDomain, error)

	// ListSystemDomains 获取所有系统域名
	ListSystemDomains() ([]*SystemDomain, error)

	// ListActiveSystemDomains 获取所有已激活的系统域名
	ListActiveSystemDomains() ([]*SystemDomain, error)

	// DeleteSystemDomain 删除系统域名
	DeleteSystemDomain(id string) error

	// IncrementSystemDomainMailboxCount 增加系统域名邮箱计数
	IncrementSystemDomainMailboxCount(domain string) error

	// DecrementSystemDomainMailboxCount 减少系统域名邮箱计数
	DecrementSystemDomainMailboxCount(domain string) error

	// DeleteUnverifiedSystemDomains 删除指定时间前创建且未验证的域名
	DeleteUnverifiedSystemDomains(before time.Time) (int, error)
}

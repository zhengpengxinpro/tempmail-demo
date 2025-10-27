package domain

import "time"

// UserTier 用户等级
type UserTier string

const (
	TierFree       UserTier = "free"
	TierBasic      UserTier = "basic"
	TierPro        UserTier = "pro"
	TierEnterprise UserTier = "enterprise"
)

// UserRole 用户角色
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
	RoleSuper UserRole = "super" // 超级管理员
)

// User 表示注册用户的业务实体
type User struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Email           string     `json:"email" gorm:"uniqueIndex;type:varchar(255);not null"`
	Username        string     `json:"username,omitempty" gorm:"type:varchar(100)"`
	PasswordHash    string     `json:"-" gorm:"type:varchar(255)"` // 不返回给前端
	Role            UserRole   `json:"role" gorm:"type:varchar(20);default:'user';index"`
	Tier            UserTier   `json:"tier" gorm:"type:varchar(20);default:'free';index"`
	IsActive        bool       `json:"isActive" gorm:"default:true"`
	IsEmailVerified bool       `json:"isEmailVerified" gorm:"default:false"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	LastLoginAt     *time.Time `json:"lastLoginAt,omitempty"`
}

// IsAdmin 判断用户是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleSuper
}

// IsSuper 判断用户是否为超级管理员
func (u *User) IsSuper() bool {
	return u.Role == RoleSuper
}

// Quota 用户配额
type Quota struct {
	UserID                  string `json:"userId"`
	MaxMailboxes            int    `json:"maxMailboxes"`
	MaxMessagesPerMailbox   int    `json:"maxMessagesPerMailbox"`
	MaxAPIRequestsPerMinute int    `json:"maxApiRequestsPerMinute"`
	MaxConcurrentRequests   int    `json:"maxConcurrentRequests"`
}

// DefaultQuotas 返回不同等级的默认配额
func DefaultQuotas(tier UserTier) Quota {
	switch tier {
	case TierBasic:
		return Quota{
			MaxMailboxes:            10,
			MaxMessagesPerMailbox:   100,
			MaxAPIRequestsPerMinute: 100,
			MaxConcurrentRequests:   20,
		}
	case TierPro:
		return Quota{
			MaxMailboxes:            50,
			MaxMessagesPerMailbox:   500,
			MaxAPIRequestsPerMinute: 500,
			MaxConcurrentRequests:   50,
		}
	case TierEnterprise:
		return Quota{
			MaxMailboxes:            -1, // 无限制
			MaxMessagesPerMailbox:   -1,
			MaxAPIRequestsPerMinute: -1,
			MaxConcurrentRequests:   100,
		}
	default: // TierFree
		return Quota{
			MaxMailboxes:            3,
			MaxMessagesPerMailbox:   30,
			MaxAPIRequestsPerMinute: 30,
			MaxConcurrentRequests:   5,
		}
	}
}

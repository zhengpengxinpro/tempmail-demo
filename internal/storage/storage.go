package storage

import (
	"errors"
	"tempmail/backend/internal/domain"
	"time"
)

var (
	// ErrAttachmentNotFound 附件未找到错误
	ErrAttachmentNotFound = errors.New("attachment not found")
	// ErrAliasNotFound 别名未找到错误
	ErrAliasNotFound = errors.New("alias not found")
	// ErrAliasExists 别名已存在错误
	ErrAliasExists = errors.New("alias already exists")
)

// MailboxRepository 定义邮箱数据存取操作。
type MailboxRepository interface {
	SaveMailbox(mailbox *domain.Mailbox) error
	GetMailbox(id string) (*domain.Mailbox, error)
	GetMailboxByAddress(address string) (*domain.Mailbox, error)
	ListMailboxes() []domain.Mailbox
	ListMailboxesByUserID(userID string) []domain.Mailbox // 按用户ID查询邮箱
	DeleteMailbox(id string) error
	DeleteExpiredMailboxes() (int, error) // 删除过期邮箱，返回删除数量
}

// MessageRepository 定义邮件数据存取操作。
type MessageRepository interface {
	SaveMessage(message *domain.Message) error
	ListMessages(mailboxID string) ([]domain.Message, error)
	GetMessage(mailboxID, messageID string) (*domain.Message, error)
	MarkMessageRead(mailboxID, messageID string) error
	DeleteMessage(mailboxID, messageID string) error
	DeleteAllMessages(mailboxID string) (int, error) // 删除邮箱所有消息，返回删除数量
	SearchMessages(criteria domain.MessageSearchCriteria) (*domain.MessageSearchResult, error)
}

// AliasRepository 定义邮箱别名数据存取操作。
type AliasRepository interface {
	SaveAlias(alias *domain.MailboxAlias) error
	GetAlias(aliasID string) (*domain.MailboxAlias, error)
	GetAliasByAddress(address string) (*domain.MailboxAlias, error)
	ListAliasesByMailboxID(mailboxID string) ([]*domain.MailboxAlias, error)
	DeleteAlias(aliasID string) error
}

// UserRepository 定义用户数据存取操作。
type UserRepository interface {
	CreateUser(user *domain.User) error
	GetUserByID(id string) (*domain.User, error)
	GetUserByEmail(email string) (*domain.User, error)
	GetUserByUsername(username string) (*domain.User, error)
	GetUserByAPIKey(apiKey string) (*domain.User, error)
	UpdateUser(user *domain.User) error
	UpdateLastLogin(userID string) error
}

// AdminRepository 定义管理员数据存取操作。
type AdminRepository interface {
	ListUsers(page, pageSize int, search string, role *domain.UserRole, tier *domain.UserTier, isActive *bool) ([]domain.User, int, error)
	DeleteUser(userID string) error
	DeleteMailboxesByUserID(userID string) error
	GetSystemStatistics() (*domain.SystemStatistics, error)
	GetDomainStatistics(domain string) (mailboxCount, messageCount int, err error)
}

// UserDomainRepository 定义用户域名数据存取操作。
type UserDomainRepository interface {
	SaveUserDomain(userDomain *domain.UserDomain) error
	GetUserDomain(domainID string) (*domain.UserDomain, error)
	GetUserDomainByDomain(domain string) (*domain.UserDomain, error)
	ListUserDomainsByUserID(userID string) ([]*domain.UserDomain, error)
	UpdateUserDomain(userDomain *domain.UserDomain) error
	DeleteUserDomain(domainID string) error
	IncrementMailboxCount(domain string) error
	DecrementMailboxCount(domain string) error
}

// SystemDomainRepository 定义系统域名数据存取操作。
type SystemDomainRepository interface {
	SaveSystemDomain(sysDomain *domain.SystemDomain) error
	GetSystemDomain(domainID string) (*domain.SystemDomain, error)
	GetSystemDomainByDomain(domain string) (*domain.SystemDomain, error)
	ListSystemDomains() ([]*domain.SystemDomain, error)
	ListActiveSystemDomains() ([]*domain.SystemDomain, error)
	DeleteSystemDomain(domainID string) error
	IncrementSystemDomainMailboxCount(domain string) error
	DecrementSystemDomainMailboxCount(domain string) error
	DeleteUnverifiedSystemDomains(before time.Time) (int, error)
}

// APIKeyRepository 定义API Key数据存取操作。
type APIKeyRepository interface {
	SaveAPIKey(apiKey *domain.APIKey) error
	GetAPIKey(id string) (*domain.APIKey, error)
	GetAPIKeyByKey(key string) (*domain.APIKey, error)
	ListAPIKeysByUserID(userID string) ([]*domain.APIKey, error)
	DeleteAPIKey(id string) error
	UpdateAPIKeyLastUsed(id string) error
}

// JWTRepository 定义 JWT 黑名单操作。
type JWTRepository interface {
	AddToBlacklist(jti string, ttl time.Duration) error
	IsBlacklisted(jti string) (bool, error)
}

// RateLimitRepository 定义限流操作。
type RateLimitRepository interface {
	IncrementRateLimit(key string, window time.Duration) (int64, error)
	GetRateLimit(key string) (int64, error)
}

// SessionRepository 定义会话管理操作。
type SessionRepository interface {
	CacheSession(sessionID string, userID string, ttl time.Duration) error
	GetCachedSession(sessionID string) (string, error)
	DeleteCachedSession(sessionID string) error
}

// PubSubRepository 定义发布订阅操作。
type PubSubRepository interface {
	PublishNewMail(mailboxID string, message *domain.Message) error
	SubscribeNewMail(mailboxID string) interface{}
}

// WebhookRepository 定义 Webhook 数据存取操作。
type WebhookRepository interface {
	CreateWebhook(webhook *domain.Webhook) error
	GetWebhook(id string) (*domain.Webhook, error)
	ListWebhooks(userID string) ([]domain.Webhook, error)
	UpdateWebhook(webhook *domain.Webhook) error
	DeleteWebhook(id string) error
	RecordDelivery(delivery *domain.WebhookDelivery) error
	GetDeliveries(webhookID string, limit int) ([]domain.WebhookDelivery, error)
	GetPendingDeliveries(limit int) ([]domain.WebhookDelivery, error)
}

// TagRepository 定义标签数据存取操作。
type TagRepository interface {
	CreateTag(tag *domain.Tag) error
	GetTag(id string) (*domain.Tag, error)
	GetTagByName(userID, name string) (*domain.Tag, error)
	ListTags(userID string) ([]domain.TagWithCount, error)
	UpdateTag(tag *domain.Tag) error
	DeleteTag(id string) error
	AddMessageTag(messageID, tagID string) error
	RemoveMessageTag(messageID, tagID string) error
	GetMessageTags(messageID string) ([]domain.Tag, error)
	ListMessagesByTag(tagID string) ([]domain.Message, error)
	DeleteMessageTags(messageID string) error
}

// SystemConfigRepository 定义系统配置数据存取操作。
type SystemConfigRepository interface {
	GetSystemConfig() (*domain.SystemConfig, error)
	SaveSystemConfig(config *domain.SystemConfig) error
}

// Store 定义完整的存储接口。
type Store interface {
	MailboxRepository
	MessageRepository
	AliasRepository
	UserRepository
	AdminRepository
	UserDomainRepository
	SystemDomainRepository
	APIKeyRepository
	WebhookRepository
	TagRepository
	SystemConfigRepository
	JWTRepository
	RateLimitRepository
	SessionRepository
	PubSubRepository

	// 工具方法
	Close() error
	Health() error

	// 兼容 domain.Store 接口的方法
	ListAllUserDomains() ([]*domain.UserDomain, error)
	UpdateUserDomain(userDomain *domain.UserDomain) error
}

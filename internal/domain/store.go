package domain

import "time"

// Store 聚合所有存储接口
type Store interface {
	// ========== Mailbox Repository ==========
	SaveMailbox(mailbox *Mailbox) error
	GetMailbox(id string) (*Mailbox, error)
	GetMailboxByAddress(address string) (*Mailbox, error)
	ListMailboxes() []Mailbox
	ListMailboxesByUserID(userID string) []Mailbox
	DeleteMailbox(id string) error
	DeleteExpiredMailboxes() (int, error)
	DeleteMailboxesByUserID(userID string) error

	// ========== Message Repository ==========
	SaveMessage(message *Message) error
	ListMessages(mailboxID string) ([]Message, error)
	GetMessage(mailboxID, messageID string) (*Message, error)
	MarkMessageRead(mailboxID, messageID string) error
	SearchMessages(criteria MessageSearchCriteria) (*MessageSearchResult, error)

	// ========== User Repository ==========
	CreateUser(user *User) error
	GetUserByID(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	UpdateUser(user *User) error
	UpdateLastLogin(userID string) error

	// ========== Admin Repository ==========
	ListUsers(page, pageSize int, search string, role *UserRole, tier *UserTier, isActive *bool) ([]User, int, error)
	DeleteUser(userID string) error
	GetSystemStatistics() (*SystemStatistics, error)
	GetDomainStatistics(domain string) (mailboxCount, messageCount int, err error)

	// ========== User Domain Repository ==========
	SaveUserDomain(domain *UserDomain) error
	GetUserDomain(id string) (*UserDomain, error)
	GetUserDomainByDomain(domain string) (*UserDomain, error)
	ListUserDomainsByUserID(userID string) ([]*UserDomain, error)
	ListAllUserDomains() ([]*UserDomain, error)
	DeleteUserDomain(id string) error
	IncrementMailboxCount(domain string) error
	DecrementMailboxCount(domain string) error

	// ========== System Domain Repository ==========
	SaveSystemDomain(domain *SystemDomain) error
	GetSystemDomain(id string) (*SystemDomain, error)
	GetSystemDomainByDomain(domain string) (*SystemDomain, error)
	ListSystemDomains() ([]*SystemDomain, error)
	ListActiveSystemDomains() ([]*SystemDomain, error)
	DeleteSystemDomain(id string) error
	IncrementSystemDomainMailboxCount(domain string) error
	DecrementSystemDomainMailboxCount(domain string) error
	DeleteUnverifiedSystemDomains(before time.Time) (int, error)

	// ========== API Key Repository ==========
	SaveAPIKey(apiKey *APIKey) error
	GetAPIKey(id string) (*APIKey, error)
	GetAPIKeyByKey(key string) (*APIKey, error)
	ListAPIKeysByUserID(userID string) ([]*APIKey, error)
	DeleteAPIKey(id string) error
	UpdateAPIKeyLastUsed(id string) error
	GetUserByAPIKey(apiKey string) (*User, error)

	// ========== Webhook Repository ==========
	CreateWebhook(webhook *Webhook) error
	GetWebhook(id string) (*Webhook, error)
	ListWebhooks(userID string) ([]Webhook, error)
	UpdateWebhook(webhook *Webhook) error
	DeleteWebhook(id string) error
	RecordDelivery(delivery *WebhookDelivery) error
	GetDeliveries(webhookID string, limit int) ([]WebhookDelivery, error)
	GetPendingDeliveries(limit int) ([]WebhookDelivery, error)

	// ========== Tag Repository ==========
	CreateTag(tag *Tag) error
	GetTag(id string) (*Tag, error)
	GetTagByName(userID, name string) (*Tag, error)
	ListTags(userID string) ([]TagWithCount, error)
	UpdateTag(tag *Tag) error
	DeleteTag(id string) error
	AddMessageTag(messageID, tagID string) error
	RemoveMessageTag(messageID, tagID string) error
	GetMessageTags(messageID string) ([]Tag, error)
	ListMessagesByTag(tagID string) ([]Message, error)
	DeleteMessageTags(messageID string) error
}

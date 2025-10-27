package hybrid

import (
	"context"
	"fmt"
	"time"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage/postgres"
	"tempmail/backend/internal/storage/redis"
)

// Store 混合存储实现，结合 PostgreSQL 和 Redis
type Store struct {
	postgres *postgres.Store
	redis    *redis.Cache
	ctx      context.Context
}

// NewStore 创建混合存储实例 (PostgreSQL)
func NewStore(postgresDSN, redisAddr, redisPassword string, redisDB int) (*Store, error) {
	return NewStoreWithType("postgres", postgresDSN, redisAddr, redisPassword, redisDB)
}

// NewStoreWithType 创建混合存储实例（指定数据库类型）
func NewStoreWithType(dbType, dsn, redisAddr, redisPassword string, redisDB int) (*Store, error) {
	var dbStore *postgres.Store
	var err error

	// 根据数据库类型创建存储
	switch dbType {
	case "mysql":
		dbStore, err = postgres.NewMySQLStore(dsn)
	case "postgres", "postgresql":
		dbStore, err = postgres.NewStore(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported: mysql, postgres)", dbType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 初始化 Redis
	redisCache, err := redis.NewCache(redisAddr, redisPassword, redisDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis: %w", err)
	}

	return &Store{
		postgres: dbStore,
		redis:    redisCache,
		ctx:      context.Background(),
	}, nil
}

// ========== Mailbox Repository ==========

// SaveMailbox 保存邮箱信息
func (s *Store) SaveMailbox(mailbox *domain.Mailbox) error {
	// 保存到 PostgreSQL
	if err := s.postgres.SaveMailbox(mailbox); err != nil {
		return err
	}

	// 缓存到 Redis（24小时过期）
	return s.redis.CacheMailbox(mailbox, 24*time.Hour)
}

// GetMailbox 根据 ID 获取邮箱
func (s *Store) GetMailbox(id string) (*domain.Mailbox, error) {
	// 先尝试从 Redis 获取
	if mailbox, err := s.redis.GetCachedMailbox(id); err == nil {
		return mailbox, nil
	}

	// 从 PostgreSQL 获取
	mailbox, err := s.postgres.GetMailbox(id)
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis
	s.redis.CacheMailbox(mailbox, 24*time.Hour)
	return mailbox, nil
}

// GetMailboxByAddress 根据完整地址获取邮箱
func (s *Store) GetMailboxByAddress(address string) (*domain.Mailbox, error) {
	// 从 PostgreSQL 获取（地址查询不缓存，因为变化频繁）
	return s.postgres.GetMailboxByAddress(address)
}

// ListMailboxes 返回全部邮箱的快照
func (s *Store) ListMailboxes() []domain.Mailbox {
	// 直接从 PostgreSQL 获取（列表查询不缓存）
	return s.postgres.ListMailboxes()
}

// ListMailboxesByUserID 返回指定用户的全部邮箱
func (s *Store) ListMailboxesByUserID(userID string) []domain.Mailbox {
	// 直接从 PostgreSQL 获取
	return s.postgres.ListMailboxesByUserID(userID)
}

// DeleteMailbox 删除指定邮箱
func (s *Store) DeleteMailbox(id string) error {
	// 从 PostgreSQL 删除
	if err := s.postgres.DeleteMailbox(id); err != nil {
		return err
	}

	// 从 Redis 删除缓存
	s.redis.DeleteCachedMailbox(id)
	s.redis.DeleteCachedMessageList(id)

	return nil
}

// DeleteExpiredMailboxes 删除所有过期的邮箱，返回删除数量
func (s *Store) DeleteExpiredMailboxes() (int, error) {
	// 直接从 PostgreSQL 删除
	return s.postgres.DeleteExpiredMailboxes()
}

// ========== Message Repository ==========

// SaveMessage 保存邮件信息
func (s *Store) SaveMessage(message *domain.Message) error {
	// 保存到 PostgreSQL
	if err := s.postgres.SaveMessage(message); err != nil {
		return err
	}

	// 缓存邮件到 Redis（24小时过期）
	if err := s.redis.CacheMessage(message, 24*time.Hour); err != nil {
		// 缓存失败不影响主流程
		fmt.Printf("Warning: failed to cache message: %v\n", err)
	}

	// 删除邮件列表缓存（因为列表已变化）
	s.redis.DeleteCachedMessageList(message.MailboxID)

	// 发布新邮件通知
	s.redis.PublishNewMail(message.MailboxID, message)

	return nil
}

// ListMessages 返回某个邮箱下的全部邮件
func (s *Store) ListMessages(mailboxID string) ([]domain.Message, error) {
	// 先尝试从 Redis 获取
	if messages, err := s.redis.GetCachedMessageList(mailboxID); err == nil {
		return messages, nil
	}

	// 从 PostgreSQL 获取
	messages, err := s.postgres.ListMessages(mailboxID)
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis（1小时过期）
	s.redis.CacheMessageList(mailboxID, messages, 1*time.Hour)

	return messages, nil
}

// GetMessage 获取单封邮件
func (s *Store) GetMessage(mailboxID, messageID string) (*domain.Message, error) {
	// 先尝试从 Redis 获取
	if message, err := s.redis.GetCachedMessage(mailboxID, messageID); err == nil {
		return message, nil
	}

	// 从 PostgreSQL 获取
	message, err := s.postgres.GetMessage(mailboxID, messageID)
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis
	s.redis.CacheMessage(message, 24*time.Hour)

	return message, nil
}

// MarkMessageRead 将邮件标记为已读
func (s *Store) MarkMessageRead(mailboxID, messageID string) error {
	// 更新 PostgreSQL
	if err := s.postgres.MarkMessageRead(mailboxID, messageID); err != nil {
		return err
	}

	// 删除相关缓存
	s.redis.Delete(fmt.Sprintf("message:%s:%s", mailboxID, messageID))
	s.redis.DeleteCachedMessageList(mailboxID)

	return nil
}

// DeleteMessage 删除单封邮件
func (s *Store) DeleteMessage(mailboxID, messageID string) error {
	// 从 PostgreSQL 删除
	if err := s.postgres.DeleteMessage(mailboxID, messageID); err != nil {
		return err
	}

	// 删除 Redis 缓存
	s.redis.Delete(fmt.Sprintf("message:%s:%s", mailboxID, messageID))
	s.redis.DeleteCachedMessageList(mailboxID)

	return nil
}

// DeleteAllMessages 删除邮箱的所有邮件
func (s *Store) DeleteAllMessages(mailboxID string) (int, error) {
	// 从 PostgreSQL 删除
	count, err := s.postgres.DeleteAllMessages(mailboxID)
	if err != nil {
		return 0, err
	}

	// 删除 Redis 缓存
	s.redis.DeleteCachedMessageList(mailboxID)

	return count, nil
}

// GetAttachment 获取邮件附件
func (s *Store) GetAttachment(mailboxID, messageID, attachmentID string) (*domain.Attachment, error) {
	// 附件直接从 PostgreSQL 获取（不缓存，因为文件较大）
	return s.postgres.GetAttachment(mailboxID, messageID, attachmentID)
}

// SearchMessages 搜索邮件
func (s *Store) SearchMessages(criteria domain.MessageSearchCriteria) (*domain.MessageSearchResult, error) {
	// 直接从 PostgreSQL 搜索（搜索结果不缓存）
	return s.postgres.SearchMessages(criteria)
}

// ========== User Repository ==========

// CreateUser 创建新用户
func (s *Store) CreateUser(user *domain.User) error {
	// 保存到 PostgreSQL
	if err := s.postgres.CreateUser(user); err != nil {
		return err
	}

	// 不缓存用户，因为PasswordHash字段无法正确序列化

	return nil
}

// GetUserByID 根据ID获取用户
// 注意：不使用Redis缓存，因为PasswordHash字段有json:"-"标签，缓存后会丢失
func (s *Store) GetUserByID(id string) (*domain.User, error) {
	// 直接从 PostgreSQL 获取，不经过Redis缓存
	user, err := s.postgres.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
// 注意：不使用Redis缓存，因为PasswordHash字段有json:"-"标签，缓存后会丢失
func (s *Store) GetUserByEmail(email string) (*domain.User, error) {
	// 直接从 PostgreSQL 获取，不经过Redis缓存
	user, err := s.postgres.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *Store) GetUserByUsername(username string) (*domain.User, error) {
	return s.postgres.GetUserByUsername(username)
}

// GetUserByAPIKey 根据API Key获取用户
func (s *Store) GetUserByAPIKey(apiKey string) (*domain.User, error) {
	// 先尝试从 Redis 获取缓存的用户ID
	if userID, err := s.redis.GetCachedAPIKeyUser(apiKey); err == nil {
		return s.GetUserByID(userID)
	}

	// 从 PostgreSQL 获取
	user, err := s.postgres.GetUserByAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	// 缓存用户信息和API Key关联（1小时过期）
	s.redis.CacheUser(user, 1*time.Hour)
	s.redis.CacheAPIKeyUser(apiKey, user.ID, 1*time.Hour)

	return user, nil
}

// ========== API Key Repository ==========

// SaveAPIKey 保存API Key
func (s *Store) SaveAPIKey(apiKey *domain.APIKey) error {
	// 保存到 PostgreSQL
	if err := s.postgres.SaveAPIKey(apiKey); err != nil {
		return err
	}

	// 缓存到 Redis（24小时过期）
	s.redis.CacheAPIKey(apiKey, 24*time.Hour)
	s.redis.CacheAPIKeyUser(apiKey.Key, apiKey.UserID, 24*time.Hour)

	return nil
}

// GetAPIKey 根据ID获取API Key
func (s *Store) GetAPIKey(id string) (*domain.APIKey, error) {
	// 先尝试从 Redis 获取
	if apiKey, err := s.redis.GetCachedAPIKey(id); err == nil {
		return apiKey, nil
	}

	// 从 PostgreSQL 获取
	apiKey, err := s.postgres.GetAPIKey(id)
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis
	s.redis.CacheAPIKey(apiKey, 24*time.Hour)

	return apiKey, nil
}

// GetAPIKeyByKey 根据Key字符串获取API Key
func (s *Store) GetAPIKeyByKey(key string) (*domain.APIKey, error) {
	// API Key查询直接从 PostgreSQL 获取（安全考虑）
	apiKey, err := s.postgres.GetAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}

	// 更新最后使用时间
	s.postgres.UpdateAPIKeyLastUsed(apiKey.ID)

	return apiKey, nil
}

// ListAPIKeysByUserID 列出用户的所有API Key
func (s *Store) ListAPIKeysByUserID(userID string) ([]*domain.APIKey, error) {
	// API Key列表直接从 PostgreSQL 获取（不缓存）
	return s.postgres.ListAPIKeysByUserID(userID)
}

// DeleteAPIKey 删除API Key
func (s *Store) DeleteAPIKey(id string) error {
	// 从 PostgreSQL 删除
	if err := s.postgres.DeleteAPIKey(id); err != nil {
		return err
	}

	// 删除 Redis 缓存
	s.redis.Delete(fmt.Sprintf("apikey:%s", id))

	return nil
}

// UpdateAPIKeyLastUsed 更新API Key最后使用时间
func (s *Store) UpdateAPIKeyLastUsed(id string) error {
	// 更新 PostgreSQL
	if err := s.postgres.UpdateAPIKeyLastUsed(id); err != nil {
		return err
	}

	// 删除缓存（强制重新加载）
	s.redis.Delete(fmt.Sprintf("apikey:%s", id))

	return nil
}

// UpdateUser 更新用户信息
func (s *Store) UpdateUser(user *domain.User) error {
	// 更新 PostgreSQL
	if err := s.postgres.UpdateUser(user); err != nil {
		return err
	}

	// 不缓存用户，因为PasswordHash字段无法正确序列化

	return nil
}

// UpdateLastLogin 更新用户最后登录时间
func (s *Store) UpdateLastLogin(userID string) error {
	// 更新 PostgreSQL
	if err := s.postgres.UpdateLastLogin(userID); err != nil {
		return err
	}

	// 删除用户缓存（强制重新加载）
	s.redis.Delete(fmt.Sprintf("user:%s", userID))

	return nil
}

// ========== Admin Repository ==========

// ListUsers 列出用户（支持分页和过滤）
func (s *Store) ListUsers(page, pageSize int, search string, role *domain.UserRole, tier *domain.UserTier, isActive *bool) ([]domain.User, int, error) {
	// 管理功能直接从 PostgreSQL 获取（不缓存）
	return s.postgres.ListUsers(page, pageSize, search, role, tier, isActive)
}

// DeleteUser 删除用户
func (s *Store) DeleteUser(userID string) error {
	// 从 PostgreSQL 删除
	if err := s.postgres.DeleteUser(userID); err != nil {
		return err
	}

	// 删除 Redis 缓存
	s.redis.Delete(fmt.Sprintf("user:%s", userID))

	return nil
}

// DeleteMailboxesByUserID 删除用户的所有邮箱
func (s *Store) DeleteMailboxesByUserID(userID string) error {
	// 从 PostgreSQL 删除
	return s.postgres.DeleteMailboxesByUserID(userID)
}

// GetSystemStatistics 获取系统统计信息
func (s *Store) GetSystemStatistics() (*domain.SystemStatistics, error) {
	// 先尝试从 Redis 获取
	if stats, err := s.redis.GetCachedStatistics(); err == nil {
		return stats, nil
	}

	// 从 PostgreSQL 获取
	stats, err := s.postgres.GetSystemStatistics()
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis（5分钟过期）
	s.redis.CacheStatistics(stats, 5*time.Minute)

	return stats, nil
}

// GetDomainStatistics 获取域名统计信息
func (s *Store) GetDomainStatistics(domain string) (mailboxCount, messageCount int, err error) {
	// 域名统计直接从 PostgreSQL 获取
	return s.postgres.GetDomainStatistics(domain)
}

// ========== Alias Repository ==========

// SaveAlias 保存邮箱别名
func (s *Store) SaveAlias(alias *domain.MailboxAlias) error {
	// 保存到 PostgreSQL
	return s.postgres.SaveAlias(alias)
}

// GetAlias 根据ID获取别名
func (s *Store) GetAlias(aliasID string) (*domain.MailboxAlias, error) {
	// 别名直接从 PostgreSQL 获取（不缓存）
	return s.postgres.GetAlias(aliasID)
}

// GetAliasByAddress 根据地址获取别名
func (s *Store) GetAliasByAddress(address string) (*domain.MailboxAlias, error) {
	// 别名地址查询直接从 PostgreSQL 获取
	return s.postgres.GetAliasByAddress(address)
}

// ListAliasesByMailboxID 列出指定邮箱的所有别名
func (s *Store) ListAliasesByMailboxID(mailboxID string) ([]*domain.MailboxAlias, error) {
	// 别名列表直接从 PostgreSQL 获取
	return s.postgres.ListAliasesByMailboxID(mailboxID)
}

// DeleteAlias 删除别名
func (s *Store) DeleteAlias(aliasID string) error {
	// 从 PostgreSQL 删除
	return s.postgres.DeleteAlias(aliasID)
}

// ========== User Domain Repository ==========

// SaveUserDomain 保存用户域名
func (s *Store) SaveUserDomain(userDomain *domain.UserDomain) error {
	// 保存到 PostgreSQL
	return s.postgres.SaveUserDomain(userDomain)
}

// GetUserDomain 根据ID获取用户域名
func (s *Store) GetUserDomain(domainID string) (*domain.UserDomain, error) {
	// 用户域名直接从 PostgreSQL 获取
	return s.postgres.GetUserDomain(domainID)
}

// GetUserDomainByDomain 根据域名获取用户域名
func (s *Store) GetUserDomainByDomain(domain string) (*domain.UserDomain, error) {
	// 域名查询直接从 PostgreSQL 获取
	return s.postgres.GetUserDomainByDomain(domain)
}

// ListUserDomainsByUserID 列出用户的所有域名
func (s *Store) ListUserDomainsByUserID(userID string) ([]*domain.UserDomain, error) {
	// 用户域名列表直接从 PostgreSQL 获取
	return s.postgres.ListUserDomainsByUserID(userID)
}

// UpdateUserDomain 更新用户域名
func (s *Store) UpdateUserDomain(userDomain *domain.UserDomain) error {
	// 更新 PostgreSQL
	return s.postgres.UpdateUserDomain(userDomain)
}

// ListAllUserDomains 列出所有用户域名
func (s *Store) ListAllUserDomains() ([]*domain.UserDomain, error) {
	return s.postgres.ListAllUserDomains()
}

// DeleteUserDomain 删除用户域名
func (s *Store) DeleteUserDomain(domainID string) error {
	// 从 PostgreSQL 删除
	return s.postgres.DeleteUserDomain(domainID)
}

// IncrementMailboxCount 增加域名邮箱计数
func (s *Store) IncrementMailboxCount(domain string) error {
	// 更新 PostgreSQL
	return s.postgres.IncrementMailboxCount(domain)
}

// DecrementMailboxCount 减少域名邮箱计数
func (s *Store) DecrementMailboxCount(domain string) error {
	// 更新 PostgreSQL
	return s.postgres.DecrementMailboxCount(domain)
}

// ========== JWT 黑名单 ==========

// AddToBlacklist 将 JWT 添加到黑名单
func (s *Store) AddToBlacklist(jti string, ttl time.Duration) error {
	// 只使用 Redis 存储黑名单
	return s.redis.AddToBlacklist(jti, ttl)
}

// IsBlacklisted 检查 JWT 是否在黑名单中
func (s *Store) IsBlacklisted(jti string) (bool, error) {
	// 只从 Redis 检查黑名单
	return s.redis.IsBlacklisted(jti)
}

// ========== 限流 ==========

// IncrementRateLimit 增加限流计数
func (s *Store) IncrementRateLimit(key string, window time.Duration) (int64, error) {
	// 只使用 Redis 进行限流
	return s.redis.IncrementRateLimit(key, window)
}

// GetRateLimit 获取限流计数
func (s *Store) GetRateLimit(key string) (int64, error) {
	// 只从 Redis 获取限流计数
	return s.redis.GetRateLimit(key)
}

// ========== 会话管理 ==========

// CacheSession 缓存用户会话
func (s *Store) CacheSession(sessionID string, userID string, ttl time.Duration) error {
	// 只使用 Redis 存储会话
	return s.redis.CacheSession(sessionID, userID, ttl)
}

// GetCachedSession 获取缓存的会话
func (s *Store) GetCachedSession(sessionID string) (string, error) {
	// 只从 Redis 获取会话
	return s.redis.GetCachedSession(sessionID)
}

// DeleteCachedSession 删除缓存的会话
func (s *Store) DeleteCachedSession(sessionID string) error {
	// 只从 Redis 删除会话
	return s.redis.DeleteCachedSession(sessionID)
}

// ========== 发布订阅 ==========

// PublishNewMail 发布新邮件通知
func (s *Store) PublishNewMail(mailboxID string, message *domain.Message) error {
	// 使用 Redis 发布订阅
	return s.redis.PublishNewMail(mailboxID, message)
}

// SubscribeNewMail 订阅新邮件通知
func (s *Store) SubscribeNewMail(mailboxID string) interface{} {
	// 使用 Redis 发布订阅
	return s.redis.SubscribeNewMail(mailboxID)
}

// ========== 工具方法 ==========

// Close 关闭存储连接
func (s *Store) Close() error {
	// 关闭 PostgreSQL 连接
	if err := s.postgres.Close(); err != nil {
		return err
	}

	// 关闭 Redis 连接
	return s.redis.Close()
}

// Health 健康检查
func (s *Store) Health() error {
	// 检查 PostgreSQL 连接
	// 这里可以添加 PostgreSQL 健康检查逻辑

	// 检查 Redis 连接
	// 这里可以添加 Redis 健康检查逻辑

	return nil
}

// ========== Webhook Repository ==========

func (s *Store) CreateWebhook(webhook *domain.Webhook) error {
	return s.postgres.CreateWebhook(webhook)
}

func (s *Store) GetWebhook(id string) (*domain.Webhook, error) {
	return s.postgres.GetWebhook(id)
}

func (s *Store) ListWebhooks(userID string) ([]domain.Webhook, error) {
	return s.postgres.ListWebhooks(userID)
}

func (s *Store) UpdateWebhook(webhook *domain.Webhook) error {
	return s.postgres.UpdateWebhook(webhook)
}

func (s *Store) DeleteWebhook(id string) error {
	return s.postgres.DeleteWebhook(id)
}

func (s *Store) RecordDelivery(delivery *domain.WebhookDelivery) error {
	return s.postgres.RecordDelivery(delivery)
}

func (s *Store) GetDeliveries(webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	return s.postgres.GetDeliveries(webhookID, limit)
}

func (s *Store) GetPendingDeliveries(limit int) ([]domain.WebhookDelivery, error) {
	return s.postgres.GetPendingDeliveries(limit)
}

// ========== Tag Repository ==========

func (s *Store) CreateTag(tag *domain.Tag) error {
	return s.postgres.CreateTag(tag)
}

func (s *Store) GetTag(id string) (*domain.Tag, error) {
	return s.postgres.GetTag(id)
}

func (s *Store) GetTagByName(userID, name string) (*domain.Tag, error) {
	return s.postgres.GetTagByName(userID, name)
}

func (s *Store) ListTags(userID string) ([]domain.TagWithCount, error) {
	return s.postgres.ListTags(userID)
}

func (s *Store) UpdateTag(tag *domain.Tag) error {
	return s.postgres.UpdateTag(tag)
}

func (s *Store) DeleteTag(id string) error {
	return s.postgres.DeleteTag(id)
}

func (s *Store) AddMessageTag(messageID, tagID string) error {
	return s.postgres.AddMessageTag(messageID, tagID)
}

func (s *Store) RemoveMessageTag(messageID, tagID string) error {
	return s.postgres.RemoveMessageTag(messageID, tagID)
}

func (s *Store) GetMessageTags(messageID string) ([]domain.Tag, error) {
	return s.postgres.GetMessageTags(messageID)
}

func (s *Store) ListMessagesByTag(tagID string) ([]domain.Message, error) {
	return s.postgres.ListMessagesByTag(tagID)
}

func (s *Store) DeleteMessageTags(messageID string) error {
	return s.postgres.DeleteMessageTags(messageID)
}

// ========== System Config Repository ==========

func (s *Store) GetSystemConfig() (*domain.SystemConfig, error) {
	return s.postgres.GetSystemConfig()
}

func (s *Store) SaveSystemConfig(config *domain.SystemConfig) error {
	return s.postgres.SaveSystemConfig(config)
}

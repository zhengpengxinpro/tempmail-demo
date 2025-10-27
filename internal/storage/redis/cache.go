package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"tempmail/backend/internal/domain"
)

// Cache Redis 缓存实现
type Cache struct {
	client *redis.Client
	ctx    context.Context
}

// NewCache 创建 Redis 缓存实例
func NewCache(addr, password string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 10,
	})

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Cache{
		client: client,
		ctx:    ctx,
	}, nil
}

// ========== 邮箱缓存 ==========

// CacheMailbox 缓存邮箱信息
func (c *Cache) CacheMailbox(mailbox *domain.Mailbox, ttl time.Duration) error {
	key := fmt.Sprintf("mailbox:%s", mailbox.ID)
	data, err := json.Marshal(mailbox)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedMailbox 获取缓存的邮箱信息
func (c *Cache) GetCachedMailbox(mailboxID string) (*domain.Mailbox, error) {
	key := fmt.Sprintf("mailbox:%s", mailboxID)
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("mailbox not found in cache")
		}
		return nil, err
	}

	var mailbox domain.Mailbox
	if err := json.Unmarshal([]byte(data), &mailbox); err != nil {
		return nil, err
	}

	return &mailbox, nil
}

// DeleteCachedMailbox 删除缓存的邮箱信息
func (c *Cache) DeleteCachedMailbox(mailboxID string) error {
	key := fmt.Sprintf("mailbox:%s", mailboxID)
	return c.client.Del(c.ctx, key).Err()
}

// ========== 邮件缓存 ==========

// CacheMessage 缓存邮件信息
func (c *Cache) CacheMessage(message *domain.Message, ttl time.Duration) error {
	key := fmt.Sprintf("message:%s:%s", message.MailboxID, message.ID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedMessage 获取缓存的邮件信息
func (c *Cache) GetCachedMessage(mailboxID, messageID string) (*domain.Message, error) {
	key := fmt.Sprintf("message:%s:%s", mailboxID, messageID)
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("message not found in cache")
		}
		return nil, err
	}

	var message domain.Message
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		return nil, err
	}

	return &message, nil
}

// CacheMessageList 缓存邮件列表
func (c *Cache) CacheMessageList(mailboxID string, messages []domain.Message, ttl time.Duration) error {
	key := fmt.Sprintf("messages:%s", mailboxID)
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedMessageList 获取缓存的邮件列表
func (c *Cache) GetCachedMessageList(mailboxID string) ([]domain.Message, error) {
	key := fmt.Sprintf("messages:%s", mailboxID)
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("message list not found in cache")
		}
		return nil, err
	}

	var messages []domain.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// DeleteCachedMessageList 删除缓存的邮件列表
func (c *Cache) DeleteCachedMessageList(mailboxID string) error {
	key := fmt.Sprintf("messages:%s", mailboxID)
	return c.client.Del(c.ctx, key).Err()
}

// ========== 用户缓存 ==========

// CacheUser 缓存用户信息
func (c *Cache) CacheUser(user *domain.User, ttl time.Duration) error {
	key := fmt.Sprintf("user:%s", user.ID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedUser 获取缓存的用户信息
func (c *Cache) GetCachedUser(userID string) (*domain.User, error) {
	key := fmt.Sprintf("user:%s", userID)
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found in cache")
		}
		return nil, err
	}

	var user domain.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// CacheUserByEmail 缓存用户邮箱映射
func (c *Cache) CacheUserByEmail(email, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("user:email:%s", email)
	return c.client.Set(c.ctx, key, userID, ttl).Err()
}

// GetCachedUserByEmail 获取缓存的用户邮箱映射
func (c *Cache) GetCachedUserByEmail(email string) (string, error) {
	key := fmt.Sprintf("user:email:%s", email)
	userID, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("user not found in cache")
		}
		return "", err
	}
	return userID, nil
}

// ========== API Key 缓存 ==========

// CacheAPIKey 缓存API Key信息
func (c *Cache) CacheAPIKey(apiKey *domain.APIKey, ttl time.Duration) error {
	key := fmt.Sprintf("apikey:%s", apiKey.ID)
	data, err := json.Marshal(apiKey)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedAPIKey 获取缓存的API Key信息
func (c *Cache) GetCachedAPIKey(apiKeyID string) (*domain.APIKey, error) {
	key := fmt.Sprintf("apikey:%s", apiKeyID)
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("api key not found in cache")
		}
		return nil, err
	}

	var apiKey domain.APIKey
	if err := json.Unmarshal([]byte(data), &apiKey); err != nil {
		return nil, err
	}

	return &apiKey, nil
}

// CacheAPIKeyUser 缓存API Key到用户ID的映射
func (c *Cache) CacheAPIKeyUser(apiKey, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("apikey:user:%s", apiKey)
	return c.client.Set(c.ctx, key, userID, ttl).Err()
}

// GetCachedAPIKeyUser 获取缓存的API Key用户映射
func (c *Cache) GetCachedAPIKeyUser(apiKey string) (string, error) {
	key := fmt.Sprintf("apikey:user:%s", apiKey)
	userID, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("api key user mapping not found in cache")
		}
		return "", err
	}
	return userID, nil
}

// ========== 系统域名缓存 ==========

// CacheSystemDomain 缓存系统域名信息
func (c *Cache) CacheSystemDomain(sysDomain *domain.SystemDomain, ttl time.Duration) error {
	key := fmt.Sprintf("system_domain:%s", sysDomain.ID)
	data, err := json.Marshal(sysDomain)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedSystemDomain 获取缓存的系统域名信息
func (c *Cache) GetCachedSystemDomain(domainID string) (*domain.SystemDomain, error) {
	key := fmt.Sprintf("system_domain:%s", domainID)
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("system domain not found in cache")
		}
		return nil, err
	}

	var sysDomain domain.SystemDomain
	if err := json.Unmarshal([]byte(data), &sysDomain); err != nil {
		return nil, err
	}

	return &sysDomain, nil
}

// CacheSystemDomainList 缓存系统域名列表
func (c *Cache) CacheSystemDomainList(sysDomains []*domain.SystemDomain, ttl time.Duration) error {
	key := "system_domains:list"
	data, err := json.Marshal(sysDomains)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedSystemDomainList 获取缓存的系统域名列表
func (c *Cache) GetCachedSystemDomainList() ([]*domain.SystemDomain, error) {
	key := "system_domains:list"
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("system domain list not found in cache")
		}
		return nil, err
	}

	var sysDomains []*domain.SystemDomain
	if err := json.Unmarshal([]byte(data), &sysDomains); err != nil {
		return nil, err
	}

	return sysDomains, nil
}

// CacheDefaultSystemDomain 缓存默认系统域名
func (c *Cache) CacheDefaultSystemDomain(sysDomain *domain.SystemDomain, ttl time.Duration) error {
	key := "system_domain:default"
	data, err := json.Marshal(sysDomain)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedDefaultSystemDomain 获取缓存的默认系统域名
func (c *Cache) GetCachedDefaultSystemDomain() (*domain.SystemDomain, error) {
	key := "system_domain:default"
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("default system domain not found in cache")
		}
		return nil, err
	}

	var sysDomain domain.SystemDomain
	if err := json.Unmarshal([]byte(data), &sysDomain); err != nil {
		return nil, err
	}

	return &sysDomain, nil
}

// ========== JWT 黑名单 ==========

// AddToBlacklist 将 JWT 添加到黑名单
func (c *Cache) AddToBlacklist(jti string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", jti)
	return c.client.Set(c.ctx, key, "1", ttl).Err()
}

// IsBlacklisted 检查 JWT 是否在黑名单中
func (c *Cache) IsBlacklisted(jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", jti)
	_, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ========== 限流缓存 ==========

// IncrementRateLimit 增加限流计数
func (c *Cache) IncrementRateLimit(key string, window time.Duration) (int64, error) {
	pipe := c.client.Pipeline()

	// 增加计数
	incr := pipe.Incr(c.ctx, key)

	// 设置过期时间（如果是新键）
	pipe.Expire(c.ctx, key, window)

	_, err := pipe.Exec(c.ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// GetRateLimit 获取限流计数
func (c *Cache) GetRateLimit(key string) (int64, error) {
	count, err := c.client.Get(c.ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

// ========== 会话缓存 ==========

// CacheSession 缓存用户会话
func (c *Cache) CacheSession(sessionID string, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.client.Set(c.ctx, key, userID, ttl).Err()
}

// GetCachedSession 获取缓存的会话
func (c *Cache) GetCachedSession(sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	userID, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("session not found in cache")
		}
		return "", err
	}
	return userID, nil
}

// ========== 系统配置缓存 ==========

// CacheConfig 缓存系统配置
func (c *Cache) CacheConfig(config *domain.SystemConfig, ttl time.Duration) error {
	key := "system:config"
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedConfig 获取缓存的系统配置
func (c *Cache) GetCachedConfig() (*domain.SystemConfig, error) {
	key := "system:config"
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("config not found in cache")
		}
		return nil, err
	}

	var config domain.SystemConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// CacheStatistics 缓存系统统计信息
func (c *Cache) CacheStatistics(stats *domain.SystemStatistics, ttl time.Duration) error {
	key := "system:statistics"
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetCachedStatistics 获取缓存的系统统计信息
func (c *Cache) GetCachedStatistics() (*domain.SystemStatistics, error) {
	key := "system:statistics"
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("statistics not found in cache")
		}
		return nil, err
	}

	var stats domain.SystemStatistics
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// DeleteCachedSession 删除缓存的会话
func (c *Cache) DeleteCachedSession(sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return c.client.Del(c.ctx, key).Err()
}

// ========== 发布订阅 ==========

// PublishNewMail 发布新邮件通知
func (c *Cache) PublishNewMail(mailboxID string, message *domain.Message) error {
	channel := fmt.Sprintf("new_mail:%s", mailboxID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return c.client.Publish(c.ctx, channel, data).Err()
}

// SubscribeNewMail 订阅新邮件通知
func (c *Cache) SubscribeNewMail(mailboxID string) *redis.PubSub {
	channel := fmt.Sprintf("new_mail:%s", mailboxID)
	return c.client.Subscribe(c.ctx, channel)
}

// ========== 工具方法 ==========

// SetTTL 设置键的过期时间
func (c *Cache) SetTTL(key string, ttl time.Duration) error {
	return c.client.Expire(c.ctx, key, ttl).Err()
}

// Delete 删除键
func (c *Cache) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

// Exists 检查键是否存在
func (c *Cache) Exists(key string) (bool, error) {
	count, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// FlushAll 清空所有缓存
func (c *Cache) FlushAll() error {
	return c.client.FlushAll(c.ctx).Err()
}

// Close 关闭 Redis 连接
func (c *Cache) Close() error {
	return c.client.Close()
}

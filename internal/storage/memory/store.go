package memory

import (
	"errors"
	"strings"
	"sync"
	"time"

	"tempmail/backend/internal/domain"
)

var (
	ErrMailboxNotFound = errors.New("mailbox not found")
	ErrMessageNotFound = errors.New("message not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrEmailExists     = errors.New("email already exists")
)

// Store 使用内存保存邮箱与邮件数据，主要用于开发验证。
type Store struct {
	mu             sync.RWMutex
	mailboxes      map[string]*domain.Mailbox
	byAddress      map[string]string
	messages       map[string]map[string]*domain.Message // mailboxID -> messageID -> message
	users          map[string]*domain.User               // userID -> user
	byEmail        map[string]string                     // email -> userID
	byUsername     map[string]string                     // username -> userID
	apiKeys        map[string]*domain.APIKey             // apiKeyID -> apiKey
	byAPIKey       map[string]string                     // key -> userID
	aliases        map[string]*domain.MailboxAlias       // aliasID -> alias
	byAlias        map[string]string                     // address -> aliasID
	userDomains    map[string]*domain.UserDomain         // domainID -> userDomain
	byDomain       map[string]string                     // domain -> domainID
	systemDomains  map[string]*domain.SystemDomain       // domainID -> systemDomain
	bySystemDomain map[string]string                     // domain -> domainID

	// Webhook 存储
	webhooks       map[string]*domain.Webhook            // 按 ID 索引
	webhooksByUser map[string]map[string]*domain.Webhook // 按用户 ID 索引
	deliveries     map[string][]*domain.WebhookDelivery  // 投递记录（按 webhook ID）
	retryQueue     []*domain.WebhookDelivery             // 重试队列

	// 标签存储
	tags          map[string]*domain.Tag                   // 按 ID 索引
	tagsByUser    map[string]map[string]*domain.Tag        // 按用户 ID 索引
	messageTags   map[string]*domain.MessageTag            // 按 "messageID:tagID" 索引
	tagsByMessage map[string]map[string]*domain.MessageTag // 按邮件 ID 索引

	// 系统配置
	systemConfig *domain.SystemConfig

	// 速率限制相关
	rateLimits        map[string]*rateLimitEntry
	rateLimitsCleanup time.Time // 下次清理过期速率限制的时间

	ttl time.Duration
}

// rateLimitEntry 速率限制条目
type rateLimitEntry struct {
	Count     int64
	ExpiresAt time.Time
}

// NewStore 创建一个内存存储实例。
func NewStore(ttl time.Duration) *Store {
	return &Store{
		mailboxes:         make(map[string]*domain.Mailbox),
		byAddress:         make(map[string]string),
		messages:          make(map[string]map[string]*domain.Message),
		users:             make(map[string]*domain.User),
		byEmail:           make(map[string]string),
		byUsername:        make(map[string]string),
		apiKeys:           make(map[string]*domain.APIKey),
		byAPIKey:          make(map[string]string),
		aliases:           make(map[string]*domain.MailboxAlias),
		byAlias:           make(map[string]string),
		userDomains:       make(map[string]*domain.UserDomain),
		byDomain:          make(map[string]string),
		systemDomains:     make(map[string]*domain.SystemDomain),
		bySystemDomain:    make(map[string]string),
		webhooks:          make(map[string]*domain.Webhook),
		webhooksByUser:    make(map[string]map[string]*domain.Webhook),
		deliveries:        make(map[string][]*domain.WebhookDelivery),
		retryQueue:        make([]*domain.WebhookDelivery, 0),
		tags:              make(map[string]*domain.Tag),
		tagsByUser:        make(map[string]map[string]*domain.Tag),
		messageTags:       make(map[string]*domain.MessageTag),
		tagsByMessage:     make(map[string]map[string]*domain.MessageTag),
		systemConfig:      domain.DefaultSystemConfig(),
		rateLimits:        make(map[string]*rateLimitEntry),
		rateLimitsCleanup: time.Now().Add(5 * time.Minute),
		ttl:               ttl,
	}
}

// SaveMailbox 保存邮箱信息。
func (s *Store) SaveMailbox(mailbox *domain.Mailbox) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked()

	s.mailboxes[mailbox.ID] = mailbox
	s.byAddress[mailbox.Address] = mailbox.ID
	return nil
}

// GetMailbox 根据 ID 获取邮箱。
func (s *Store) GetMailbox(id string) (*domain.Mailbox, error) {
	s.mu.RLock()
	mailbox, ok := s.mailboxes[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrMailboxNotFound
	}
	if mailboxExpired(mailbox, s.ttl) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.deleteMailboxLocked(id)
		return nil, ErrMailboxNotFound
	}
	return mailbox, nil
}

// GetMailboxByAddress 根据完整地址获取邮箱。
func (s *Store) GetMailboxByAddress(address string) (*domain.Mailbox, error) {
	s.mu.RLock()
	id, ok := s.byAddress[address]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrMailboxNotFound
	}
	return s.GetMailbox(id)
}

// ListMailboxes 返回全部邮箱的快照。
func (s *Store) ListMailboxes() []domain.Mailbox {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked()

	result := make([]domain.Mailbox, 0, len(s.mailboxes))
	for _, mb := range s.mailboxes {
		if mailboxExpired(mb, s.ttl) {
			continue
		}
		result = append(result, *mb)
	}
	return result
}

// DeleteMailbox 删除指定邮箱。
func (s *Store) DeleteMailbox(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.mailboxes[id]; !ok {
		return ErrMailboxNotFound
	}
	s.deleteMailboxLocked(id)
	return nil
}

// ListMailboxesByUserID 返回指定用户的全部邮箱。
func (s *Store) ListMailboxesByUserID(userID string) []domain.Mailbox {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked()

	result := make([]domain.Mailbox, 0)
	for _, mb := range s.mailboxes {
		if mailboxExpired(mb, s.ttl) {
			continue
		}
		// 检查是否属于该用户
		if mb.UserID != nil && *mb.UserID == userID {
			result = append(result, *mb)
		}
	}
	return result
}

// DeleteExpiredMailboxes 删除所有过期的邮箱，返回删除数量。
func (s *Store) DeleteExpiredMailboxes() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	now := time.Now()
	for id, mb := range s.mailboxes {
		if mailboxExpiredAt(mb, now, s.ttl) {
			s.deleteMailboxLocked(id)
			count++
		}
	}
	return count, nil
}

func (s *Store) deleteMailboxLocked(id string) {
	if mb, ok := s.mailboxes[id]; ok {
		delete(s.byAddress, mb.Address)
	}
	delete(s.mailboxes, id)
	delete(s.messages, id)
}

// SaveMessage 保存邮件信息。
func (s *Store) SaveMessage(message *domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked()

	if _, ok := s.mailboxes[message.MailboxID]; !ok {
		return ErrMailboxNotFound
	}

	if _, ok := s.messages[message.MailboxID]; !ok {
		s.messages[message.MailboxID] = make(map[string]*domain.Message)
	}
	s.messages[message.MailboxID][message.ID] = message

	mb := s.mailboxes[message.MailboxID]
	mb.TotalCount++
	if !message.IsRead {
		mb.Unread++
	}

	return nil
}

// ListMessages 返回某个邮箱下的全部邮件。
func (s *Store) ListMessages(mailboxID string) ([]domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked()

	if _, ok := s.mailboxes[mailboxID]; !ok {
		return nil, ErrMailboxNotFound
	}

	msgMap, ok := s.messages[mailboxID]
	if !ok {
		return []domain.Message{}, nil
	}

	result := make([]domain.Message, 0, len(msgMap))
	for _, msg := range msgMap {
		result = append(result, *msg)
	}

	return result, nil
}

// GetMessage 获取单封邮件。
func (s *Store) GetMessage(mailboxID, messageID string) (*domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredLocked()

	msgMap, ok := s.messages[mailboxID]
	if !ok {
		return nil, ErrMessageNotFound
	}

	msg, ok := msgMap[messageID]
	if !ok {
		return nil, ErrMessageNotFound
	}

	return msg, nil
}

// MarkMessageRead 将邮件标记为已读。
func (s *Store) MarkMessageRead(mailboxID, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgMap, ok := s.messages[mailboxID]
	if !ok {
		return ErrMessageNotFound
	}

	msg, ok := msgMap[messageID]
	if !ok {
		return ErrMessageNotFound
	}

	if !msg.IsRead {
		msg.IsRead = true
		if mb, ok := s.mailboxes[mailboxID]; ok && mb.Unread > 0 {
			mb.Unread--
		}
	}

	return nil
}

// DeleteMessage 删除指定邮件。
func (s *Store) DeleteMessage(mailboxID, messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgMap, ok := s.messages[mailboxID]
	if !ok {
		return ErrMessageNotFound
	}

	msg, ok := msgMap[messageID]
	if !ok {
		return ErrMessageNotFound
	}

	// 更新邮箱统计
	if mb, ok := s.mailboxes[mailboxID]; ok {
		mb.TotalCount--
		if !msg.IsRead && mb.Unread > 0 {
			mb.Unread--
		}
	}

	// 删除消息
	delete(msgMap, messageID)

	return nil
}

// DeleteAllMessages 删除邮箱中的所有消息，返回删除数量。
func (s *Store) DeleteAllMessages(mailboxID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证邮箱是否存在
	if _, ok := s.mailboxes[mailboxID]; !ok {
		return 0, ErrMailboxNotFound
	}

	msgMap, ok := s.messages[mailboxID]
	if !ok {
		return 0, nil // 没有消息，返回0
	}

	count := len(msgMap)

	// 重置邮箱统计
	if mb, ok := s.mailboxes[mailboxID]; ok {
		mb.TotalCount = 0
		mb.Unread = 0
	}

	// 删除所有消息
	delete(s.messages, mailboxID)

	return count, nil
}

// pruneExpiredLocked 清理过期邮箱。
func (s *Store) pruneExpiredLocked() {
	now := time.Now()
	for id, mb := range s.mailboxes {
		if mailboxExpiredAt(mb, now, s.ttl) {
			s.deleteMailboxLocked(id)
		}
	}
}

// mailboxExpired 判断邮箱是否已过期。
func mailboxExpired(mailbox *domain.Mailbox, ttl time.Duration) bool {
	return mailboxExpiredAt(mailbox, time.Now(), ttl)
}

// mailboxExpiredAt 在指定时间判断邮箱是否过期。
func mailboxExpiredAt(mailbox *domain.Mailbox, now time.Time, ttl time.Duration) bool {
	if mailbox.ExpiresAt != nil {
		return now.After(*mailbox.ExpiresAt)
	}
	if ttl <= 0 {
		return false
	}
	return now.After(mailbox.CreatedAt.Add(ttl))
}

// ========== User Repository ==========

// CreateUser 创建新用户
func (s *Store) CreateUser(user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查邮箱是否已存在
	if _, exists := s.byEmail[user.Email]; exists {
		return ErrEmailExists
	}

	// 检查用户名是否已存在（用户名不区分大小写）
	if _, exists := s.byUsername[strings.ToLower(user.Username)]; exists {
		return ErrEmailExists
	}

	// 检查ID是否存在
	if user.ID == "" {
		return errors.New("user ID is required")
	}

	// 如果时间戳为零值，则设置为当前时间
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	s.users[user.ID] = user
	s.byEmail[user.Email] = user.ID
	s.byUsername[strings.ToLower(user.Username)] = user.ID

	return nil
}

// GetUserByID 根据ID获取用户
func (s *Store) GetUserByID(id string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *Store) GetUserByEmail(email string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.byEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}

	user, ok := s.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *Store) GetUserByUsername(username string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.byUsername[strings.ToLower(username)]
	if !ok {
		return nil, ErrUserNotFound
	}

	user, ok := s.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// UpdateUser 更新用户信息
func (s *Store) UpdateUser(user *domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[user.ID]; !ok {
		return ErrUserNotFound
	}

	user.UpdatedAt = time.Now().UTC()
	oldUsername := ""
	for username, id := range s.byUsername {
		if id == user.ID {
			oldUsername = username
			break
		}
	}

	newUsername := strings.ToLower(user.Username)
	if oldUsername != "" && oldUsername != newUsername {
		delete(s.byUsername, oldUsername)
		if _, exists := s.byUsername[newUsername]; exists {
			return ErrEmailExists
		}
	}

	s.users[user.ID] = user
	s.byUsername[newUsername] = user.ID

	return nil
}

// UpdateLastLogin 更新用户最后登录时间
func (s *Store) UpdateLastLogin(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return ErrUserNotFound
	}

	now := time.Now().UTC()
	user.LastLoginAt = &now
	user.UpdatedAt = now

	return nil
}

// GetUserByAPIKey 根据API Key获取用户
func (s *Store) GetUserByAPIKey(apiKey string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, ok := s.byAPIKey[apiKey]
	if !ok {
		return nil, ErrUserNotFound
	}

	user, ok := s.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// SaveAPIKey 保存API Key
func (s *Store) SaveAPIKey(apiKey *domain.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiKeys[apiKey.ID] = apiKey
	s.byAPIKey[apiKey.Key] = apiKey.UserID

	return nil
}

// GetAPIKey 根据ID获取API Key
func (s *Store) GetAPIKey(id string) (*domain.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, ok := s.apiKeys[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return apiKey, nil
}

// GetAPIKeyByKey 根据Key字符串获取API Key
func (s *Store) GetAPIKeyByKey(key string) (*domain.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 首先通过key找到userID
	userID, ok := s.byAPIKey[key]
	if !ok {
		return nil, ErrUserNotFound
	}

	// 然后遍历找到对应的API Key对象
	for _, apiKey := range s.apiKeys {
		if apiKey.Key == key && apiKey.UserID == userID {
			return apiKey, nil
		}
	}

	return nil, ErrUserNotFound
}

// ListAPIKeysByUserID 列出用户的所有API Key
func (s *Store) ListAPIKeysByUserID(userID string) ([]*domain.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]*domain.APIKey, 0)
	for _, apiKey := range s.apiKeys {
		if apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}

	return keys, nil
}

// DeleteAPIKey 删除API Key
func (s *Store) DeleteAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKey, ok := s.apiKeys[id]
	if !ok {
		return ErrUserNotFound
	}

	// 删除key索引
	delete(s.byAPIKey, apiKey.Key)
	// 删除API Key
	delete(s.apiKeys, id)

	return nil
}

// UpdateAPIKeyLastUsed 更新API Key最后使用时间
func (s *Store) UpdateAPIKeyLastUsed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKey, ok := s.apiKeys[id]
	if !ok {
		return ErrUserNotFound
	}

	now := time.Now()
	apiKey.LastUsedAt = &now

	return nil
}

// ========== Admin Repository ==========

// ListUsers 列出用户（支持分页和过滤）
func (s *Store) ListUsers(page, pageSize int, search string, role *domain.UserRole, tier *domain.UserTier, isActive *bool) ([]domain.User, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 收集所有符合条件的用户
	filtered := make([]domain.User, 0)
	for _, user := range s.users {
		// 搜索过滤
		if search != "" {
			if !containsIgnoreCase(user.Email, search) && !containsIgnoreCase(user.Username, search) {
				continue
			}
		}

		// 角色过滤
		if role != nil && user.Role != *role {
			continue
		}

		// 等级过滤
		if tier != nil && user.Tier != *tier {
			continue
		}

		// 激活状态过滤
		if isActive != nil && user.IsActive != *isActive {
			continue
		}

		filtered = append(filtered, *user)
	}

	total := len(filtered)

	// 分页处理
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	return filtered[start:end], total, nil
}

// DeleteUser 删除用户
func (s *Store) DeleteUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return ErrUserNotFound
	}

	// 删除用户
	delete(s.users, userID)
	delete(s.byEmail, user.Email)

	return nil
}

// DeleteMailboxesByUserID 删除用户的所有邮箱
func (s *Store) DeleteMailboxesByUserID(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	toDelete := make([]string, 0)
	for id, mb := range s.mailboxes {
		if mb.UserID != nil && *mb.UserID == userID {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		s.deleteMailboxLocked(id)
	}

	return nil
}

// GetSystemStatistics 获取系统统计信息
func (s *Store) GetSystemStatistics() (*domain.SystemStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &domain.SystemStatistics{
		TotalUsers:        len(s.users),
		ActiveUsers:       0,
		TotalMailboxes:    len(s.mailboxes),
		ActiveMailboxes:   0,
		TotalMessages:     0,
		MessagesToday:     0,
		UsersByTier:       make(map[domain.UserTier]int),
		UsersByRole:       make(map[domain.UserRole]int),
		MailboxesByDomain: make(map[string]int),
		RecentActivity:    make([]domain.ActivityLog, 0),
	}

	// 统计用户信息
	for _, user := range s.users {
		if user.IsActive {
			stats.ActiveUsers++
		}
		stats.UsersByTier[user.Tier]++
		stats.UsersByRole[user.Role]++
	}

	// 统计邮箱信息
	now := time.Now()
	for _, mb := range s.mailboxes {
		if !mailboxExpiredAt(mb, now, s.ttl) {
			stats.ActiveMailboxes++
			stats.MailboxesByDomain[mb.Domain]++
		}
	}

	// 统计邮件信息
	today := time.Now().Truncate(24 * time.Hour)
	for _, msgMap := range s.messages {
		for _, msg := range msgMap {
			stats.TotalMessages++
			if msg.CreatedAt.After(today) {
				stats.MessagesToday++
			}
		}
	}

	return stats, nil
}

// GetDomainStatistics 获取域名统计信息
func (s *Store) GetDomainStatistics(domain string) (mailboxCount, messageCount int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, mb := range s.mailboxes {
		if mb.Domain == domain {
			mailboxCount++
		}
	}

	for mbID, msgMap := range s.messages {
		if mb, ok := s.mailboxes[mbID]; ok && mb.Domain == domain {
			messageCount += len(msgMap)
		}
	}

	return mailboxCount, messageCount, nil
}

// containsIgnoreCase 不区分大小写的字符串包含检查
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// ========== System Config Repository ==========

// GetSystemConfig 获取系统配置
func (s *Store) GetSystemConfig() (*domain.SystemConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.systemConfig == nil {
		return domain.DefaultSystemConfig(), nil
	}

	// 返回配置的副本
	config := *s.systemConfig
	return &config, nil
}

// SaveSystemConfig 保存系统配置
func (s *Store) SaveSystemConfig(config *domain.SystemConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.UpdatedAt = time.Now()
	s.systemConfig = config
	return nil
}

// ========== Alias Repository ==========

// SaveAlias 保存邮箱别名
func (s *Store) SaveAlias(alias *domain.MailboxAlias) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查邮箱是否存在
	if _, ok := s.mailboxes[alias.MailboxID]; !ok {
		return ErrMailboxNotFound
	}

	// 检查别名地址是否已被使用（作为主地址或其他别名）
	if _, ok := s.byAddress[alias.Address]; ok {
		return errors.New("address already exists as mailbox")
	}
	// 检查地址是否被其他别名使用（更新同一别名时允许）
	if existingID, ok := s.byAlias[alias.Address]; ok && existingID != alias.ID {
		return errors.New("alias address already exists")
	}

	s.aliases[alias.ID] = alias
	s.byAlias[alias.Address] = alias.ID

	return nil
}

// GetAlias 根据ID获取别名
func (s *Store) GetAlias(aliasID string) (*domain.MailboxAlias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alias, ok := s.aliases[aliasID]
	if !ok {
		return nil, errors.New("alias not found")
	}

	return alias, nil
}

// GetAliasByAddress 根据地址获取别名
func (s *Store) GetAliasByAddress(address string) (*domain.MailboxAlias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aliasID, ok := s.byAlias[address]
	if !ok {
		return nil, errors.New("alias not found")
	}

	alias, ok := s.aliases[aliasID]
	if !ok {
		return nil, errors.New("alias not found")
	}

	return alias, nil
}

// ListAliasesByMailboxID 列出指定邮箱的所有别名
func (s *Store) ListAliasesByMailboxID(mailboxID string) ([]*domain.MailboxAlias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.MailboxAlias, 0)
	for _, alias := range s.aliases {
		if alias.MailboxID == mailboxID {
			result = append(result, alias)
		}
	}

	return result, nil
}

// DeleteAlias 删除别名
func (s *Store) DeleteAlias(aliasID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alias, ok := s.aliases[aliasID]
	if !ok {
		return errors.New("alias not found")
	}

	delete(s.aliases, aliasID)
	delete(s.byAlias, alias.Address)

	return nil
}

// ========== JWT 黑名单 ==========

// AddToBlacklist 将 JWT 添加到黑名单
func (s *Store) AddToBlacklist(jti string, ttl time.Duration) error {
	// 内存存储不支持 JWT 黑名单，返回错误
	return errors.New("JWT blacklist not supported in memory storage")
}

// IsBlacklisted 检查 JWT 是否在黑名单中
func (s *Store) IsBlacklisted(jti string) (bool, error) {
	// 内存存储不支持 JWT 黑名单，总是返回 false
	return false, nil
}

// ========== 限流 ==========

// IncrementRateLimit 增加限流计数
func (s *Store) IncrementRateLimit(key string, window time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// 清理过期的速率限制条目（每5分钟清理一次）
	if now.After(s.rateLimitsCleanup) {
		for k, v := range s.rateLimits {
			if now.After(v.ExpiresAt) {
				delete(s.rateLimits, k)
			}
		}
		s.rateLimitsCleanup = now.Add(5 * time.Minute)
	}

	// 获取或创建速率限制条目
	entry, exists := s.rateLimits[key]
	if !exists || now.After(entry.ExpiresAt) {
		// 创建新条目
		entry = &rateLimitEntry{
			Count:     1,
			ExpiresAt: now.Add(window),
		}
		s.rateLimits[key] = entry
		return 1, nil
	}

	// 增加计数
	entry.Count++
	return entry.Count, nil
}

// GetRateLimit 获取限流计数
func (s *Store) GetRateLimit(key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.rateLimits[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return 0, nil
	}

	return entry.Count, nil
}

// ========== 会话管理 ==========

// CacheSession 缓存用户会话
func (s *Store) CacheSession(sessionID string, userID string, ttl time.Duration) error {
	// 内存存储不支持会话缓存，返回错误
	return errors.New("session caching not supported in memory storage")
}

// GetCachedSession 获取缓存的会话
func (s *Store) GetCachedSession(sessionID string) (string, error) {
	// 内存存储不支持会话缓存，返回错误
	return "", errors.New("session caching not supported in memory storage")
}

// DeleteCachedSession 删除缓存的会话
func (s *Store) DeleteCachedSession(sessionID string) error {
	// 内存存储不支持会话缓存，返回错误
	return errors.New("session caching not supported in memory storage")
}

// ========== 发布订阅 ==========

// PublishNewMail 发布新邮件通知
func (s *Store) PublishNewMail(mailboxID string, message *domain.Message) error {
	// 内存存储不支持发布订阅，返回错误
	return errors.New("pub/sub not supported in memory storage")
}

// SubscribeNewMail 订阅新邮件通知
func (s *Store) SubscribeNewMail(mailboxID string) interface{} {
	// 内存存储不支持发布订阅，返回 nil
	return nil
}

// ========== 工具方法 ==========

// Close 关闭存储连接
func (s *Store) Close() error {
	// 内存存储不需要关闭连接
	return nil
}

// Health 健康检查
func (s *Store) Health() error {
	// 内存存储总是健康的
	return nil
}

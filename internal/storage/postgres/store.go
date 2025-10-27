package postgres

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tempmail/backend/internal/domain"
)

var (
	ErrMailboxNotFound = fmt.Errorf("mailbox not found")
	ErrMessageNotFound = fmt.Errorf("message not found")
	ErrUserNotFound    = fmt.Errorf("user not found")
	ErrEmailExists     = fmt.Errorf("email already exists")
	ErrAliasNotFound   = fmt.Errorf("alias not found")
	ErrAliasExists     = fmt.Errorf("alias already exists")
)

// Store PostgreSQL 存储实现
type Store struct {
	db *gorm.DB
}

// NewStore 创建 PostgreSQL 存储实例
func NewStore(dsn string) (*Store, error) {
	return NewStoreWithDialector(postgres.Open(dsn))
}

// NewMySQLStore 创建 MySQL 存储实例
func NewMySQLStore(dsn string) (*Store, error) {
	return NewStoreWithDialector(mysql.Open(dsn))
}

// NewStoreWithDialector 使用指定的GORM dialector创建存储实例
func NewStoreWithDialector(dialector gorm.Dialector) (*Store, error) {
	// 配置 GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 静默模式
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// 连接数据库
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	store := &Store{db: db}

	// 自动迁移数据库表
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

// migrate 自动迁移数据库表结构
func (s *Store) migrate() error {
	return s.db.AutoMigrate(
		&domain.User{},
		&domain.Mailbox{},
		&domain.Message{},
		&domain.MailboxAlias{},
		&domain.SystemDomain{},
		&domain.UserDomain{},
		&domain.APIKey{},
		&domain.Attachment{},
		&domain.Webhook{},
		&domain.WebhookDelivery{},
		&domain.Tag{},
		&domain.MessageTag{},
	)
}

// ========== Mailbox Repository ==========

// SaveMailbox 保存邮箱信息
func (s *Store) SaveMailbox(mailbox *domain.Mailbox) error {
	return s.db.Save(mailbox).Error
}

// GetMailbox 根据 ID 获取邮箱
func (s *Store) GetMailbox(id string) (*domain.Mailbox, error) {
	var mailbox domain.Mailbox
	err := s.db.Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).First(&mailbox).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMailboxNotFound
		}
		return nil, err
	}
	return &mailbox, nil
}

// GetMailboxByAddress 根据完整地址获取邮箱
func (s *Store) GetMailboxByAddress(address string) (*domain.Mailbox, error) {
	var mailbox domain.Mailbox
	err := s.db.Where("address = ? AND (expires_at IS NULL OR expires_at > ?)", address, time.Now()).First(&mailbox).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMailboxNotFound
		}
		return nil, err
	}
	return &mailbox, nil
}

// ListMailboxes 返回全部邮箱的快照
func (s *Store) ListMailboxes() []domain.Mailbox {
	var mailboxes []domain.Mailbox
	s.db.Where("expires_at IS NULL OR expires_at > ?", time.Now()).Find(&mailboxes)
	return mailboxes
}

// ListMailboxesByUserID 返回指定用户的全部邮箱
func (s *Store) ListMailboxesByUserID(userID string) []domain.Mailbox {
	var mailboxes []domain.Mailbox
	s.db.Where("user_id = ? AND (expires_at IS NULL OR expires_at > ?)", userID, time.Now()).Find(&mailboxes)
	return mailboxes
}

// DeleteMailbox 删除指定邮箱
func (s *Store) DeleteMailbox(id string) error {
	// 使用事务删除邮箱及其相关数据
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除邮箱的邮件
		if err := tx.Where("mailbox_id = ?", id).Delete(&domain.Message{}).Error; err != nil {
			return err
		}

		// 删除邮箱的别名
		if err := tx.Where("mailbox_id = ?", id).Delete(&domain.MailboxAlias{}).Error; err != nil {
			return err
		}

		// 删除邮箱
		return tx.Where("id = ?", id).Delete(&domain.Mailbox{}).Error
	})
}

// DeleteExpiredMailboxes 删除所有过期的邮箱，返回删除数量
func (s *Store) DeleteExpiredMailboxes() (int, error) {
	var count int64

	// 使用事务删除过期邮箱
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 查找过期的邮箱
		var expiredMailboxes []domain.Mailbox
		if err := tx.Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).Find(&expiredMailboxes).Error; err != nil {
			return err
		}

		count = int64(len(expiredMailboxes))
		if count == 0 {
			return nil
		}

		// 删除过期邮箱的邮件
		for _, mb := range expiredMailboxes {
			if err := tx.Where("mailbox_id = ?", mb.ID).Delete(&domain.Message{}).Error; err != nil {
				return err
			}
		}

		// 删除过期邮箱的别名
		for _, mb := range expiredMailboxes {
			if err := tx.Where("mailbox_id = ?", mb.ID).Delete(&domain.MailboxAlias{}).Error; err != nil {
				return err
			}
		}

		// 删除过期邮箱
		return tx.Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).Delete(&domain.Mailbox{}).Error
	})

	return int(count), err
}

// ========== Message Repository ==========

// SaveMessage 保存邮件信息
func (s *Store) SaveMessage(message *domain.Message) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 保存邮件
		if err := tx.Save(message).Error; err != nil {
			return err
		}

		// 更新邮箱统计
		var mailbox domain.Mailbox
		if err := tx.Where("id = ?", message.MailboxID).First(&mailbox).Error; err != nil {
			return err
		}

		mailbox.TotalCount++
		if !message.IsRead {
			mailbox.Unread++
		}

		return tx.Save(&mailbox).Error
	})
}

// ListMessages 返回某个邮箱下的全部邮件
func (s *Store) ListMessages(mailboxID string) ([]domain.Message, error) {
	var messages []domain.Message
	err := s.db.Where("mailbox_id = ?", mailboxID).Order("created_at DESC").Find(&messages).Error
	return messages, err
}

// GetMessage 获取单封邮件
func (s *Store) GetMessage(mailboxID, messageID string) (*domain.Message, error) {
	var message domain.Message
	err := s.db.Where("id = ? AND mailbox_id = ?", messageID, mailboxID).First(&message).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

// MarkMessageRead 将邮件标记为已读
func (s *Store) MarkMessageRead(mailboxID, messageID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新邮件状态
		result := tx.Model(&domain.Message{}).
			Where("id = ? AND mailbox_id = ? AND is_read = ?", messageID, mailboxID, false).
			Update("is_read", true)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return ErrMessageNotFound
		}

		// 更新邮箱未读数
		return tx.Model(&domain.Mailbox{}).
			Where("id = ?", mailboxID).
			UpdateColumn("unread", gorm.Expr("unread - 1")).
			Error
	})
}

// GetAttachment 获取邮件附件
func (s *Store) GetAttachment(mailboxID, messageID, attachmentID string) (*domain.Attachment, error) {
	var attachment domain.Attachment
	err := s.db.Where("id = ? AND message_id = ? AND mailbox_id = ?", attachmentID, messageID, mailboxID).First(&attachment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("attachment not found")
		}
		return nil, err
	}
	return &attachment, nil
}

// SearchMessages 已在 search_webhook.go 中实现

// ========== API Key Repository ==========

// SaveAPIKey 保存API Key
func (s *Store) SaveAPIKey(apiKey *domain.APIKey) error {
	if apiKey.ID == "" {
		apiKey.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	apiKey.CreatedAt = now

	return s.db.Create(apiKey).Error
}

// GetAPIKey 根据ID获取API Key
func (s *Store) GetAPIKey(id string) (*domain.APIKey, error) {
	var apiKey domain.APIKey
	err := s.db.Where("id = ?", id).First(&apiKey).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &apiKey, nil
}

// GetAPIKeyByKey 根据Key字符串获取API Key
func (s *Store) GetAPIKeyByKey(key string) (*domain.APIKey, error) {
	var apiKey domain.APIKey
	err := s.db.Where("key_hash = ?", key).First(&apiKey).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &apiKey, nil
}

// ListAPIKeysByUserID 列出用户的所有API Key
func (s *Store) ListAPIKeysByUserID(userID string) ([]*domain.APIKey, error) {
	var apiKeys []*domain.APIKey
	err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&apiKeys).Error
	return apiKeys, err
}

// DeleteAPIKey 删除API Key
func (s *Store) DeleteAPIKey(id string) error {
	result := s.db.Where("id = ?", id).Delete(&domain.APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateAPIKeyLastUsed 更新API Key最后使用时间
func (s *Store) UpdateAPIKeyLastUsed(id string) error {
	now := time.Now().UTC()
	result := s.db.Model(&domain.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// GetUserByAPIKey 根据API Key获取用户
func (s *Store) GetUserByAPIKey(apiKey string) (*domain.User, error) {
	var user domain.User
	err := s.db.Table("users").
		Joins("JOIN api_keys ON users.id = api_keys.user_id").
		Where("api_keys.key_hash = ? AND api_keys.is_active = ?", apiKey, true).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// ========== System Domain Repository ==========

// SaveSystemDomain 保存系统域名
func (s *Store) SaveSystemDomain(sysDomain *domain.SystemDomain) error {
	if sysDomain.ID == "" {
		sysDomain.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	sysDomain.CreatedAt = now

	return s.db.Create(sysDomain).Error
}

// GetSystemDomain 根据ID获取系统域名
func (s *Store) GetSystemDomain(domainID string) (*domain.SystemDomain, error) {
	var sysDomain domain.SystemDomain
	err := s.db.Where("id = ?", domainID).First(&sysDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("system domain not found")
		}
		return nil, err
	}
	return &sysDomain, nil
}

// GetSystemDomainByDomain 根据域名获取系统域名
func (s *Store) GetSystemDomainByDomain(domainName string) (*domain.SystemDomain, error) {
	var sysDomain domain.SystemDomain
	err := s.db.Where("domain = ?", domainName).First(&sysDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("system domain not found")
		}
		return nil, err
	}
	return &sysDomain, nil
}

// ListSystemDomains 获取所有系统域名
func (s *Store) ListSystemDomains() ([]*domain.SystemDomain, error) {
	var sysDomains []*domain.SystemDomain
	err := s.db.Order("created_at DESC").Find(&sysDomains).Error
	return sysDomains, err
}

// UpdateSystemDomain 更新系统域名
func (s *Store) UpdateSystemDomain(sysDomain *domain.SystemDomain) error {
	return s.db.Save(sysDomain).Error
}

// DeleteSystemDomain 删除系统域名
func (s *Store) DeleteSystemDomain(domainID string) error {
	result := s.db.Where("id = ?", domainID).Delete(&domain.SystemDomain{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("system domain not found")
	}
	return nil
}

// SetDefaultSystemDomain 设置默认系统域名
func (s *Store) SetDefaultSystemDomain(domainID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 先取消所有默认域名
		if err := tx.Model(&domain.SystemDomain{}).
			Where("is_default = ?", true).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// 设置新的默认域名
		result := tx.Model(&domain.SystemDomain{}).
			Where("id = ?", domainID).
			Update("is_default", true)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("system domain not found")
		}

		return nil
	})
}

// GetDefaultSystemDomain 获取默认系统域名
func (s *Store) GetDefaultSystemDomain() (*domain.SystemDomain, error) {
	var sysDomain domain.SystemDomain
	err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&sysDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no default system domain found")
		}
		return nil, err
	}
	return &sysDomain, nil
}

// IncrementSystemDomainMailboxCount 增加系统域名邮箱计数
func (s *Store) IncrementSystemDomainMailboxCount(domainName string) error {
	return s.db.Model(&domain.SystemDomain{}).
		Where("domain = ?", domainName).
		UpdateColumn("mailbox_count", gorm.Expr("mailbox_count + 1")).
		Error
}

// DecrementSystemDomainMailboxCount 减少系统域名邮箱计数
func (s *Store) DecrementSystemDomainMailboxCount(domainName string) error {
	return s.db.Model(&domain.SystemDomain{}).
		Where("domain = ? AND mailbox_count > 0", domainName).
		UpdateColumn("mailbox_count", gorm.Expr("mailbox_count - 1")).
		Error
}

// ListActiveSystemDomains 获取所有已激活的系统域名
func (s *Store) ListActiveSystemDomains() ([]*domain.SystemDomain, error) {
	var sysDomains []*domain.SystemDomain
	err := s.db.Where("is_active = ?", true).Order("created_at DESC").Find(&sysDomains).Error
	return sysDomains, err
}

// DeleteUnverifiedSystemDomains 删除指定时间前创建且未验证的域名
func (s *Store) DeleteUnverifiedSystemDomains(before time.Time) (int, error) {
	result := s.db.Where("status != ? AND created_at < ?", domain.SystemDomainStatusVerified, before).Delete(&domain.SystemDomain{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// ========== User Repository ==========

// CreateUser 创建新用户
func (s *Store) CreateUser(user *domain.User) error {
	// 检查邮箱是否已存在
	var existingUser domain.User
	err := s.db.Where("email = ?", user.Email).First(&existingUser).Error
	if err == nil {
		return ErrEmailExists
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	// 生成ID
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	return s.db.Create(user).Error
}

// GetUserByID 根据ID获取用户
func (s *Store) GetUserByID(id string) (*domain.User, error) {
	var user domain.User
	err := s.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *Store) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *Store) GetUserByUsername(username string) (*domain.User, error) {
	var user domain.User
	err := s.db.Where("lower(username) = ?", strings.ToLower(username)).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func (s *Store) UpdateUser(user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()
	return s.db.Save(user).Error
}

// UpdateLastLogin 更新用户最后登录时间
func (s *Store) UpdateLastLogin(userID string) error {
	now := time.Now().UTC()
	return s.db.Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
}

// ========== Admin Repository ==========

// ListUsers 列出用户（支持分页和过滤）
func (s *Store) ListUsers(page, pageSize int, search string, role *domain.UserRole, tier *domain.UserTier, isActive *bool) ([]domain.User, int, error) {
	query := s.db.Model(&domain.User{})

	// 搜索过滤
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(email) LIKE ? OR LOWER(username) LIKE ?", searchPattern, searchPattern)
	}

	// 角色过滤
	if role != nil {
		query = query.Where("role = ?", *role)
	}

	// 等级过滤
	if tier != nil {
		query = query.Where("tier = ?", *tier)
	}

	// 激活状态过滤
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var users []domain.User
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error

	return users, int(total), err
}

// DeleteUser 删除用户
func (s *Store) DeleteUser(userID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除用户的所有邮箱
		if err := s.DeleteMailboxesByUserID(userID); err != nil {
			return err
		}

		// 删除用户
		return tx.Where("id = ?", userID).Delete(&domain.User{}).Error
	})
}

// DeleteMailboxesByUserID 删除用户的所有邮箱
func (s *Store) DeleteMailboxesByUserID(userID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 查找用户的所有邮箱
		var mailboxes []domain.Mailbox
		if err := tx.Where("user_id = ?", userID).Find(&mailboxes).Error; err != nil {
			return err
		}

		// 删除每个邮箱的相关数据
		for _, mb := range mailboxes {
			// 删除邮件
			if err := tx.Where("mailbox_id = ?", mb.ID).Delete(&domain.Message{}).Error; err != nil {
				return err
			}

			// 删除别名
			if err := tx.Where("mailbox_id = ?", mb.ID).Delete(&domain.MailboxAlias{}).Error; err != nil {
				return err
			}
		}

		// 删除邮箱
		return tx.Where("user_id = ?", userID).Delete(&domain.Mailbox{}).Error
	})
}

// GetSystemStatistics 获取系统统计信息
func (s *Store) GetSystemStatistics() (*domain.SystemStatistics, error) {
	stats := &domain.SystemStatistics{
		UsersByTier:       make(map[domain.UserTier]int),
		UsersByRole:       make(map[domain.UserRole]int),
		MailboxesByDomain: make(map[string]int),
		RecentActivity:    make([]domain.ActivityLog, 0),
	}

	// 用户统计
	var totalUsers, activeUsers int64
	if err := s.db.Model(&domain.User{}).Count(&totalUsers).Error; err != nil {
		return nil, err
	}
	stats.TotalUsers = int(totalUsers)

	if err := s.db.Model(&domain.User{}).Where("is_active = ?", true).Count(&activeUsers).Error; err != nil {
		return nil, err
	}
	stats.ActiveUsers = int(activeUsers)

	// 邮箱统计
	var totalMailboxes, activeMailboxes int64
	if err := s.db.Model(&domain.Mailbox{}).Count(&totalMailboxes).Error; err != nil {
		return nil, err
	}
	stats.TotalMailboxes = int(totalMailboxes)

	if err := s.db.Model(&domain.Mailbox{}).Where("expires_at IS NULL OR expires_at > ?", time.Now()).Count(&activeMailboxes).Error; err != nil {
		return nil, err
	}
	stats.ActiveMailboxes = int(activeMailboxes)

	// 邮件统计
	var totalMessages, messagesToday int64
	if err := s.db.Model(&domain.Message{}).Count(&totalMessages).Error; err != nil {
		return nil, err
	}
	stats.TotalMessages = int(totalMessages)

	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&domain.Message{}).Where("created_at >= ?", today).Count(&messagesToday).Error; err != nil {
		return nil, err
	}
	stats.MessagesToday = int(messagesToday)

	// 按等级统计用户
	var tierStats []struct {
		Tier  domain.UserTier `json:"tier"`
		Count int             `json:"count"`
	}
	if err := s.db.Model(&domain.User{}).Select("tier, COUNT(*) as count").Group("tier").Scan(&tierStats).Error; err != nil {
		return nil, err
	}
	for _, stat := range tierStats {
		stats.UsersByTier[stat.Tier] = stat.Count
	}

	// 按角色统计用户
	var roleStats []struct {
		Role  domain.UserRole `json:"role"`
		Count int             `json:"count"`
	}
	if err := s.db.Model(&domain.User{}).Select("role, COUNT(*) as count").Group("role").Scan(&roleStats).Error; err != nil {
		return nil, err
	}
	for _, stat := range roleStats {
		stats.UsersByRole[stat.Role] = stat.Count
	}

	// 按域名统计邮箱
	var domainStats []struct {
		Domain string `json:"domain"`
		Count  int    `json:"count"`
	}
	if err := s.db.Model(&domain.Mailbox{}).Select("domain, COUNT(*) as count").Group("domain").Scan(&domainStats).Error; err != nil {
		return nil, err
	}
	for _, stat := range domainStats {
		stats.MailboxesByDomain[stat.Domain] = stat.Count
	}

	return stats, nil
}

// GetDomainStatistics 获取域名统计信息
func (s *Store) GetDomainStatistics(domainName string) (mailboxCount, messageCount int, err error) {
	// 统计邮箱数量
	var mailboxCountInt64 int64
	if err := s.db.Model(&domain.Mailbox{}).Where("domain = ?", domainName).Count(&mailboxCountInt64).Error; err != nil {
		return 0, 0, err
	}
	mailboxCount = int(mailboxCountInt64)

	// 统计邮件数量
	var messageCountInt64 int64
	if err := s.db.Model(&domain.Message{}).
		Joins("JOIN mailboxes ON messages.mailbox_id = mailboxes.id").
		Where("mailboxes.domain = ?", domainName).
		Count(&messageCountInt64).Error; err != nil {
		return 0, 0, err
	}
	messageCount = int(messageCountInt64)

	return mailboxCount, messageCount, nil
}

// ========== Alias Repository ==========

// SaveAlias 保存邮箱别名
func (s *Store) SaveAlias(alias *domain.MailboxAlias) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查邮箱是否存在
		var mailbox domain.Mailbox
		if err := tx.Where("id = ?", alias.MailboxID).First(&mailbox).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrMailboxNotFound
			}
			return err
		}

		// 检查别名地址是否已被使用
		var existingMailbox domain.Mailbox
		if err := tx.Where("address = ?", alias.Address).First(&existingMailbox).Error; err == nil {
			return fmt.Errorf("address already exists as mailbox")
		}

		// 检查地址是否被其他别名使用
		var existingAlias domain.MailboxAlias
		query := tx.Where("address = ?", alias.Address)
		if alias.ID != "" {
			query = query.Where("id != ?", alias.ID)
		}
		if err := query.First(&existingAlias).Error; err == nil {
			return ErrAliasExists
		}

		return tx.Save(alias).Error
	})
}

// GetAlias 根据ID获取别名
func (s *Store) GetAlias(aliasID string) (*domain.MailboxAlias, error) {
	var alias domain.MailboxAlias
	err := s.db.Where("id = ?", aliasID).First(&alias).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAliasNotFound
		}
		return nil, err
	}
	return &alias, nil
}

// GetAliasByAddress 根据地址获取别名
func (s *Store) GetAliasByAddress(address string) (*domain.MailboxAlias, error) {
	var alias domain.MailboxAlias
	err := s.db.Where("address = ?", address).First(&alias).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAliasNotFound
		}
		return nil, err
	}
	return &alias, nil
}

// ListAliasesByMailboxID 列出指定邮箱的所有别名
func (s *Store) ListAliasesByMailboxID(mailboxID string) ([]*domain.MailboxAlias, error) {
	var aliases []*domain.MailboxAlias
	err := s.db.Where("mailbox_id = ?", mailboxID).Find(&aliases).Error
	return aliases, err
}

// DeleteAlias 删除别名
func (s *Store) DeleteAlias(aliasID string) error {
	return s.db.Where("id = ?", aliasID).Delete(&domain.MailboxAlias{}).Error
}

// ========== System Config Repository ==========

// GetSystemConfig 获取系统配置
func (s *Store) GetSystemConfig() (*domain.SystemConfig, error) {
	// TODO: 实现数据库存储
	// 目前返回默认配置
	return domain.DefaultSystemConfig(), nil
}

// SaveSystemConfig 保存系统配置
func (s *Store) SaveSystemConfig(config *domain.SystemConfig) error {
	// TODO: 实现数据库存储
	// 目前只返回成功
	return nil
}

// ========== User Domain Repository ==========

// SaveUserDomain 保存用户域名
func (s *Store) SaveUserDomain(userDomain *domain.UserDomain) error {
	return s.db.Save(userDomain).Error
}

// GetUserDomain 根据ID获取用户域名
func (s *Store) GetUserDomain(domainID string) (*domain.UserDomain, error) {
	var userDomain domain.UserDomain
	err := s.db.Where("id = ?", domainID).First(&userDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user domain not found")
		}
		return nil, err
	}
	return &userDomain, nil
}

// GetUserDomainByDomain 根据域名获取用户域名
func (s *Store) GetUserDomainByDomain(domainName string) (*domain.UserDomain, error) {
	var userDomain domain.UserDomain
	err := s.db.Where("domain = ?", domainName).First(&userDomain).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user domain not found")
		}
		return nil, err
	}
	return &userDomain, nil
}

// ListUserDomainsByUserID 列出用户的所有域名
func (s *Store) ListUserDomainsByUserID(userID string) ([]*domain.UserDomain, error) {
	var userDomains []*domain.UserDomain
	err := s.db.Where("user_id = ?", userID).Find(&userDomains).Error
	return userDomains, err
}

// UpdateUserDomain 更新用户域名
func (s *Store) UpdateUserDomain(userDomain *domain.UserDomain) error {
	return s.db.Save(userDomain).Error
}

// DeleteUserDomain 删除用户域名
func (s *Store) DeleteUserDomain(domainID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否有活跃邮箱
		var count int64
		if err := tx.Model(&domain.Mailbox{}).Where("domain IN (SELECT domain FROM user_domains WHERE id = ?)", domainID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("cannot delete domain with active mailboxes")
		}

		return tx.Where("id = ?", domainID).Delete(&domain.UserDomain{}).Error
	})
}

// IncrementMailboxCount 增加域名邮箱计数
func (s *Store) IncrementMailboxCount(domainName string) error {
	return s.db.Model(&domain.UserDomain{}).
		Where("domain = ?", domainName).
		UpdateColumn("mailbox_count", gorm.Expr("mailbox_count + 1")).Error
}

// DecrementMailboxCount 减少域名邮箱计数
func (s *Store) DecrementMailboxCount(domainName string) error {
	return s.db.Model(&domain.UserDomain{}).
		Where("domain = ? AND mailbox_count > 0", domainName).
		UpdateColumn("mailbox_count", gorm.Expr("mailbox_count - 1")).Error
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ListAllUserDomains 列出所有用户域名
func (s *Store) ListAllUserDomains() ([]*domain.UserDomain, error) {
	var userDomains []*domain.UserDomain
	err := s.db.Find(&userDomains).Error
	if err != nil {
		return nil, err
	}
	return userDomains, nil
}

// ========== Webhook Repository ==========

// Webhook Repository 方法已在 search_webhook.go 中实现

// ========== Tag Repository ==========

// CreateTag 创建标签
func (s *Store) CreateTag(tag *domain.Tag) error {
	if tag.ID == "" {
		tag.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	tag.CreatedAt = now
	tag.UpdatedAt = now

	return s.db.Create(tag).Error
}

// GetTag 根据ID获取标签
func (s *Store) GetTag(id string) (*domain.Tag, error) {
	var tag domain.Tag
	err := s.db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// GetTagByName 根据用户ID和标签名获取标签
func (s *Store) GetTagByName(userID, name string) (*domain.Tag, error) {
	var tag domain.Tag
	err := s.db.Where("user_id = ? AND name = ?", userID, name).First(&tag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// ListTags 列出用户的所有标签（带使用计数）
func (s *Store) ListTags(userID string) ([]domain.TagWithCount, error) {
	var results []domain.TagWithCount

	err := s.db.Table("tags").
		Select("tags.*, COUNT(message_tags.tag_id) as message_count").
		Joins("LEFT JOIN message_tags ON tags.id = message_tags.tag_id").
		Where("tags.user_id = ?", userID).
		Group("tags.id").
		Order("tags.created_at DESC").
		Scan(&results).Error

	return results, err
}

// UpdateTag 更新标签
func (s *Store) UpdateTag(tag *domain.Tag) error {
	tag.UpdatedAt = time.Now().UTC()
	return s.db.Save(tag).Error
}

// DeleteTag 删除标签
func (s *Store) DeleteTag(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 先删除标签与邮件的关联
		if err := tx.Where("tag_id = ?", id).Delete(&domain.MessageTag{}).Error; err != nil {
			return err
		}

		// 再删除标签
		result := tx.Where("id = ?", id).Delete(&domain.Tag{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("tag not found")
		}

		return nil
	})
}

// AddMessageTag 为邮件添加标签
func (s *Store) AddMessageTag(messageID, tagID string) error {
	messageTag := &domain.MessageTag{
		MessageID: messageID,
		TagID:     tagID,
		CreatedAt: time.Now().UTC(),
	}

	// 使用 ON CONFLICT DO NOTHING 避免重复添加
	return s.db.Create(messageTag).Error
}

// RemoveMessageTag 移除邮件标签
func (s *Store) RemoveMessageTag(messageID, tagID string) error {
	result := s.db.Where("message_id = ? AND tag_id = ?", messageID, tagID).Delete(&domain.MessageTag{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("message tag not found")
	}
	return nil
}

// GetMessageTags 获取邮件的所有标签
func (s *Store) GetMessageTags(messageID string) ([]domain.Tag, error) {
	var tags []domain.Tag

	err := s.db.Table("tags").
		Joins("JOIN message_tags ON tags.id = message_tags.tag_id").
		Where("message_tags.message_id = ?", messageID).
		Order("tags.name").
		Find(&tags).Error

	return tags, err
}

// ListMessagesByTag 列出标签下的所有邮件
func (s *Store) ListMessagesByTag(tagID string) ([]domain.Message, error) {
	var messages []domain.Message

	err := s.db.Table("messages").
		Joins("JOIN message_tags ON messages.id = message_tags.message_id").
		Where("message_tags.tag_id = ?", tagID).
		Order("messages.created_at DESC").
		Find(&messages).Error

	return messages, err
}

// DeleteMessageTags 删除邮件的所有标签
func (s *Store) DeleteMessageTags(messageID string) error {
	return s.db.Where("message_id = ?", messageID).Delete(&domain.MessageTag{}).Error
}

// DeleteMessage 删除单封邮件
func (s *Store) DeleteMessage(mailboxID, messageID string) error {
	result := s.db.Where("id = ? AND mailbox_id = ?", messageID, mailboxID).Delete(&domain.Message{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}

// DeleteAllMessages 删除邮箱所有消息，返回删除数量
func (s *Store) DeleteAllMessages(mailboxID string) (int, error) {
	result := s.db.Where("mailbox_id = ?", mailboxID).Delete(&domain.Message{})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

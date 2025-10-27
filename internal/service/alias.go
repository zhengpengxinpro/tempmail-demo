package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage"
)

// AliasService 封装邮箱别名处理逻辑。
type AliasService struct {
	aliasRepo   storage.AliasRepository
	mailboxRepo storage.MailboxRepository
	cfg         *config.Config
}

// NewAliasService 创建别名业务服务。
func NewAliasService(aliasRepo storage.AliasRepository, mailboxRepo storage.MailboxRepository, cfg *config.Config) *AliasService {
	return &AliasService{
		aliasRepo:   aliasRepo,
		mailboxRepo: mailboxRepo,
		cfg:         cfg,
	}
}

// CreateAliasInput 定义创建别名的输入。
type CreateAliasInput struct {
	MailboxID string
	Address   string // 完整的别名地址，如 alias@temp.mail
}

// Create 创建一个新的邮箱别名。
func (s *AliasService) Create(input CreateAliasInput) (*domain.MailboxAlias, error) {
	// 验证邮箱是否存在
	mailbox, err := s.mailboxRepo.GetMailbox(input.MailboxID)
	if err != nil {
		return nil, fmt.Errorf("mailbox not found: %w", err)
	}

	// 标准化地址
	address := strings.ToLower(strings.TrimSpace(input.Address))

	// 验证地址格式
	if !strings.Contains(address, "@") {
		return nil, fmt.Errorf("invalid email address format")
	}

	// 提取域名并验证是否在允许列表中
	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid email address format")
	}
	domainName := parts[1]

	// 检查域名是否允许
	allowed := false
	for _, allowedDomain := range s.cfg.Mailbox.AllowedDomains {
		if domainName == allowedDomain {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("domain %s is not allowed", domainName)
	}

	// 别名不能与主邮箱地址相同
	if address == mailbox.Address {
		return nil, fmt.Errorf("alias cannot be the same as mailbox address")
	}

	// 创建别名
	alias := &domain.MailboxAlias{
		ID:        uuid.NewString(),
		MailboxID: input.MailboxID,
		Address:   address,
		CreatedAt: time.Now().UTC(),
		IsActive:  true,
	}

	if err := s.aliasRepo.SaveAlias(alias); err != nil {
		return nil, fmt.Errorf("failed to save alias: %w", err)
	}

	return alias, nil
}

// List 列出指定邮箱的所有别名。
func (s *AliasService) List(mailboxID string) ([]*domain.MailboxAlias, error) {
	// 验证邮箱是否存在
	if _, err := s.mailboxRepo.GetMailbox(mailboxID); err != nil {
		return nil, fmt.Errorf("mailbox not found: %w", err)
	}

	return s.aliasRepo.ListAliasesByMailboxID(mailboxID)
}

// Get 获取别名详情。
func (s *AliasService) Get(aliasID string) (*domain.MailboxAlias, error) {
	return s.aliasRepo.GetAlias(aliasID)
}

// GetByAddress 根据地址获取别名。
func (s *AliasService) GetByAddress(address string) (*domain.MailboxAlias, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	return s.aliasRepo.GetAliasByAddress(address)
}

// Delete 删除别名。
func (s *AliasService) Delete(mailboxID, aliasID string) error {
	// 获取别名
	alias, err := s.aliasRepo.GetAlias(aliasID)
	if err != nil {
		return fmt.Errorf("alias not found: %w", err)
	}

	// 验证别名属于该邮箱
	if alias.MailboxID != mailboxID {
		return fmt.Errorf("alias does not belong to this mailbox")
	}

	return s.aliasRepo.DeleteAlias(aliasID)
}

// Toggle 切换别名的激活状态。
func (s *AliasService) Toggle(mailboxID, aliasID string, isActive bool) error {
	// 获取别名
	alias, err := s.aliasRepo.GetAlias(aliasID)
	if err != nil {
		return fmt.Errorf("alias not found: %w", err)
	}

	// 验证别名属于该邮箱
	if alias.MailboxID != mailboxID {
		return fmt.Errorf("alias does not belong to this mailbox")
	}

	// 更新状态
	alias.IsActive = isActive
	return s.aliasRepo.SaveAlias(alias)
}

package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage"
)

var (
	ErrDomainNotAllowed = errors.New("domain not allowed")
	ErrPrefixInvalid    = errors.New("prefix invalid")
)

// MailboxService 封装邮箱相关业务操作。
type MailboxService struct {
	repo              storage.MailboxRepository
	store             domain.Store
	cfg               *config.Config
	domainSet         map[string]struct{}
	random            *rand.Rand
	tokenAlphabet     []rune
	userDomainService *UserDomainService     // 用于检查用户域名权限
	emailValidator    *domain.EmailValidator // 邮箱验证器
}

// NewMailboxService 创建邮箱业务服务。
func NewMailboxService(repo storage.MailboxRepository, store domain.Store, cfg *config.Config) *MailboxService {
	domainSet := make(map[string]struct{}, len(cfg.Mailbox.AllowedDomains))
	for _, d := range cfg.Mailbox.AllowedDomains {
		domainSet[d] = struct{}{}
	}

	return &MailboxService{
		repo:      repo,
		store:     store,
		cfg:       cfg,
		domainSet: domainSet,
		random:    rand.New(rand.NewSource(time.Now().UnixNano())),
		tokenAlphabet: []rune("abcdefghijklmnopqrstuvwxyz" +
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"),
		emailValidator: domain.NewEmailValidator(),
	}
}

// SetUserDomainService 设置用户域名服务（避免循环依赖）
func (s *MailboxService) SetUserDomainService(service *UserDomainService) {
	s.userDomainService = service
}

// CreateMailboxInput 定义创建邮箱所需的输入。
type CreateMailboxInput struct {
	Prefix    string
	Domain    string
	IPSource  string
	UserID    *string // 可选：关联的用户ID
	ExpiresAt *time.Time
}

// Create 创建新的临时邮箱。
func (s *MailboxService) Create(input CreateMailboxInput) (*domain.Mailbox, error) {
	selectedDomain := s.pickDomain(input.Domain)
	if selectedDomain == "" {
		return nil, ErrDomainNotAllowed
	}

	// 检查用户域名权限（独享模式检查）
	if s.userDomainService != nil {
		canCreate, err := s.userDomainService.CanCreateMailboxOnDomain(selectedDomain, input.UserID)
		if err != nil {
			return nil, err
		}
		if !canCreate {
			return nil, errors.New("no permission to create mailbox on this domain")
		}
	}

	localPart, err := s.resolveLocalPart(input.Prefix)
	if err != nil {
		return nil, err
	}

	address := fmt.Sprintf("%s@%s", localPart, selectedDomain)

	// 验证完整的邮箱地址
	if err := s.emailValidator.ValidateEmail(address); err != nil {
		// 如果是长度问题，返回前缀无效错误（保持向后兼容）
		if err == domain.ErrLocalPartTooLong || err == domain.ErrEmailTooLong {
			return nil, ErrPrefixInvalid
		}
		// 其他验证错误也返回前缀无效（保持简单的错误类型）
		return nil, ErrPrefixInvalid
	}

	id := uuid.NewString()
	token := s.generateToken(32)
	now := time.Now().UTC()

	mailbox := &domain.Mailbox{
		ID:        id,
		Address:   address,
		LocalPart: localPart,
		Domain:    selectedDomain,
		Token:     token,
		UserID:    input.UserID, // 关联用户ID（游客模式为nil）
		CreatedAt: now,
		IPSource:  input.IPSource,
	}

	if input.ExpiresAt != nil {
		mailbox.ExpiresAt = input.ExpiresAt
	}

	if err := s.repo.SaveMailbox(mailbox); err != nil {
		return nil, err
	}

	// 增加用户域名的邮箱计数
	if s.store != nil {
		s.store.IncrementMailboxCount(selectedDomain)
	}

	return mailbox, nil
}

// Get 根据 ID 获取邮箱。
func (s *MailboxService) Get(id string) (*domain.Mailbox, error) {
	return s.repo.GetMailbox(id)
}

// List 返回全部邮箱快照。
func (s *MailboxService) List() []domain.Mailbox {
	return s.repo.ListMailboxes()
}

// ListByUserID 返回指定用户的全部邮箱。
func (s *MailboxService) ListByUserID(userID string) []domain.Mailbox {
	return s.repo.ListMailboxesByUserID(userID)
}

// Delete 删除指定邮箱。
func (s *MailboxService) Delete(id string) error {
	// 先获取邮箱信息（用于减少计数）
	mailbox, err := s.repo.GetMailbox(id)
	if err != nil {
		return err
	}

	// 删除邮箱
	if err := s.repo.DeleteMailbox(id); err != nil {
		return err
	}

	// 减少用户域名的邮箱计数
	if s.store != nil && mailbox != nil {
		s.store.DecrementMailboxCount(mailbox.Domain)
	}

	return nil
}

// GetByAddress 根据邮箱地址获取邮箱。
func (s *MailboxService) GetByAddress(address string) (*domain.Mailbox, error) {
	address = strings.ToLower(strings.TrimSpace(address))
	if address == "" {
		return nil, ErrDomainNotAllowed
	}
	return s.repo.GetMailboxByAddress(address)
}

// pickDomain 挑选合法的邮箱域名。
func (s *MailboxService) pickDomain(requested string) string {
	if requested == "" {
		return s.cfg.Mailbox.AllowedDomains[0]
	}
	requested = strings.ToLower(strings.TrimSpace(requested))
	if _, ok := s.domainSet[requested]; ok {
		return requested
	}
	return ""
}

// resolveLocalPart 生成或验证邮箱前缀。
func (s *MailboxService) resolveLocalPart(prefix string) (string, error) {
	if prefix == "" {
		return s.generateRandomLocalPart(), nil
	}
	prefix = strings.ToLower(prefix)
	// 使用新的验证器验证本地部分
	if err := s.emailValidator.ValidateLocalPart(prefix); err != nil {
		// 返回原始错误类型，以便HTTP层能正确识别
		return "", ErrPrefixInvalid
	}
	return prefix, nil
}

// generateRandomLocalPart 生成随机前缀。
func (s *MailboxService) generateRandomLocalPart() string {
	// base on uuid truncated for uniqueness + randomness
	base := strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", ""))
	return base[:12]
}

// generateToken 生成邮箱访问令牌。
func (s *MailboxService) generateToken(length int) string {
	b := make([]rune, length)
	for i := 0; i < length; i++ {
		idx := s.random.Intn(len(s.tokenAlphabet))
		b[i] = s.tokenAlphabet[idx]
	}
	return string(b)
}

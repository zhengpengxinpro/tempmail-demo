package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"

	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
)

var (
	ErrDomainAlreadyExists = errors.New("domain already exists")
	ErrDomainNotFound      = errors.New("domain not found")
	ErrDomainNotVerified   = errors.New("domain not verified")
	ErrDomainVerifyFailed  = errors.New("domain verification failed")
	ErrNotDomainOwner      = errors.New("not domain owner")
	ErrDomainExclusiveMode = errors.New("domain is in exclusive mode")
	ErrInvalidDomain       = errors.New("invalid domain")
)

// UserDomainService 用户域名服务
type UserDomainService struct {
	store domain.Store
	cfg   *config.Config
}

// NewUserDomainService 创建用户域名服务
func NewUserDomainService(store domain.Store, cfg *config.Config) *UserDomainService {
	return &UserDomainService{
		store: store,
		cfg:   cfg,
	}
}

// AddDomainInput 添加域名输入
type AddDomainInput struct {
	UserID string
	Domain string
	Mode   domain.DomainMode
}

// AddDomain 添加用户域名
func (s *UserDomainService) AddDomain(input AddDomainInput) (*domain.UserDomain, error) {
	// 验证域名格式
	domainName := strings.TrimSpace(strings.ToLower(input.Domain))
	if !isValidDomain(domainName) {
		return nil, ErrInvalidDomain
	}

	// 检查域名是否已存在
	_, err := s.store.GetUserDomainByDomain(domainName)
	if err == nil {
		return nil, ErrDomainAlreadyExists
	}

	// 生成验证令牌
	verifyToken := generateToken(32)

	// 生成 MX 记录配置
	mxRecords := s.generateMXRecords(domainName)

	// 计算月费
	monthlyFee := 0.0
	if input.Mode == domain.DomainModeExclusive {
		monthlyFee = 9.99 // 独享模式月费
	}

	now := time.Now().UTC()
	userDomain := &domain.UserDomain{
		ID:           uuid.NewString(),
		UserID:       input.UserID,
		Domain:       domainName,
		Mode:         input.Mode,
		Status:       domain.DomainStatusPending,
		VerifyToken:  verifyToken,
		VerifyMethod: "dns_txt",
		CreatedAt:    now,
		MXRecords:    mxRecords,
		IsActive:     false, // 需要验证后才激活
		MailboxCount: 0,
		MonthlyFee:   monthlyFee,
	}

	if err := s.store.SaveUserDomain(userDomain); err != nil {
		return nil, err
	}

	return userDomain, nil
}

// VerifyDomain 验证域名所有权
func (s *UserDomainService) VerifyDomain(domainID, userID string) (*domain.UserDomain, error) {
	userDomain, err := s.store.GetUserDomain(domainID)
	if err != nil {
		return nil, ErrDomainNotFound
	}

	// 检查权限
	if userDomain.UserID != userID {
		return nil, ErrNotDomainOwner
	}

	// 如果已经验证，直接返回
	if userDomain.Status == domain.DomainStatusVerified {
		return userDomain, nil
	}

	// DNS TXT 记录验证
	expectedTxt := fmt.Sprintf("tempmail-verify=%s", userDomain.VerifyToken)
	verified, err := checkDNSTXTRecord(userDomain.Domain, expectedTxt)
	if err != nil || !verified {
		// 更新验证失败状态
		now := time.Now().UTC()
		userDomain.Status = domain.DomainStatusFailed
		userDomain.LastCheckAt = &now
		s.store.SaveUserDomain(userDomain)
		return nil, ErrDomainVerifyFailed
	}

	// 验证成功
	now := time.Now().UTC()
	userDomain.Status = domain.DomainStatusVerified
	userDomain.VerifiedAt = &now
	userDomain.LastCheckAt = &now
	userDomain.IsActive = true

	if err := s.store.SaveUserDomain(userDomain); err != nil {
		return nil, err
	}

	return userDomain, nil
}

// GetUserDomain 获取用户域名
func (s *UserDomainService) GetUserDomain(domainID, userID string) (*domain.UserDomain, error) {
	userDomain, err := s.store.GetUserDomain(domainID)
	if err != nil {
		return nil, ErrDomainNotFound
	}

	// 检查权限
	if userDomain.UserID != userID {
		return nil, ErrNotDomainOwner
	}

	return userDomain, nil
}

// ListUserDomains 获取用户的所有域名
func (s *UserDomainService) ListUserDomains(userID string) ([]*domain.UserDomain, error) {
	return s.store.ListUserDomainsByUserID(userID)
}

// DeleteUserDomain 删除用户域名
func (s *UserDomainService) DeleteUserDomain(domainID, userID string) error {
	userDomain, err := s.store.GetUserDomain(domainID)
	if err != nil {
		return ErrDomainNotFound
	}

	// 检查权限
	if userDomain.UserID != userID {
		return ErrNotDomainOwner
	}

	// TODO: 检查是否还有该域名下的邮箱，如果有则不允许删除
	if userDomain.MailboxCount > 0 {
		return errors.New("cannot delete domain with active mailboxes")
	}

	return s.store.DeleteUserDomain(domainID)
}

// UpdateDomainMode 更新域名模式（共享/独享）
func (s *UserDomainService) UpdateDomainMode(domainID, userID string, mode domain.DomainMode) (*domain.UserDomain, error) {
	userDomain, err := s.store.GetUserDomain(domainID)
	if err != nil {
		return nil, ErrDomainNotFound
	}

	// 检查权限
	if userDomain.UserID != userID {
		return nil, ErrNotDomainOwner
	}

	// 更新模式
	userDomain.Mode = mode
	if mode == domain.DomainModeExclusive {
		userDomain.MonthlyFee = 9.99
	} else {
		userDomain.MonthlyFee = 0
	}

	if err := s.store.SaveUserDomain(userDomain); err != nil {
		return nil, err
	}

	return userDomain, nil
}

// CanCreateMailboxOnDomain 检查用户是否可以在该域名下创建邮箱
func (s *UserDomainService) CanCreateMailboxOnDomain(domainName string, userID *string) (bool, error) {
	// 尝试获取用户域名
	userDomain, err := s.store.GetUserDomainByDomain(domainName)
	if err != nil {
		// 域名不存在，说明是系统默认域名，允许创建
		return true, nil
	}

	// 检查域名状态
	if !userDomain.IsActive || userDomain.Status != domain.DomainStatusVerified {
		return false, ErrDomainNotVerified
	}

	// 检查是否过期
	if userDomain.ExpiresAt != nil && time.Now().After(*userDomain.ExpiresAt) {
		return false, errors.New("domain expired")
	}

	// 如果是共享模式，允许任何人创建
	if userDomain.Mode == domain.DomainModeShared {
		return true, nil
	}

	// 如果是独享模式，只有所有者可以创建
	if userDomain.Mode == domain.DomainModeExclusive {
		if userID == nil || *userID != userDomain.UserID {
			return false, ErrDomainExclusiveMode
		}
		return true, nil
	}

	return false, errors.New("unknown domain mode")
}

// GetDomainSetupInstructions 获取域名配置说明
func (s *UserDomainService) GetDomainSetupInstructions(domainID, userID string) (map[string]interface{}, error) {
	userDomain, err := s.store.GetUserDomain(domainID)
	if err != nil {
		return nil, ErrDomainNotFound
	}

	// 检查权限
	if userDomain.UserID != userID {
		return nil, ErrNotDomainOwner
	}

	// 构建配置说明
	instructions := map[string]interface{}{
		"domain": userDomain.Domain,
		"status": userDomain.Status,
		"steps": []map[string]interface{}{
			{
				"step":        1,
				"title":       "添加 TXT 记录验证域名所有权",
				"description": "在您的 DNS 提供商处添加以下 TXT 记录：",
				"record": map[string]string{
					"type":  "TXT",
					"name":  "@",
					"value": fmt.Sprintf("tempmail-verify=%s", userDomain.VerifyToken),
					"ttl":   "3600",
				},
			},
			{
				"step":        2,
				"title":       "添加 MX 记录接收邮件",
				"description": "添加以下 MX 记录以接收邮件：",
				"records":     s.formatMXRecords(userDomain.MXRecords),
			},
			{
				"step":        3,
				"title":       "等待 DNS 生效并验证",
				"description": "DNS 记录通常需要 5-30 分钟生效，请耐心等待后点击验证按钮。",
			},
		},
	}

	return instructions, nil
}

// generateMXRecords 生成 MX 记录配置
func (s *UserDomainService) generateMXRecords(domainName string) []string {
	// 获取服务器地址（从配置或环境变量）
	// 这里假设邮件服务器地址是当前服务器
	serverHost := s.cfg.SMTP.Domain
	if serverHost == "" {
		serverHost = "mail.tempmail.dev"
	}

	return []string{
		fmt.Sprintf("10 %s", serverHost),
	}
}

// formatMXRecords 格式化 MX 记录显示
func (s *UserDomainService) formatMXRecords(mxRecords []string) []map[string]string {
	result := make([]map[string]string, len(mxRecords))
	for i, record := range mxRecords {
		parts := strings.Split(record, " ")
		result[i] = map[string]string{
			"type":     "MX",
			"name":     "@",
			"priority": parts[0],
			"value":    parts[1],
			"ttl":      "3600",
		}
	}
	return result
}

// checkDNSTXTRecord 检查 DNS TXT 记录
func checkDNSTXTRecord(domain, expectedValue string) (bool, error) {
	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		return false, err
	}

	for _, txt := range txtRecords {
		if strings.TrimSpace(txt) == expectedValue {
			return true, nil
		}
	}

	return false, nil
}

// isValidDomain 验证域名格式
func isValidDomain(domain string) bool {
	if domain == "" || len(domain) > 253 {
		return false
	}

	// 简单的域名格式验证
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		// 只允许字母、数字和连字符
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}

	return true
}

// generateToken 生成随机令牌
func generateToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

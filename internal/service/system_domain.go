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
	ErrSystemDomainAlreadyExists = errors.New("system domain already exists")
	ErrSystemDomainNotFound      = errors.New("system domain not found")
	ErrSystemDomainNotVerified   = errors.New("system domain not verified")
	ErrSystemDomainVerifyFailed  = errors.New("system domain verification failed")
	ErrSystemDomainHasMailboxes  = errors.New("cannot delete domain with active mailboxes")
	ErrInvalidSystemDomain       = errors.New("invalid system domain")
	ErrCannotDeleteDefaultDomain = errors.New("cannot delete default domain")
)

// SystemDomainService 系统域名服务
type SystemDomainService struct {
	store domain.Store
	cfg   *config.Config
}

// NewSystemDomainService 创建系统域名服务
func NewSystemDomainService(store domain.Store, cfg *config.Config) *SystemDomainService {
	return &SystemDomainService{
		store: store,
		cfg:   cfg,
	}
}

// GetStore 获取存储接口（用于内部初始化）
func (s *SystemDomainService) GetStore() domain.Store {
	return s.store
}

// AddSystemDomainInput 添加系统域名输入
type AddSystemDomainInput struct {
	Domain    string // 域名
	CreatedBy string // 创建者用户ID（管理员）
	Notes     string // 备注
}

// AddSystemDomain 添加系统域名
//
// 管理员添加系统域名，生成 DNS 验证令牌，需要验证后才能激活
//
// 参数:
//   - input: 添加域名输入
//
// 返回值:
//   - *domain.SystemDomain: 创建的系统域名
//   - error: 错误信息
func (s *SystemDomainService) AddSystemDomain(input AddSystemDomainInput) (*domain.SystemDomain, error) {
	// 验证域名格式
	domainName := strings.TrimSpace(strings.ToLower(input.Domain))
	if !isValidSystemDomain(domainName) {
		return nil, ErrInvalidSystemDomain
	}

	// 检查域名是否已存在
	_, err := s.store.GetSystemDomainByDomain(domainName)
	if err == nil {
		return nil, ErrSystemDomainAlreadyExists
	}

	// 生成验证令牌
	verifyToken := generateSystemToken(32)

	// 生成 MX 记录配置
	mxRecords := s.generateSystemMXRecords(domainName)

	now := time.Now().UTC()
	sysDomain := &domain.SystemDomain{
		ID:           uuid.NewString(),
		Domain:       domainName,
		Status:       domain.SystemDomainStatusPending,
		VerifyToken:  verifyToken,
		VerifyMethod: "dns_txt",
		CreatedAt:    now,
		CreatedBy:    input.CreatedBy,
		MXRecords:    mxRecords,
		IsActive:     false, // 需要验证后才激活
		IsDefault:    false,
		MailboxCount: 0,
		Notes:        input.Notes,
	}

	if err := s.store.SaveSystemDomain(sysDomain); err != nil {
		return nil, err
	}

	return sysDomain, nil
}

// VerifySystemDomain 验证系统域名所有权
//
// 通过 DNS TXT 记录验证域名所有权，验证成功后激活域名
//
// 参数:
//   - domainID: 域名ID
//
// 返回值:
//   - *domain.SystemDomain: 验证后的域名信息
//   - error: 错误信息
func (s *SystemDomainService) VerifySystemDomain(domainID string) (*domain.SystemDomain, error) {
	sysDomain, err := s.store.GetSystemDomain(domainID)
	if err != nil {
		return nil, ErrSystemDomainNotFound
	}

	// 如果已经验证，直接返回
	if sysDomain.Status == domain.SystemDomainStatusVerified {
		return sysDomain, nil
	}

	// DNS TXT 记录验证
	expectedTxt := fmt.Sprintf("tempmail-verify=%s", sysDomain.VerifyToken)
	verified, err := checkSystemDNSTXTRecord(sysDomain.Domain, expectedTxt)
	if err != nil || !verified {
		// 更新验证失败状态
		now := time.Now().UTC()
		sysDomain.Status = domain.SystemDomainStatusFailed
		sysDomain.LastCheckAt = &now
		s.store.SaveSystemDomain(sysDomain)
		return nil, ErrSystemDomainVerifyFailed
	}

	// 验证成功
	now := time.Now().UTC()
	sysDomain.Status = domain.SystemDomainStatusVerified
	sysDomain.VerifiedAt = &now
	sysDomain.LastCheckAt = &now
	sysDomain.IsActive = true

	if err := s.store.SaveSystemDomain(sysDomain); err != nil {
		return nil, err
	}

	return sysDomain, nil
}

// RecoverSystemDomain 找回系统域名
//
// 如果域名被误删除或验证失败，可以通过 DNS 验证重新找回
//
// 参数:
//   - domainName: 域名
//   - createdBy: 创建者用户ID
//
// 返回值:
//   - *domain.SystemDomain: 找回的域名信息
//   - error: 错误信息
func (s *SystemDomainService) RecoverSystemDomain(domainName string, createdBy string) (*domain.SystemDomain, error) {
	domainName = strings.TrimSpace(strings.ToLower(domainName))
	if !isValidSystemDomain(domainName) {
		return nil, ErrInvalidSystemDomain
	}

	// 检查域名是否已存在
	existingDomain, err := s.store.GetSystemDomainByDomain(domainName)
	if err == nil {
		// 域名已存在，尝试重新验证
		return s.VerifySystemDomain(existingDomain.ID)
	}

	// 域名不存在，尝试查找 DNS TXT 记录中的验证令牌
	// 查询所有 TXT 记录
	txtRecords, err := net.LookupTXT(domainName)
	if err != nil {
		return nil, errors.New("无法查询域名 DNS 记录")
	}

	// 查找 tempmail-verify= 开头的记录
	var verifyToken string
	for _, txt := range txtRecords {
		if strings.HasPrefix(txt, "tempmail-verify=") {
			verifyToken = strings.TrimPrefix(txt, "tempmail-verify=")
			break
		}
	}

	if verifyToken == "" {
		return nil, errors.New("未找到有效的验证令牌，请先添加 DNS TXT 记录")
	}

	// 创建新的域名记录
	mxRecords := s.generateSystemMXRecords(domainName)
	now := time.Now().UTC()

	sysDomain := &domain.SystemDomain{
		ID:           uuid.NewString(),
		Domain:       domainName,
		Status:       domain.SystemDomainStatusVerified, // 直接标记为已验证
		VerifyToken:  verifyToken,
		VerifyMethod: "dns_txt",
		CreatedAt:    now,
		VerifiedAt:   &now,
		LastCheckAt:  &now,
		CreatedBy:    createdBy,
		MXRecords:    mxRecords,
		IsActive:     true,
		IsDefault:    false,
		MailboxCount: 0,
		Notes:        "通过找回功能恢复",
	}

	if err := s.store.SaveSystemDomain(sysDomain); err != nil {
		return nil, err
	}

	return sysDomain, nil
}

// ListSystemDomains 列出所有系统域名
func (s *SystemDomainService) ListSystemDomains() ([]*domain.SystemDomain, error) {
	return s.store.ListSystemDomains()
}

// GetSystemDomain 获取系统域名详情
func (s *SystemDomainService) GetSystemDomain(domainID string) (*domain.SystemDomain, error) {
	sysDomain, err := s.store.GetSystemDomain(domainID)
	if err != nil {
		return nil, ErrSystemDomainNotFound
	}
	return sysDomain, nil
}

// DeleteSystemDomain 删除系统域名
//
// 参数:
//   - domainID: 域名ID
//
// 返回值:
//   - error: 错误信息
func (s *SystemDomainService) DeleteSystemDomain(domainID string) error {
	sysDomain, err := s.store.GetSystemDomain(domainID)
	if err != nil {
		return ErrSystemDomainNotFound
	}

	// 不能删除默认域名
	if sysDomain.IsDefault {
		return ErrCannotDeleteDefaultDomain
	}

	// 检查是否还有邮箱
	if sysDomain.MailboxCount > 0 {
		return ErrSystemDomainHasMailboxes
	}

	return s.store.DeleteSystemDomain(domainID)
}

// ToggleDomainStatus 切换域名状态（启用/禁用）
//
// 参数:
//   - domainID: 域名ID
//   - isActive: 是否激活
//
// 返回值:
//   - *domain.SystemDomain: 更新后的域名信息
//   - error: 错误信息
func (s *SystemDomainService) ToggleDomainStatus(domainID string, isActive bool) (*domain.SystemDomain, error) {
	sysDomain, err := s.store.GetSystemDomain(domainID)
	if err != nil {
		return nil, ErrSystemDomainNotFound
	}

	// 只有已验证的域名才能启用
	if isActive && sysDomain.Status != domain.SystemDomainStatusVerified {
		return nil, ErrSystemDomainNotVerified
	}

	sysDomain.IsActive = isActive

	if err := s.store.SaveSystemDomain(sysDomain); err != nil {
		return nil, err
	}

	return sysDomain, nil
}

// SetDefaultDomain 设置默认域名
//
// 参数:
//   - domainID: 域名ID
//
// 返回值:
//   - error: 错误信息
func (s *SystemDomainService) SetDefaultDomain(domainID string) error {
	sysDomain, err := s.store.GetSystemDomain(domainID)
	if err != nil {
		return ErrSystemDomainNotFound
	}

	// 只有已验证且激活的域名才能设为默认
	if !sysDomain.IsActive || sysDomain.Status != domain.SystemDomainStatusVerified {
		return ErrSystemDomainNotVerified
	}

	// 取消其他域名的默认状态
	allDomains, err := s.store.ListSystemDomains()
	if err != nil {
		return err
	}

	for _, d := range allDomains {
		if d.ID != domainID && d.IsDefault {
			d.IsDefault = false
			s.store.SaveSystemDomain(d)
		}
	}

	// 设置当前域名为默认
	sysDomain.IsDefault = true
	return s.store.SaveSystemDomain(sysDomain)
}

// GetSetupInstructions 获取域名配置说明
//
// 参数:
//   - domainID: 域名ID
//
// 返回值:
//   - map[string]interface{}: 配置说明
//   - error: 错误信息
func (s *SystemDomainService) GetSetupInstructions(domainID string) (map[string]interface{}, error) {
	sysDomain, err := s.store.GetSystemDomain(domainID)
	if err != nil {
		return nil, ErrSystemDomainNotFound
	}

	// 构建配置说明
	instructions := map[string]interface{}{
		"domain": sysDomain.Domain,
		"status": sysDomain.Status,
		"steps": []map[string]interface{}{
			{
				"step":        1,
				"title":       "添加 TXT 记录验证域名所有权",
				"description": "在您的 DNS 提供商处添加以下 TXT 记录：",
				"record": map[string]string{
					"type":  "TXT",
					"name":  "@",
					"value": fmt.Sprintf("tempmail-verify=%s", sysDomain.VerifyToken),
					"ttl":   "3600",
				},
			},
			{
				"step":        2,
				"title":       "添加 MX 记录接收邮件",
				"description": "添加以下 MX 记录以接收邮件：",
				"records":     s.formatSystemMXRecords(sysDomain.MXRecords),
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

// CleanupUnverifiedDomains 清理未验证的系统域名
//
// 删除创建超过 24 小时仍未验证的域名
//
// 返回值:
//   - int: 删除的域名数量
//   - error: 错误信息
func (s *SystemDomainService) CleanupUnverifiedDomains() (int, error) {
	// 删除 24 小时前创建且未验证的域名
	before := time.Now().UTC().Add(-24 * time.Hour)
	return s.store.DeleteUnverifiedSystemDomains(before)
}

// GetActiveDomains 获取所有激活的系统域名
//
// 返回值:
//   - []*domain.SystemDomain: 激活的系统域名列表
//   - error: 错误信息
func (s *SystemDomainService) GetActiveDomains() ([]*domain.SystemDomain, error) {
	return s.store.ListActiveSystemDomains()
}

// GetAllActiveDomains 获取所有激活的域名（包括系统域名和用户域名）
//
// 用于 SMTP 服务器动态加载域名列表
//
// 返回值:
//   - []string: 激活的域名列表
//   - error: 错误信息
func (s *SystemDomainService) GetAllActiveDomains() ([]string, error) {
	systemDomains, err := s.store.ListActiveSystemDomains()
	if err != nil {
		return nil, err
	}

	domains := make([]string, 0, len(systemDomains))
	for _, d := range systemDomains {
		domains = append(domains, d.Domain)
	}

	return domains, nil
}

// generateSystemMXRecords 生成 MX 记录配置
func (s *SystemDomainService) generateSystemMXRecords(domainName string) []string {
	// 获取邮件服务器地址
	serverHost := s.cfg.SMTP.Domain
	if serverHost == "" {
		serverHost = "mail.tempmail.dev"
	}

	return []string{
		fmt.Sprintf("10 %s", serverHost),
	}
}

// formatSystemMXRecords 格式化 MX 记录显示
func (s *SystemDomainService) formatSystemMXRecords(mxRecords []string) []map[string]string {
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

// checkSystemDNSTXTRecord 检查 DNS TXT 记录
func checkSystemDNSTXTRecord(domainName, expectedValue string) (bool, error) {
	txtRecords, err := net.LookupTXT(domainName)
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

// isValidSystemDomain 验证系统域名格式
func isValidSystemDomain(domainName string) bool {
	if domainName == "" || len(domainName) > 253 {
		return false
	}

	// 简单的域名格式验证
	parts := strings.Split(domainName, ".")
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

// generateSystemToken 生成随机令牌
func generateSystemToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

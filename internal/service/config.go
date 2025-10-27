package service

import (
	"errors"
	"time"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage"
)

var (
	// ErrConfigNotFound 配置未找到
	ErrConfigNotFound = errors.New("config not found")
	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("invalid config")
)

// ConfigService 系统配置服务
type ConfigService struct {
	store storage.Store
}

// NewConfigService 创建配置服务
func NewConfigService(store storage.Store) *ConfigService {
	return &ConfigService{
		store: store,
	}
}

// GetSystemConfig 获取系统配置
func (s *ConfigService) GetSystemConfig() (*domain.SystemConfig, error) {
	config, err := s.store.GetSystemConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateSystemConfigInput 更新系统配置输入
type UpdateSystemConfigInput struct {
	SMTP      *domain.SMTPConfig      `json:"smtp,omitempty"`
	Mailbox   *domain.MailboxConfig   `json:"mailbox,omitempty"`
	RateLimit *domain.RateLimitConfig `json:"rateLimit,omitempty"`
	Security  *domain.SecurityConfig  `json:"security,omitempty"`
	UpdatedBy string                  `json:"-"` // 更新者用户ID
}

// UpdateSystemConfig 更新系统配置（需要超级管理员权限）
func (s *ConfigService) UpdateSystemConfig(input UpdateSystemConfigInput) (*domain.SystemConfig, error) {
	// 获取当前配置
	config, err := s.store.GetSystemConfig()
	if err != nil {
		return nil, err
	}

	// 更新字段
	if input.SMTP != nil {
		// 验证SMTP配置
		if input.SMTP.BindAddr == "" {
			return nil, errors.New("SMTP BindAddr不能为空")
		}
		if input.SMTP.Domain == "" {
			return nil, errors.New("SMTP Domain不能为空")
		}
		if input.SMTP.MaxSize <= 0 {
			return nil, errors.New("SMTP MaxSize必须大于0")
		}
		config.SMTP = *input.SMTP
	}

	if input.Mailbox != nil {
		// 验证邮箱配置
		if input.Mailbox.DefaultTTL == "" {
			return nil, errors.New("Mailbox DefaultTTL不能为空")
		}
		// 验证TTL格式
		if _, err := time.ParseDuration(input.Mailbox.DefaultTTL); err != nil {
			return nil, errors.New("Mailbox DefaultTTL格式无效")
		}
		if input.Mailbox.MaxPerIP <= 0 {
			return nil, errors.New("Mailbox MaxPerIP必须大于0")
		}
		if len(input.Mailbox.AllowedDomains) == 0 {
			return nil, errors.New("Mailbox AllowedDomains不能为空")
		}
		config.Mailbox = *input.Mailbox
	}

	if input.RateLimit != nil {
		// 验证限流配置
		if input.RateLimit.RequestsPerMinute <= 0 {
			return nil, errors.New("RateLimit RequestsPerMinute必须大于0")
		}
		if input.RateLimit.BurstSize <= 0 {
			return nil, errors.New("RateLimit BurstSize必须大于0")
		}
		config.RateLimit = *input.RateLimit
	}

	if input.Security != nil {
		// 验证安全配置
		if input.Security.JWTAccessExpiry == "" {
			return nil, errors.New("Security JWTAccessExpiry不能为空")
		}
		if input.Security.JWTRefreshExpiry == "" {
			return nil, errors.New("Security JWTRefreshExpiry不能为空")
		}
		// 验证JWT有效期格式
		if _, err := time.ParseDuration(input.Security.JWTAccessExpiry); err != nil {
			return nil, errors.New("Security JWTAccessExpiry格式无效")
		}
		if _, err := time.ParseDuration(input.Security.JWTRefreshExpiry); err != nil {
			return nil, errors.New("Security JWTRefreshExpiry格式无效")
		}
		if input.Security.PasswordMinLength < 6 {
			return nil, errors.New("Security PasswordMinLength必须至少为6")
		}
		if input.Security.MaxLoginAttempts <= 0 {
			return nil, errors.New("Security MaxLoginAttempts必须大于0")
		}
		config.Security = *input.Security
	}

	// 设置更新者
	config.UpdatedBy = input.UpdatedBy
	config.UpdatedAt = time.Now()

	// 保存配置
	if err := s.store.SaveSystemConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// ResetSystemConfig 重置系统配置为默认值（需要超级管理员权限）
func (s *ConfigService) ResetSystemConfig(updatedBy string) (*domain.SystemConfig, error) {
	config := domain.DefaultSystemConfig()
	config.UpdatedBy = updatedBy
	config.UpdatedAt = time.Now()

	if err := s.store.SaveSystemConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

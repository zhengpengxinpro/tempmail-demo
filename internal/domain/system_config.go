package domain

import "time"

// SystemConfig 系统配置
type SystemConfig struct {
	ID        string    `json:"id"`
	SMTP      SMTPConfig      `json:"smtp"`
	Mailbox   MailboxConfig   `json:"mailbox"`
	RateLimit RateLimitConfig `json:"rateLimit"`
	Security  SecurityConfig  `json:"security"`
	UpdatedAt time.Time       `json:"updatedAt"`
	UpdatedBy string          `json:"updatedBy"` // 更新者用户ID
}

// SMTPConfig SMTP服务配置
type SMTPConfig struct {
	BindAddr   string `json:"bindAddr"`   // 监听地址，如 ":25"
	Domain     string `json:"domain"`     // SMTP服务器域名
	MaxSize    int64  `json:"maxSize"`    // 最大邮件大小（字节），默认10MB
	ReadTimeout int   `json:"readTimeout"` // 读取超时（秒），默认60
}

// MailboxConfig 邮箱配置
type MailboxConfig struct {
	DefaultTTL         string   `json:"defaultTtl"`         // 默认过期时间，如 "24h"
	MaxPerIP           int      `json:"maxPerIp"`           // 单IP最大邮箱数
	AllowedDomains     []string `json:"allowedDomains"`     // 允许的域名列表
	RequireVerification bool     `json:"requireVerification"` // 是否需要邮箱验证
	MaxAliases         int      `json:"maxAliases"`         // 每个邮箱最大别名数
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled            bool `json:"enabled"`            // 是否启用限流
	RequestsPerMinute  int  `json:"requestsPerMinute"`  // 每分钟请求数
	BurstSize          int  `json:"burstSize"`          // 突发请求数
	CreateMailboxLimit int  `json:"createMailboxLimit"` // 创建邮箱限流（每小时）
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	JWTAccessExpiry  string `json:"jwtAccessExpiry"`  // JWT访问令牌有效期，如 "15m"
	JWTRefreshExpiry string `json:"jwtRefreshExpiry"` // JWT刷新令牌有效期，如 "7d"
	PasswordMinLength int    `json:"passwordMinLength"` // 最小密码长度
	EnableCaptcha    bool   `json:"enableCaptcha"`    // 是否启用验证码
	MaxLoginAttempts int    `json:"maxLoginAttempts"` // 最大登录尝试次数
}

// DefaultSystemConfig 返回默认系统配置
func DefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		ID: "system",
		SMTP: SMTPConfig{
			BindAddr:    ":25",
			Domain:      "temp.mail",
			MaxSize:     10 * 1024 * 1024, // 10MB
			ReadTimeout: 60,
		},
		Mailbox: MailboxConfig{
			DefaultTTL:          "24h",
			MaxPerIP:            3,
			AllowedDomains:      []string{"temp.mail"},
			RequireVerification: false,
			MaxAliases:          5,
		},
		RateLimit: RateLimitConfig{
			Enabled:            true,
			RequestsPerMinute:  60,
			BurstSize:          10,
			CreateMailboxLimit: 10,
		},
		Security: SecurityConfig{
			JWTAccessExpiry:   "15m",
			JWTRefreshExpiry:  "7d",
			PasswordMinLength: 8,
			EnableCaptcha:     false,
			MaxLoginAttempts:  5,
		},
		UpdatedAt: time.Now(),
	}
}

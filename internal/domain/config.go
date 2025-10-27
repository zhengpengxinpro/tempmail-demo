package domain

// Config 应用配置（用于管理服务）
type Config struct {
	AllowedDomains []string // 允许的邮箱域名列表
}

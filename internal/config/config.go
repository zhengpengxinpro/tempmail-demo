package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// ServerConfig 定义 HTTP 服务器的监听配置参数
type ServerConfig struct {
	Host string // 监听地址，默认 "0.0.0.0"
	Port int    // 监听端口，默认 8080
}

// MailboxConfig 定义邮箱服务的核心业务配置
type MailboxConfig struct {
	AllowedDomains []string      // 允许创建邮箱的域名列表
	DefaultTTL     time.Duration // 邮箱默认生存时间，过期后自动清理
	MaxPerIP       int           // 单个 IP 地址最多可创建的邮箱数量
}

// SMTPConfig 定义 SMTP 邮件接收服务器的配置
type SMTPConfig struct {
	BindAddr string // SMTP 服务监听地址，格式 "host:port"，默认 ":25"
	Domain   string // SMTP 服务器域名，用于 HELO/EHLO 响应
}

// CORSConfig 定义跨域资源共享 (CORS) 配置
type CORSConfig struct {
	AllowedOrigins []string // 允许的来源列表，"*" 表示允许所有来源
}

// LogConfig 定义日志系统配置
type LogConfig struct {
	Level       string // 日志级别: debug, info, warn, error
	Development bool   // 开发模式: 启用彩色输出和详细堆栈信息
}

// DatabaseConfig 定义数据库连接配置（支持 MySQL 和 PostgreSQL）
type DatabaseConfig struct {
	Type            string        // 数据库类型: "mysql" 或 "postgres"
	DSN             string        // 数据库连接字符串
	                             // MySQL 格式: user:password@tcp(host:port)/dbname?parseTime=true&charset=utf8mb4
	                             // PostgreSQL 格式: postgres://user:password@host:port/dbname?sslmode=disable
	MaxOpenConns    int           // 最大打开连接数，默认 25
	MaxIdleConns    int           // 最大空闲连接数，默认 5
	ConnMaxLifetime time.Duration // 连接最大生命周期，默认 5 分钟
}

// RedisConfig 定义 Redis 缓存服务配置
type RedisConfig struct {
	Address  string // Redis 服务地址，格式 "host:port"，默认 "localhost:6379"
	Password string // Redis 认证密码，留空表示无密码
	DB       int    // Redis 数据库编号，默认 0
}

// JWTConfig 定义 JWT 认证相关配置
type JWTConfig struct {
	Secret        string        // JWT 签名密钥，必须至少 32 字符
	Issuer        string        // JWT 签发者标识，默认 "tempmail"
	AccessExpiry  time.Duration // 访问令牌有效期，默认 15 分钟
	RefreshExpiry time.Duration // 刷新令牌有效期，默认 7 天
}

// Config 是系统核心配置的根结构体，包含所有子系统的配置
type Config struct {
	Server   ServerConfig   // HTTP 服务器配置
	Mailbox  MailboxConfig  // 邮箱服务配置
	SMTP     SMTPConfig     // SMTP 服务配置
	CORS     CORSConfig     // 跨域配置
	Log      LogConfig      // 日志配置
	Database DatabaseConfig // 数据库配置
	Redis    RedisConfig    // Redis 配置
	JWT      JWTConfig      // JWT 认证配置
}

// Load 从环境变量和 .env 文件加载系统配置
//
// 配置加载优先级（从高到低）：
//   1. 系统环境变量（最高优先级）
//   2. .env 文件（如果存在）
//   3. 默认值
//
// 环境变量前缀: TEMPMAIL_
// 例如: TEMPMAIL_SERVER_HOST, TEMPMAIL_JWT_SECRET
//
// .env 文件位置：
//   - 当前目录的 .env
//   - 父目录的 .env（如果在 backend/ 子目录中运行）
//
// 返回值:
//   - *Config: 加载成功的配置对象
//   - error: 配置验证失败时返回错误
func Load() (*Config, error) {
	// 尝试加载 .env 文件（静默失败，因为 .env 文件是可选的）
	loadEnvFile()

	viper.SetEnvPrefix("tempmail")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("mailbox.allowed_domains", "temp.mail")
	viper.SetDefault("mailbox.default_ttl", "1h")
	viper.SetDefault("mailbox.max_per_ip", 3)
	viper.SetDefault("smtp.bind_addr", ":25")
	viper.SetDefault("smtp.domain", "temp.mail")
	viper.SetDefault("cors.allowed_origins", "*")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.development", false)
	viper.SetDefault("database.type", "")     // 默认为空，使用内存存储
	viper.SetDefault("database.dsn", "")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.issuer", "tempmail")
	viper.SetDefault("jwt.access_expiry", "15m")
	viper.SetDefault("jwt.refresh_expiry", "7d")

	serverHost := viper.GetString("server.host")
	serverPort := viper.GetInt("server.port")

	ttlStr := viper.GetString("mailbox.default_ttl")
	defaultTTL, err := time.ParseDuration(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid mailbox.default_ttl: %w", err)
	}

	domainList := parseDomains(viper.GetString("mailbox.allowed_domains"))
	if len(domainList) == 0 {
		return nil, fmt.Errorf("mailbox.allowed_domains must not be empty")
	}

	maxPerIP := viper.GetInt("mailbox.max_per_ip")
	if maxPerIP <= 0 {
		maxPerIP = 3
	}

	corsOrigins := parseList(viper.GetString("cors.allowed_origins"))
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"*"}
	}

	connMaxLifetime, err := time.ParseDuration(viper.GetString("database.conn_max_lifetime"))
	if err != nil {
		connMaxLifetime = 5 * time.Minute
	}

	accessExpiry, err := time.ParseDuration(viper.GetString("jwt.access_expiry"))
	if err != nil {
		accessExpiry = 15 * time.Minute
	}

	refreshExpiry, err := time.ParseDuration(viper.GetString("jwt.refresh_expiry"))
	if err != nil {
		refreshExpiry = 7 * 24 * time.Hour
	}

	jwtSecret := viper.GetString("jwt.secret")

	// 安全检查：禁止使用默认的 JWT secret
	if jwtSecret == "change-me-in-production" {
		return nil, fmt.Errorf("SECURITY ERROR: JWT secret cannot be the default value. Please set TEMPMAIL_JWT_SECRET environment variable")
	}

	// JWT secret 必须至少 32 字符
	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("SECURITY ERROR: JWT secret must be at least 32 characters long")
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: serverHost,
			Port: serverPort,
		},
		Mailbox: MailboxConfig{
			AllowedDomains: domainList,
			DefaultTTL:     defaultTTL,
			MaxPerIP:       maxPerIP,
		},
		SMTP: SMTPConfig{
			BindAddr: viper.GetString("smtp.bind_addr"),
			Domain:   viper.GetString("smtp.domain"),
		},
		CORS: CORSConfig{
			AllowedOrigins: corsOrigins,
		},
		Log: LogConfig{
			Level:       viper.GetString("log.level"),
			Development: viper.GetBool("log.development"),
		},
	Database: DatabaseConfig{
		Type:            viper.GetString("database.type"),
		DSN:             viper.GetString("database.dsn"),
		MaxOpenConns:    viper.GetInt("database.max_open_conns"),
		MaxIdleConns:    viper.GetInt("database.max_idle_conns"),
		ConnMaxLifetime: connMaxLifetime,
	},
		Redis: RedisConfig{
			Address:  viper.GetString("redis.address"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
		},
		JWT: JWTConfig{
			Secret:        jwtSecret,
			Issuer:        viper.GetString("jwt.issuer"),
			AccessExpiry:  accessExpiry,
			RefreshExpiry: refreshExpiry,
		},
	}

	return cfg, nil
}

// parseDomains 将逗号分隔的域名字符串解析为小写域名数组
//
// 参数:
//   - value: 逗号分隔的域名字符串，如 "temp.mail,example.com"
//
// 返回值:
//   - []string: 解析后的小写域名数组
func parseDomains(value string) []string {
	out := parseList(value)
	for i := range out {
		out[i] = strings.ToLower(out[i])
	}
	return out
}

// parseList 将逗号分隔的字符串解析为字符串切片
//
// 参数:
//   - value: 逗号分隔的字符串，如 "item1,item2,item3"
//
// 返回值:
//   - []string: 解析后的字符串切片，已去除空白字符
func parseList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

// loadEnvFile 尝试加载 .env 文件
//
// 加载顺序：
//   1. 当前目录的 .env
//   2. 父目录的 .env（用于从 backend/ 子目录运行的情况）
//
// 注意：
//   - 如果文件不存在，静默失败（.env 是可选的）
//   - 环境变量不会被覆盖（已存在的环境变量优先级更高）
func loadEnvFile() {
	// 尝试当前目录的 .env
	if err := godotenv.Load(".env"); err == nil {
		return
	}

	// 尝试父目录的 .env（从 backend/ 目录运行时）
	parentEnv := filepath.Join("..", ".env")
	if _, err := os.Stat(parentEnv); err == nil {
		_ = godotenv.Load(parentEnv)
	}
}

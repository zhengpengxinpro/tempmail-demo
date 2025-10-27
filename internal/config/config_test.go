package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// 保存原始环境变量
	originalEnvs := make(map[string]string)
	envKeys := []string{
		"TEMPMAIL_JWT_SECRET",
		"TEMPMAIL_SERVER_HOST",
		"TEMPMAIL_SERVER_PORT",
		"TEMPMAIL_MAILBOX_ALLOWED_DOMAINS",
		"TEMPMAIL_MAILBOX_DEFAULT_TTL",
		"TEMPMAIL_SMTP_BIND_ADDR",
		"TEMPMAIL_SMTP_DOMAIN",
		"TEMPMAIL_LOG_LEVEL",
		"TEMPMAIL_LOG_DEVELOPMENT",
	}

	for _, key := range envKeys {
		originalEnvs[key] = os.Getenv(key)
	}

	// 测试后恢复环境变量
	defer func() {
		for key, value := range originalEnvs {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("加载默认配置成功", func(t *testing.T) {
		// 清除所有环境变量
		for _, key := range envKeys {
			os.Unsetenv(key)
		}

		// 设置必需的JWT密钥
		os.Setenv("TEMPMAIL_JWT_SECRET", "test-secret-key-for-development-32-chars-long-at-least")

		cfg, err := Load()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 验证默认值
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, []string{"temp.mail"}, cfg.Mailbox.AllowedDomains)
		assert.Equal(t, time.Hour, cfg.Mailbox.DefaultTTL)
		assert.Equal(t, 3, cfg.Mailbox.MaxPerIP)
		assert.Equal(t, ":25", cfg.SMTP.BindAddr)
		assert.Equal(t, "temp.mail", cfg.SMTP.Domain)
		assert.Equal(t, []string{"*"}, cfg.CORS.AllowedOrigins)
		assert.Equal(t, "info", cfg.Log.Level)
		assert.False(t, cfg.Log.Development)
		assert.Equal(t, "test-secret-key-for-development-32-chars-long-at-least", cfg.JWT.Secret)
		assert.Equal(t, "tempmail", cfg.JWT.Issuer)
		assert.Equal(t, 15*time.Minute, cfg.JWT.AccessExpiry)
		assert.Equal(t, 7*24*time.Hour, cfg.JWT.RefreshExpiry)
	})

	t.Run("加载自定义配置成功", func(t *testing.T) {
		// 设置自定义环境变量
		os.Setenv("TEMPMAIL_JWT_SECRET", "custom-jwt-secret-key-32-chars-long-minimum")
		os.Setenv("TEMPMAIL_SERVER_HOST", "127.0.0.1")
		os.Setenv("TEMPMAIL_SERVER_PORT", "9090")
		os.Setenv("TEMPMAIL_MAILBOX_ALLOWED_DOMAINS", "custom.mail,test.dev")
		os.Setenv("TEMPMAIL_MAILBOX_DEFAULT_TTL", "2h")
		os.Setenv("TEMPMAIL_MAILBOX_MAX_PER_IP", "5")
		os.Setenv("TEMPMAIL_SMTP_BIND_ADDR", ":587")
		os.Setenv("TEMPMAIL_SMTP_DOMAIN", "custom.mail")
		os.Setenv("TEMPMAIL_CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
		os.Setenv("TEMPMAIL_LOG_LEVEL", "debug")
		os.Setenv("TEMPMAIL_LOG_DEVELOPMENT", "true")

		cfg, err := Load()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 验证自定义值
		assert.Equal(t, "127.0.0.1", cfg.Server.Host)
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, []string{"custom.mail", "test.dev"}, cfg.Mailbox.AllowedDomains)
		assert.Equal(t, 2*time.Hour, cfg.Mailbox.DefaultTTL)
		assert.Equal(t, 5, cfg.Mailbox.MaxPerIP)
		assert.Equal(t, ":587", cfg.SMTP.BindAddr)
		assert.Equal(t, "custom.mail", cfg.SMTP.Domain)
		assert.Equal(t, []string{"http://localhost:3000", "http://localhost:5173"}, cfg.CORS.AllowedOrigins)
		assert.Equal(t, "debug", cfg.Log.Level)
		assert.True(t, cfg.Log.Development)
		assert.Equal(t, "custom-jwt-secret-key-32-chars-long-minimum", cfg.JWT.Secret)
	})

	t.Run("JWT密钥太短失败", func(t *testing.T) {
		os.Setenv("TEMPMAIL_JWT_SECRET", "short-key") // 少于32字符

		cfg, err := Load()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "JWT secret must be at least 32 characters long")
	})

	t.Run("使用默认JWT密钥失败", func(t *testing.T) {
		os.Setenv("TEMPMAIL_JWT_SECRET", "change-me-in-production")

		cfg, err := Load()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "JWT secret cannot be the default value")
	})

	t.Run("无效的TTL格式失败", func(t *testing.T) {
		os.Setenv("TEMPMAIL_JWT_SECRET", "valid-jwt-secret-key-32-chars-long-minimum")
		os.Setenv("TEMPMAIL_MAILBOX_DEFAULT_TTL", "invalid-duration")

		cfg, err := Load()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "invalid mailbox.default_ttl")
	})

	t.Run("空的允许域名失败", func(t *testing.T) {
		// 清除所有环境变量
		for _, key := range envKeys {
			os.Unsetenv(key)
		}
		
		os.Setenv("TEMPMAIL_JWT_SECRET", "valid-jwt-secret-key-32-chars-long-minimum")
		os.Setenv("TEMPMAIL_MAILBOX_ALLOWED_DOMAINS", " , , ") // 只有空格和逗号

		cfg, err := Load()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "mailbox.allowed_domains must not be empty")
	})
}

func TestParseDomains(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "单个域名",
			input:    "temp.mail",
			expected: []string{"temp.mail"},
		},
		{
			name:     "多个域名",
			input:    "temp.mail,test.com,example.org",
			expected: []string{"temp.mail", "test.com", "example.org"},
		},
		{
			name:     "带空格的域名",
			input:    " temp.mail , test.com , example.org ",
			expected: []string{"temp.mail", "test.com", "example.org"},
		},
		{
			name:     "大写域名转小写",
			input:    "TEMP.MAIL,Test.Com",
			expected: []string{"temp.mail", "test.com"},
		},
		{
			name:     "空字符串",
			input:    "",
			expected: []string{},
		},
		{
			name:     "只有逗号",
			input:    ",,,",
			expected: []string{},
		},
		{
			name:     "混合空值",
			input:    "temp.mail,,test.com,",
			expected: []string{"temp.mail", "test.com"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseDomains(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseList(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "单个项目",
			input:    "item1",
			expected: []string{"item1"},
		},
		{
			name:     "多个项目",
			input:    "item1,item2,item3",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "带空格的项目",
			input:    " item1 , item2 , item3 ",
			expected: []string{"item1", "item2", "item3"},
		},
		{
			name:     "空字符串",
			input:    "",
			expected: []string{},
		},
		{
			name:     "只有逗号",
			input:    ",,,",
			expected: []string{},
		},
		{
			name:     "混合空值",
			input:    "item1,,item2,",
			expected: []string{"item1", "item2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseList(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDatabaseConfig(t *testing.T) {
	// 保存原始环境变量
	originalEnvs := make(map[string]string)
	envKeys := []string{
		"TEMPMAIL_JWT_SECRET",
		"TEMPMAIL_DATABASE_DSN",
		"TEMPMAIL_DATABASE_MAX_OPEN_CONNS",
		"TEMPMAIL_DATABASE_MAX_IDLE_CONNS",
		"TEMPMAIL_DATABASE_CONN_MAX_LIFETIME",
		"TEMPMAIL_REDIS_ADDRESS",
		"TEMPMAIL_REDIS_PASSWORD",
		"TEMPMAIL_REDIS_DB",
	}

	for _, key := range envKeys {
		originalEnvs[key] = os.Getenv(key)
	}

	defer func() {
		for key, value := range originalEnvs {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("数据库配置加载成功", func(t *testing.T) {
		os.Setenv("TEMPMAIL_JWT_SECRET", "valid-jwt-secret-key-32-chars-long-minimum")
		os.Setenv("TEMPMAIL_DATABASE_DSN", "postgres://user:pass@localhost:5432/testdb")
		os.Setenv("TEMPMAIL_DATABASE_MAX_OPEN_CONNS", "50")
		os.Setenv("TEMPMAIL_DATABASE_MAX_IDLE_CONNS", "10")
		os.Setenv("TEMPMAIL_DATABASE_CONN_MAX_LIFETIME", "10m")
		os.Setenv("TEMPMAIL_REDIS_ADDRESS", "localhost:6379")
		os.Setenv("TEMPMAIL_REDIS_PASSWORD", "redis-password")
		os.Setenv("TEMPMAIL_REDIS_DB", "1")

		cfg, err := Load()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "postgres://user:pass@localhost:5432/testdb", cfg.Database.DSN)
		assert.Equal(t, 50, cfg.Database.MaxOpenConns)
		assert.Equal(t, 10, cfg.Database.MaxIdleConns)
		assert.Equal(t, 10*time.Minute, cfg.Database.ConnMaxLifetime)
		assert.Equal(t, "localhost:6379", cfg.Redis.Address)
		assert.Equal(t, "redis-password", cfg.Redis.Password)
		assert.Equal(t, 1, cfg.Redis.DB)
	})
}
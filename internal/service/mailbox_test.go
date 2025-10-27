package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage/memory"
)

// MockStore 模拟存储接口
type MockStore struct {
	mock.Mock
}

func (m *MockStore) SaveMailbox(mailbox *domain.Mailbox) error {
	args := m.Called(mailbox)
	return args.Error(0)
}

func (m *MockStore) GetMailbox(id string) (*domain.Mailbox, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Mailbox), args.Error(1)
}

func (m *MockStore) GetMailboxByAddress(address string) (*domain.Mailbox, error) {
	args := m.Called(address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Mailbox), args.Error(1)
}

func (m *MockStore) ListMailboxes() []domain.Mailbox {
	args := m.Called()
	return args.Get(0).([]domain.Mailbox)
}

func (m *MockStore) DeleteMailbox(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStore) ListMailboxesByUserID(userID string) []domain.Mailbox {
	args := m.Called(userID)
	return args.Get(0).([]domain.Mailbox)
}

func (m *MockStore) DeleteExpiredMailboxes() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// 实现其他必需的接口方法（简化版）
func (m *MockStore) SaveMessage(message *domain.Message) error { return nil }
func (m *MockStore) ListMessages(mailboxID string) ([]domain.Message, error) { return nil, nil }
func (m *MockStore) GetMessage(mailboxID, messageID string) (*domain.Message, error) { return nil, nil }
func (m *MockStore) MarkMessageRead(mailboxID, messageID string) error { return nil }
func (m *MockStore) CreateUser(user *domain.User) error { return nil }
func (m *MockStore) GetUserByID(id string) (*domain.User, error) { return nil, nil }
func (m *MockStore) GetUserByEmail(email string) (*domain.User, error) { return nil, nil }
func (m *MockStore) UpdateUser(user *domain.User) error { return nil }
func (m *MockStore) UpdateLastLogin(userID string) error { return nil }
func (m *MockStore) GetUserByAPIKey(apiKey string) (*domain.User, error) { return nil, nil }
func (m *MockStore) SaveAPIKey(apiKey *domain.APIKey) error { return nil }
func (m *MockStore) GetAPIKey(id string) (*domain.APIKey, error) { return nil, nil }
func (m *MockStore) GetAPIKeyByKey(key string) (*domain.APIKey, error) { return nil, nil }
func (m *MockStore) ListAPIKeysByUserID(userID string) ([]*domain.APIKey, error) { return nil, nil }
func (m *MockStore) DeleteAPIKey(id string) error { return nil }
func (m *MockStore) UpdateAPIKeyLastUsed(id string) error { return nil }
func (m *MockStore) ListUsers(page, pageSize int, search string, role *domain.UserRole, tier *domain.UserTier, isActive *bool) ([]domain.User, int, error) { return nil, 0, nil }
func (m *MockStore) DeleteUser(userID string) error { return nil }
func (m *MockStore) DeleteMailboxesByUserID(userID string) error { return nil }
func (m *MockStore) GetSystemStatistics() (*domain.SystemStatistics, error) { return nil, nil }
func (m *MockStore) GetDomainStatistics(domain string) (int, int, error) { return 0, 0, nil }
func (m *MockStore) SaveAlias(alias *domain.MailboxAlias) error { return nil }
func (m *MockStore) GetAlias(aliasID string) (*domain.MailboxAlias, error) { return nil, nil }
func (m *MockStore) GetAliasByAddress(address string) (*domain.MailboxAlias, error) { return nil, nil }
func (m *MockStore) ListAliasesByMailboxID(mailboxID string) ([]*domain.MailboxAlias, error) { return nil, nil }
func (m *MockStore) DeleteAlias(aliasID string) error { return nil }
func (m *MockStore) AddToBlacklist(jti string, ttl time.Duration) error { return nil }
func (m *MockStore) IsBlacklisted(jti string) (bool, error) { return false, nil }
func (m *MockStore) IncrementRateLimit(key string, window time.Duration) (int64, error) { return 0, nil }
func (m *MockStore) GetRateLimit(key string) (int64, error) { return 0, nil }
func (m *MockStore) CacheSession(sessionID string, userID string, ttl time.Duration) error { return nil }
func (m *MockStore) GetCachedSession(sessionID string) (string, error) { return "", nil }
func (m *MockStore) DeleteCachedSession(sessionID string) error { return nil }
func (m *MockStore) PublishNewMail(mailboxID string, message *domain.Message) error { return nil }
func (m *MockStore) SubscribeNewMail(mailboxID string) interface{} { return nil }
func (m *MockStore) Close() error { return nil }
func (m *MockStore) Health() error { return nil }

func TestMailboxService_CreateRandomMailbox(t *testing.T) {
	// 使用内存存储进行测试
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.Config{
		Mailbox: config.MailboxConfig{
			AllowedDomains: []string{"temp.mail", "test.com"},
			DefaultTTL:     24 * time.Hour,
			MaxPerIP:       3,
		},
	}

	service := NewMailboxService(store, store, cfg)

	t.Run("创建随机邮箱成功", func(t *testing.T) {
		input := CreateMailboxInput{
			IPSource: "192.168.1.1",
		}

		mailbox, err := service.Create(input)

		assert.NoError(t, err)
		assert.NotNil(t, mailbox)
		assert.NotEmpty(t, mailbox.ID)
		assert.NotEmpty(t, mailbox.Address)
		assert.NotEmpty(t, mailbox.Token)
		assert.Contains(t, cfg.Mailbox.AllowedDomains, mailbox.Domain)
		assert.Equal(t, "192.168.1.1", mailbox.IPSource)
		assert.Equal(t, 0, mailbox.TotalCount)
		assert.Equal(t, 0, mailbox.Unread)
	})

	t.Run("创建自定义前缀邮箱成功", func(t *testing.T) {
		input := CreateMailboxInput{
			Prefix:   "custom",
			Domain:   "temp.mail",
			IPSource: "192.168.1.1",
		}

		mailbox, err := service.Create(input)

		assert.NoError(t, err)
		assert.NotNil(t, mailbox)
		assert.Equal(t, "custom@temp.mail", mailbox.Address)
		assert.Equal(t, "custom", mailbox.LocalPart)
		assert.Equal(t, "temp.mail", mailbox.Domain)
	})

	t.Run("使用不允许的域名创建邮箱失败", func(t *testing.T) {
		input := CreateMailboxInput{
			Prefix:   "test",
			Domain:   "invalid.com",
			IPSource: "192.168.1.1",
		}

		mailbox, err := service.Create(input)

		assert.Error(t, err)
		assert.Nil(t, mailbox)
		assert.Equal(t, ErrDomainNotAllowed, err)
	})

	t.Run("使用无效前缀创建邮箱失败", func(t *testing.T) {
		input := CreateMailboxInput{
			Prefix:   "a", // 太短
			Domain:   "temp.mail",
			IPSource: "192.168.1.1",
		}

		mailbox, err := service.Create(input)

		assert.Error(t, err)
		assert.Nil(t, mailbox)
		assert.Equal(t, ErrPrefixInvalid, err)
	})
}

func TestMailboxService_GetMailbox(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.Config{
		Mailbox: config.MailboxConfig{
			AllowedDomains: []string{"temp.mail"},
			DefaultTTL:     24 * time.Hour,
		},
	}

	service := NewMailboxService(store, store, cfg)

	// 先创建一个邮箱
	input := CreateMailboxInput{
		IPSource: "192.168.1.1",
	}
	createdMailbox, err := service.Create(input)
	assert.NoError(t, err)

	t.Run("根据ID获取邮箱成功", func(t *testing.T) {
		mailbox, err := service.Get(createdMailbox.ID)

		assert.NoError(t, err)
		assert.NotNil(t, mailbox)
		assert.Equal(t, createdMailbox.ID, mailbox.ID)
		assert.Equal(t, createdMailbox.Address, mailbox.Address)
	})

	t.Run("根据地址获取邮箱成功", func(t *testing.T) {
		mailbox, err := service.GetByAddress(createdMailbox.Address)

		assert.NoError(t, err)
		assert.NotNil(t, mailbox)
		assert.Equal(t, createdMailbox.ID, mailbox.ID)
		assert.Equal(t, createdMailbox.Address, mailbox.Address)
	})

	t.Run("获取不存在的邮箱失败", func(t *testing.T) {
		mailbox, err := service.Get("nonexistent")

		assert.Error(t, err)
		assert.Nil(t, mailbox)
	})
}

func TestMailboxService_DeleteMailbox(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.Config{
		Mailbox: config.MailboxConfig{
			AllowedDomains: []string{"temp.mail"},
			DefaultTTL:     24 * time.Hour,
		},
	}

	service := NewMailboxService(store, store, cfg)

	// 先创建一个邮箱
	input := CreateMailboxInput{
		IPSource: "192.168.1.1",
	}
	createdMailbox, err := service.Create(input)
	assert.NoError(t, err)

	t.Run("删除邮箱成功", func(t *testing.T) {
		err := service.Delete(createdMailbox.ID)

		assert.NoError(t, err)

		// 验证邮箱已被删除
		mailbox, err := service.Get(createdMailbox.ID)
		assert.Error(t, err)
		assert.Nil(t, mailbox)
	})

	t.Run("删除不存在的邮箱失败", func(t *testing.T) {
		err := service.Delete("nonexistent")

		assert.Error(t, err)
	})
}

func TestMailboxService_CreateWithDifferentInputs(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.Config{
		Mailbox: config.MailboxConfig{
			AllowedDomains: []string{"temp.mail", "test.com"},
			DefaultTTL:     24 * time.Hour,
		},
	}

	service := NewMailboxService(store, store, cfg)

	t.Run("创建邮箱时自动选择域名", func(t *testing.T) {
		input := CreateMailboxInput{
			IPSource: "192.168.1.1",
		}

		mailbox, err := service.Create(input)

		assert.NoError(t, err)
		assert.NotNil(t, mailbox)
		assert.Contains(t, cfg.Mailbox.AllowedDomains, mailbox.Domain)
	})

	t.Run("创建多个邮箱应该有不同的地址", func(t *testing.T) {
		input1 := CreateMailboxInput{IPSource: "192.168.1.1"}
		input2 := CreateMailboxInput{IPSource: "192.168.1.2"}

		mailbox1, err1 := service.Create(input1)
		mailbox2, err2 := service.Create(input2)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, mailbox1.Address, mailbox2.Address)
	})
}
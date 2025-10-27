package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage"
)

var (
	ErrAPIKeyNotFound = errors.New("API key not found")
	ErrAPIKeyInvalid  = errors.New("invalid API key")
)

// APIKeyService API Key业务逻辑服务
type APIKeyService struct {
	store storage.Store
}

// NewAPIKeyService 创建API Key服务
func NewAPIKeyService(store storage.Store) *APIKeyService {
	return &APIKeyService{
		store: store,
	}
}

// CreateAPIKeyInput 创建API Key的输入参数
type CreateAPIKeyInput struct {
	UserID    string
	Name      string
	ExpiresIn *time.Duration // 过期时间（可选）
}

// CreateAPIKey 创建新的API Key
//
// 参数:
//   - input: 创建参数
//
// 返回值:
//   - *domain.APIKey: 创建的API Key
//   - error: 错误信息
func (s *APIKeyService) CreateAPIKey(input CreateAPIKeyInput) (*domain.APIKey, error) {
	// 验证用户是否存在
	_, err := s.store.GetUserByID(input.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 生成随机API Key
	key, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	// 计算过期时间
	var expiresAt *time.Time
	if input.ExpiresIn != nil {
		t := time.Now().Add(*input.ExpiresIn)
		expiresAt = &t
	}

	// 生成密钥前缀（前8个字符）
	keyPrefix := key
	if len(key) > 8 {
		keyPrefix = key[:8]
	}

	apiKey := &domain.APIKey{
		ID:        uuid.New().String(),
		UserID:    input.UserID,
		Key:       key,
		KeyPrefix: keyPrefix,
		Name:      input.Name,
		IsActive:  true,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	if err := s.store.SaveAPIKey(apiKey); err != nil {
		return nil, err
	}

	return apiKey, nil
}

// ListAPIKeys 列出用户的所有API Key
//
// 参数:
//   - userID: 用户ID
//
// 返回值:
//   - []*domain.APIKey: API Key列表
//   - error: 错误信息
func (s *APIKeyService) ListAPIKeys(userID string) ([]*domain.APIKey, error) {
	return s.store.ListAPIKeysByUserID(userID)
}

// GetAPIKey 获取API Key详情
//
// 参数:
//   - id: API Key ID
//
// 返回值:
//   - *domain.APIKey: API Key详情
//   - error: 错误信息
func (s *APIKeyService) GetAPIKey(id string) (*domain.APIKey, error) {
	apiKey, err := s.store.GetAPIKey(id)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}
	return apiKey, nil
}

// DeleteAPIKey 删除API Key
//
// 参数:
//   - userID: 用户ID（用于权限验证）
//   - id: API Key ID
//
// 返回值:
//   - error: 错误信息
func (s *APIKeyService) DeleteAPIKey(userID, id string) error {
	// 获取API Key
	apiKey, err := s.store.GetAPIKey(id)
	if err != nil {
		return ErrAPIKeyNotFound
	}

	// 验证所有权
	if apiKey.UserID != userID {
		return errors.New("permission denied")
	}

	return s.store.DeleteAPIKey(id)
}

// ValidateAPIKey 验证API Key并返回关联的用户
//
// 参数:
//   - key: API Key字符串
//
// 返回值:
//   - *domain.User: 关联的用户
//   - error: 错误信息
func (s *APIKeyService) ValidateAPIKey(key string) (*domain.User, error) {
	// 获取API Key
	apiKey, err := s.store.GetAPIKeyByKey(key)
	if err != nil {
		return nil, ErrAPIKeyInvalid
	}

	// 检查是否激活
	if !apiKey.IsActive {
		return nil, ErrAPIKeyInvalid
	}

	// 检查是否过期
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, errors.New("API key expired")
	}

	// 更新最后使用时间
	_ = s.store.UpdateAPIKeyLastUsed(apiKey.ID)

	// 获取用户信息
	user, err := s.store.GetUserByID(apiKey.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 检查用户是否激活
	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}

	return user, nil
}

// generateAPIKey 生成一个安全的随机API Key
//
// 返回值:
//   - string: 生成的API Key（48字符）
//   - error: 错误信息
func generateAPIKey() (string, error) {
	// 生成32字节的随机数据
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 使用base64编码并截取前48个字符
	key := base64.URLEncoding.EncodeToString(bytes)
	if len(key) > 48 {
		key = key[:48]
	}

	return key, nil
}

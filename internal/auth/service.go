package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"tempmail/backend/internal/domain"
)

var (
	// ErrInvalidEmail 无效的邮箱格式
	ErrInvalidEmail = errors.New("invalid email format")
	// ErrInvalidPassword 无效的密码
	ErrInvalidPassword = errors.New("invalid password")
	// ErrEmailExists 邮箱已存在
	ErrEmailExists = errors.New("email already exists")
	// ErrUsernameExists 用户名已存在
	ErrUsernameExists = errors.New("username already exists")
	// ErrUserNotFound 用户不存在
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidCredentials 凭证无效
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserInactive 用户已被禁用
	ErrUserInactive = errors.New("user is inactive")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Service 认证服务
type Service struct {
	userRepo UserRepository
}

// UserRepository 用户存储接口
type UserRepository interface {
	CreateUser(user *domain.User) error
	GetUserByID(id string) (*domain.User, error)
	GetUserByEmail(email string) (*domain.User, error)
	GetUserByUsername(username string) (*domain.User, error)
	UpdateUser(user *domain.User) error
	UpdateLastLogin(userID string) error
	GetUserByAPIKey(apiKey string) (*domain.User, error)
}

// NewService 创建认证服务
func NewService(userRepo UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
	}
}

// RegisterInput 注册输入
type RegisterInput struct {
	Email    string
	Password string
	Username string
}

// LoginInput 登录输入
type LoginInput struct {
	Identifier string
	Password   string
}

// Register 用户注册
func (s *Service) Register(input RegisterInput) (*domain.User, error) {
	// 验证邮箱格式
	if !ValidateEmail(input.Email) {
		return nil, ErrInvalidEmail
	}

	// 验证密码强度
	if err := ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	// 检查邮箱是否已存在
	if user, err := s.userRepo.GetUserByEmail(strings.ToLower(input.Email)); err == nil && user != nil {
		return nil, ErrEmailExists
	}

	// 检查用户名是否已存在
	if user, err := s.userRepo.GetUserByUsername(strings.ToLower(input.Username)); err == nil && user != nil {
		return nil, ErrUsernameExists
	}

	// 哈希密码
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 创建用户
	now := time.Now()
	user := &domain.User{
		ID:              uuid.New().String(),
		Email:           strings.ToLower(input.Email),
		Username:        input.Username,
		PasswordHash:    passwordHash,
		Role:            domain.RoleUser, // 默认为普通用户
		Tier:            domain.TierFree,
		IsActive:        true,
		IsEmailVerified: false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login 用户登录
func (s *Service) Login(input LoginInput) (*domain.User, error) {
	identifier := strings.ToLower(input.Identifier)

	// 优先按邮箱查找
	user, err := s.userRepo.GetUserByEmail(identifier)
	if err != nil {
		// 如果按邮箱查找失败，尝试按用户名查找
		user, err = s.userRepo.GetUserByUsername(identifier)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	// 检查用户是否激活
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 验证密码
	if !CheckPassword(input.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// 更新最后登录时间
	_ = s.userRepo.UpdateLastLogin(user.ID)

	return user, nil
}

// GetUserByID 根据 ID 获取用户
func (s *Service) GetUserByID(userID string) (*domain.User, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// ChangePassword 修改密码
func (s *Service) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if !CheckPassword(oldPassword, user.PasswordHash) {
		return errors.New("invalid old password")
	}

	// 验证新密码强度
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	// 哈希新密码
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = newHash
	return s.userRepo.UpdateUser(user)
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 72 {
		return errors.New("password must be at most 72 characters")
	}
	return nil
}

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 检查密码是否匹配
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidateAPIKey 验证API Key并返回用户
func (s *Service) ValidateAPIKey(apiKey string) (*domain.User, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	// 从存储中获取API Key对应的用户
	user, err := s.userRepo.GetUserByAPIKey(apiKey)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查用户是否激活
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	return user, nil
}

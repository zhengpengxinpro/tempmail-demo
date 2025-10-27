package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"tempmail/backend/internal/auth"
	"tempmail/backend/internal/domain"
)

var (
	// ErrUnauthorized 未授权访问
	ErrUnauthorized = errors.New("unauthorized access")
	// ErrInsufficientPermission 权限不足
	ErrInsufficientPermission = errors.New("insufficient permissions")
	// ErrUserNotFound 用户不存在
	ErrAdminUserNotFound = errors.New("user not found")
	// ErrCannotModifySelf 不能修改自己
	ErrCannotModifySelf = errors.New("cannot modify self")
	// ErrCannotModifySuper 不能修改超级管理员
	ErrCannotModifySuper = errors.New("cannot modify super admin")
)

// AdminService 管理服务
type AdminService struct {
	store  domain.Store
	config *domain.Config
}

// NewAdminService 创建管理服务
func NewAdminService(store domain.Store, config *domain.Config) *AdminService {
	return &AdminService{
		store:  store,
		config: config,
	}
}

// ListUsersInput 列出用户的输入参数
type ListUsersInput struct {
	Page     int
	PageSize int
	Search   string // 搜索关键词（邮箱/用户名）
	Role     *domain.UserRole
	Tier     *domain.UserTier
	IsActive *bool
}

// ListUsersOutput 列出用户的输出结果
type ListUsersOutput struct {
	Users      []domain.User `json:"users"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
}

// ListUsers 列出所有用户（需要管理员权限）
func (s *AdminService) ListUsers(input ListUsersInput) (*ListUsersOutput, error) {
	// 设置默认分页
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}

	// 获取用户列表
	users, total, err := s.store.ListUsers(input.Page, input.PageSize, input.Search, input.Role, input.Tier, input.IsActive)
	if err != nil {
		return nil, err
	}

	totalPages := (total + input.PageSize - 1) / input.PageSize

	return &ListUsersOutput{
		Users:      users,
		Total:      total,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUser 获取用户详情（需要管理员权限）
func (s *AdminService) GetUser(userID string) (*domain.User, error) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		return nil, ErrAdminUserNotFound
	}
	return user, nil
}

// UpdateUserInput 更新用户的输入参数
type UpdateUserInput struct {
	UserID          string
	Role            *domain.UserRole
	Tier            *domain.UserTier
	IsActive        *bool
	IsEmailVerified *bool
	OperatorID      string // 操作者ID
}

// UpdateUser 更新用户信息（需要管理员权限）
func (s *AdminService) UpdateUser(input UpdateUserInput) (*domain.User, error) {
	// 不能修改自己
	if input.UserID == input.OperatorID {
		return nil, ErrCannotModifySelf
	}

	// 获取目标用户
	user, err := s.store.GetUserByID(input.UserID)
	if err != nil {
		return nil, ErrAdminUserNotFound
	}

	// 获取操作者
	operator, err := s.store.GetUserByID(input.OperatorID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// 不能修改超级管理员（除非自己也是超级管理员）
	if user.Role == domain.RoleSuper && operator.Role != domain.RoleSuper {
		return nil, ErrCannotModifySuper
	}

	// 更新字段
	if input.Role != nil {
		// 只有超级管理员才能设置角色
		if operator.Role != domain.RoleSuper {
			return nil, ErrInsufficientPermission
		}
		user.Role = *input.Role
	}

	if input.Tier != nil {
		user.Tier = *input.Tier
	}

	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}

	if input.IsEmailVerified != nil {
		user.IsEmailVerified = *input.IsEmailVerified
	}

	user.UpdatedAt = time.Now()

	// 保存更新
	if err := s.store.UpdateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser 删除用户（需要超级管理员权限）
func (s *AdminService) DeleteUser(userID, operatorID string) error {
	// 不能删除自己
	if userID == operatorID {
		return ErrCannotModifySelf
	}

	// 获取目标用户
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		return ErrAdminUserNotFound
	}

	// 不能删除超级管理员
	if user.Role == domain.RoleSuper {
		return ErrCannotModifySuper
	}

	// 删除用户的邮箱
	if err := s.store.DeleteMailboxesByUserID(userID); err != nil {
		return err
	}

	// 删除用户
	return s.store.DeleteUser(userID)
}

// GetStatistics 获取系统统计（需要管理员权限）
func (s *AdminService) GetStatistics() (*domain.SystemStatistics, error) {
	stats, err := s.store.GetSystemStatistics()
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// CreateAdminUserInput 创建管理员用户的输入参数
type CreateAdminUserInput struct {
	Email    string
	Password string
	Username string
	Role     domain.UserRole
}

// CreateAdminUser 创建管理员用户（仅用于开发测试）
func (s *AdminService) CreateAdminUser(input CreateAdminUserInput) (*domain.User, error) {
	// 验证邮箱格式
	if !auth.ValidateEmail(input.Email) {
		return nil, auth.ErrInvalidEmail
	}

	// 验证密码强度
	if err := auth.ValidatePassword(input.Password); err != nil {
		return nil, auth.ErrInvalidPassword
	}

	// 检查邮箱是否已存在
	if _, err := s.store.GetUserByEmail(input.Email); err == nil {
		return nil, auth.ErrEmailExists
	}

	// 创建用户实体
	user := &domain.User{
		ID:       uuid.New().String(),
		Email:    input.Email,
		Username: input.Username,
		Role:     input.Role,
		Tier:     domain.TierFree, // 默认免费套餐
		IsActive: true,
		IsEmailVerified: true, // 管理员默认邮箱已验证
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 加密密码
	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = hashedPassword

	// 保存用户
	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

// ListDomainsInput 列出域名的输入参数
type ListDomainsInput struct {
	Page     int
	PageSize int
	Search   string
}

// ListDomainsOutput 列出域名的输出结果
type ListDomainsOutput struct {
	Domains    []DomainInfo `json:"domains"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalPages int          `json:"totalPages"`
}

// DomainInfo 域名信息
type DomainInfo struct {
	ID           string    `json:"id"`
	Domain       string    `json:"domain"`
	IsActive     bool      `json:"isActive"`
	IsDefault    bool      `json:"isDefault"`
	MailboxCount int       `json:"mailboxCount"`
	MessageCount int       `json:"messageCount"`
	CreatedAt    time.Time `json:"createdAt"`
	LastUsedAt   time.Time `json:"lastUsedAt"`
}

// ListDomains 列出所有域名（需要管理员权限）
func (s *AdminService) ListDomains(input ListDomainsInput) (*ListDomainsOutput, error) {
	// 设置默认分页
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = 20
	}

	// 从配置获取域名列表
	allowedDomains := s.config.AllowedDomains
	domains := make([]DomainInfo, 0, len(allowedDomains))

	for i, domain := range allowedDomains {
		// 获取域名的统计信息
		mailboxCount, messageCount, err := s.store.GetDomainStatistics(domain)
		if err != nil {
			continue
		}

		domains = append(domains, DomainInfo{
			ID:           domain, // 使用域名作为ID
			Domain:       domain,
			IsActive:     true,
			IsDefault:    i == 0, // 第一个域名为默认域名
			MailboxCount: mailboxCount,
			MessageCount: messageCount,
			CreatedAt:    time.Now(), // TODO: 需要存储域名创建时间
			LastUsedAt:   time.Now(), // TODO: 需要存储最后使用时间
		})
	}

	// 简单的分页处理
	start := (input.Page - 1) * input.PageSize
	end := start + input.PageSize
	if end > len(domains) {
		end = len(domains)
	}
	if start > len(domains) {
		start = len(domains)
	}

	pagedDomains := domains[start:end]
	totalPages := (len(domains) + input.PageSize - 1) / input.PageSize

	return &ListDomainsOutput{
		Domains:    pagedDomains,
		Total:      len(domains),
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// AddDomain 添加域名（需要超级管理员权限）
func (s *AdminService) AddDomain(domain string) error {
	// 检查域名是否已存在
	for _, d := range s.config.AllowedDomains {
		if d == domain {
			return errors.New("domain already exists")
		}
	}

	// 添加到配置
	s.config.AllowedDomains = append(s.config.AllowedDomains, domain)

	// TODO: 持久化配置更改
	return nil
}

// RemoveDomain 删除域名（需要超级管理员权限）
func (s *AdminService) RemoveDomain(domain string) error {
	// 不能删除最后一个域名
	if len(s.config.AllowedDomains) <= 1 {
		return errors.New("cannot remove last domain")
	}

	// 从配置中移除
	newDomains := make([]string, 0, len(s.config.AllowedDomains)-1)
	found := false
	for _, d := range s.config.AllowedDomains {
		if d != domain {
			newDomains = append(newDomains, d)
		} else {
			found = true
		}
	}

	if !found {
		return errors.New("domain not found")
	}

	s.config.AllowedDomains = newDomains

	// TODO: 持久化配置更改
	return nil
}

// GetUserQuota 获取用户配额（需要管理员权限）
func (s *AdminService) GetUserQuota(userID string) (*domain.Quota, error) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		return nil, ErrAdminUserNotFound
	}

	quota := domain.DefaultQuotas(user.Tier)
	quota.UserID = userID
	return &quota, nil
}

// UpdateUserQuota 更新用户配额（需要管理员权限）
func (s *AdminService) UpdateUserQuota(userID string, quota domain.Quota) error {
	// 验证用户存在
	_, err := s.store.GetUserByID(userID)
	if err != nil {
		return ErrAdminUserNotFound
	}

	// TODO: 实现自定义配额存储
	return nil
}

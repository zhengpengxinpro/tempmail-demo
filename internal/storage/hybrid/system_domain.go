package hybrid

import (
	"fmt"
	"time"

	"tempmail/backend/internal/domain"
)

// ========== System Domain Repository ==========

// SaveSystemDomain 保存系统域名
func (s *Store) SaveSystemDomain(sysDomain *domain.SystemDomain) error {
	// 保存到 PostgreSQL
	if err := s.postgres.SaveSystemDomain(sysDomain); err != nil {
		return err
	}

	// 缓存到 Redis（1小时过期）
	s.redis.CacheSystemDomain(sysDomain, 1*time.Hour)

	// 删除系统域名列表缓存（因为列表已变化）
	s.redis.Delete("system_domains:list")

	return nil
}

// GetSystemDomain 根据 ID 获取系统域名
func (s *Store) GetSystemDomain(domainID string) (*domain.SystemDomain, error) {
	// 先尝试从 Redis 获取
	if sysDomain, err := s.redis.GetCachedSystemDomain(domainID); err == nil {
		return sysDomain, nil
	}

	// 从 PostgreSQL 获取
	sysDomain, err := s.postgres.GetSystemDomain(domainID)
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis
	s.redis.CacheSystemDomain(sysDomain, 1*time.Hour)

	return sysDomain, nil
}

// GetSystemDomainByDomain 根据域名获取系统域名
func (s *Store) GetSystemDomainByDomain(domainName string) (*domain.SystemDomain, error) {
	// 域名查询直接从 PostgreSQL 获取（不缓存，因为查询频繁且变化多）
	return s.postgres.GetSystemDomainByDomain(domainName)
}

// ListSystemDomains 获取所有系统域名
func (s *Store) ListSystemDomains() ([]*domain.SystemDomain, error) {
	// 先尝试从 Redis 获取
	if sysDomains, err := s.redis.GetCachedSystemDomainList(); err == nil {
		return sysDomains, nil
	}

	// 从 PostgreSQL 获取
	sysDomains, err := s.postgres.ListSystemDomains()
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis（30分钟过期）
	s.redis.CacheSystemDomainList(sysDomains, 30*time.Minute)

	return sysDomains, nil
}

// ListActiveSystemDomains 获取所有已激活的系统域名
func (s *Store) ListActiveSystemDomains() ([]*domain.SystemDomain, error) {
	// 活跃域名查询直接从 PostgreSQL 获取（不缓存）
	return s.postgres.ListActiveSystemDomains()
}

// UpdateSystemDomain 更新系统域名
func (s *Store) UpdateSystemDomain(sysDomain *domain.SystemDomain) error {
	// 更新 PostgreSQL
	if err := s.postgres.UpdateSystemDomain(sysDomain); err != nil {
		return err
	}

	// 更新 Redis 缓存
	s.redis.CacheSystemDomain(sysDomain, 1*time.Hour)

	// 删除系统域名列表缓存
	s.redis.Delete("system_domains:list")

	return nil
}

// DeleteSystemDomain 删除系统域名
func (s *Store) DeleteSystemDomain(domainID string) error {
	// 从 PostgreSQL 删除
	if err := s.postgres.DeleteSystemDomain(domainID); err != nil {
		return err
	}

	// 删除 Redis 缓存
	s.redis.Delete(fmt.Sprintf("system_domain:%s", domainID))
	s.redis.Delete("system_domains:list")

	return nil
}

// SetDefaultSystemDomain 设置默认系统域名
func (s *Store) SetDefaultSystemDomain(domainID string) error {
	// 更新 PostgreSQL
	if err := s.postgres.SetDefaultSystemDomain(domainID); err != nil {
		return err
	}

	// 删除相关缓存（强制重新加载）
	s.redis.Delete("system_domains:list")
	s.redis.Delete("system_domain:default")

	return nil
}

// GetDefaultSystemDomain 获取默认系统域名
func (s *Store) GetDefaultSystemDomain() (*domain.SystemDomain, error) {
	// 先尝试从 Redis 获取
	if sysDomain, err := s.redis.GetCachedDefaultSystemDomain(); err == nil {
		return sysDomain, nil
	}

	// 从 PostgreSQL 获取
	sysDomain, err := s.postgres.GetDefaultSystemDomain()
	if err != nil {
		return nil, err
	}

	// 缓存到 Redis（1小时过期）
	s.redis.CacheDefaultSystemDomain(sysDomain, 1*time.Hour)

	return sysDomain, nil
}

// IncrementSystemDomainMailboxCount 增加系统域名邮箱计数
func (s *Store) IncrementSystemDomainMailboxCount(domainName string) error {
	// 更新 PostgreSQL
	if err := s.postgres.IncrementSystemDomainMailboxCount(domainName); err != nil {
		return err
	}

	// 删除相关缓存（强制重新加载）
	s.redis.Delete("system_domains:list")

	return nil
}

// DecrementSystemDomainMailboxCount 减少系统域名邮箱计数
func (s *Store) DecrementSystemDomainMailboxCount(domainName string) error {
	// 更新 PostgreSQL
	if err := s.postgres.DecrementSystemDomainMailboxCount(domainName); err != nil {
		return err
	}

	// 删除相关缓存（强制重新加载）
	s.redis.Delete("system_domains:list")

	return nil
}

// DeleteUnverifiedSystemDomains 删除指定时间前创建且未验证的域名
func (s *Store) DeleteUnverifiedSystemDomains(before time.Time) (int, error) {
	// 直接从 PostgreSQL 删除
	count, err := s.postgres.DeleteUnverifiedSystemDomains(before)
	if err != nil {
		return 0, err
	}

	// 删除相关缓存
	s.redis.Delete("system_domains:list")

	return count, nil
}
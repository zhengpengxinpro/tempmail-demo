package memory

import (
	"errors"
	"time"

	"tempmail/backend/internal/domain"
)

var (
	ErrSystemDomainNotFound = errors.New("system domain not found")
	ErrSystemDomainExists   = errors.New("system domain already exists")
)

// SaveSystemDomain 保存系统域名
func (s *Store) SaveSystemDomain(sysDomain *domain.SystemDomain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查域名是否已被其他记录使用（更新同一记录时允许）
	if existingID, ok := s.bySystemDomain[sysDomain.Domain]; ok && existingID != sysDomain.ID {
		return ErrSystemDomainExists
	}

	s.systemDomains[sysDomain.ID] = sysDomain
	s.bySystemDomain[sysDomain.Domain] = sysDomain.ID

	return nil
}

// GetSystemDomain 根据 ID 获取系统域名
func (s *Store) GetSystemDomain(id string) (*domain.SystemDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sysDomain, ok := s.systemDomains[id]
	if !ok {
		return nil, ErrSystemDomainNotFound
	}

	return sysDomain, nil
}

// GetSystemDomainByDomain 根据域名获取系统域名
func (s *Store) GetSystemDomainByDomain(domainName string) (*domain.SystemDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	domainID, ok := s.bySystemDomain[domainName]
	if !ok {
		return nil, ErrSystemDomainNotFound
	}

	sysDomain, ok := s.systemDomains[domainID]
	if !ok {
		return nil, ErrSystemDomainNotFound
	}

	return sysDomain, nil
}

// ListSystemDomains 获取所有系统域名
func (s *Store) ListSystemDomains() ([]*domain.SystemDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.SystemDomain, 0, len(s.systemDomains))
	for _, d := range s.systemDomains {
		result = append(result, d)
	}

	return result, nil
}

// ListActiveSystemDomains 获取所有已激活的系统域名
func (s *Store) ListActiveSystemDomains() ([]*domain.SystemDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.SystemDomain, 0)
	for _, d := range s.systemDomains {
		if d.IsActive && d.Status == domain.SystemDomainStatusVerified {
			result = append(result, d)
		}
	}

	return result, nil
}

// DeleteSystemDomain 删除系统域名
func (s *Store) DeleteSystemDomain(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sysDomain, ok := s.systemDomains[id]
	if !ok {
		return ErrSystemDomainNotFound
	}

	delete(s.systemDomains, id)
	delete(s.bySystemDomain, sysDomain.Domain)

	return nil
}

// IncrementSystemDomainMailboxCount 增加系统域名邮箱计数
func (s *Store) IncrementSystemDomainMailboxCount(domainName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainID, ok := s.bySystemDomain[domainName]
	if !ok {
		// 如果域名不存在，不报错（可能是旧数据）
		return nil
	}

	sysDomain, ok := s.systemDomains[domainID]
	if ok {
		sysDomain.MailboxCount++
	}

	return nil
}

// DecrementSystemDomainMailboxCount 减少系统域名邮箱计数
func (s *Store) DecrementSystemDomainMailboxCount(domainName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainID, ok := s.bySystemDomain[domainName]
	if !ok {
		// 如果域名不存在，不报错（可能是旧数据）
		return nil
	}

	sysDomain, ok := s.systemDomains[domainID]
	if ok && sysDomain.MailboxCount > 0 {
		sysDomain.MailboxCount--
	}

	return nil
}

// DeleteUnverifiedSystemDomains 删除指定时间前创建且未验证的域名
func (s *Store) DeleteUnverifiedSystemDomains(before time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	toDelete := make([]string, 0)

	for id, sysDomain := range s.systemDomains {
		// 未验证且创建时间早于指定时间
		if sysDomain.Status == domain.SystemDomainStatusPending && sysDomain.CreatedAt.Before(before) {
			toDelete = append(toDelete, id)
		}
	}

	// 删除域名
	for _, id := range toDelete {
		sysDomain := s.systemDomains[id]
		delete(s.systemDomains, id)
		delete(s.bySystemDomain, sysDomain.Domain)
		count++
	}

	return count, nil
}

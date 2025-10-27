package memory

import (
	"errors"

	"tempmail/backend/internal/domain"
)

// ========== User Domain Repository ==========

var (
	ErrUserDomainNotFound = errors.New("user domain not found")
	ErrDomainExists       = errors.New("domain already exists")
)

// SaveUserDomain 保存用户域名
func (s *Store) SaveUserDomain(domain *domain.UserDomain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查域名是否已被其他记录使用（更新同一记录时允许）
	if existingID, ok := s.byDomain[domain.Domain]; ok && existingID != domain.ID {
		return ErrDomainExists
	}

	s.userDomains[domain.ID] = domain
	s.byDomain[domain.Domain] = domain.ID

	return nil
}

// GetUserDomain 根据 ID 获取用户域名
func (s *Store) GetUserDomain(id string) (*domain.UserDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	domain, ok := s.userDomains[id]
	if !ok {
		return nil, ErrUserDomainNotFound
	}

	return domain, nil
}

// GetUserDomainByDomain 根据域名获取
func (s *Store) GetUserDomainByDomain(domainName string) (*domain.UserDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	domainID, ok := s.byDomain[domainName]
	if !ok {
		return nil, ErrUserDomainNotFound
	}

	domain, ok := s.userDomains[domainID]
	if !ok {
		return nil, ErrUserDomainNotFound
	}

	return domain, nil
}

// ListUserDomainsByUserID 获取用户的所有域名
func (s *Store) ListUserDomainsByUserID(userID string) ([]*domain.UserDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.UserDomain, 0)
	for _, d := range s.userDomains {
		if d.UserID == userID {
			result = append(result, d)
		}
	}

	return result, nil
}

// ListAllUserDomains 获取所有用户域名
func (s *Store) ListAllUserDomains() ([]*domain.UserDomain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.UserDomain, 0, len(s.userDomains))
	for _, d := range s.userDomains {
		result = append(result, d)
	}

	return result, nil
}

// UpdateUserDomain 更新用户域名
func (s *Store) UpdateUserDomain(userDomain *domain.UserDomain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.userDomains[userDomain.ID] = userDomain
	return nil
}

// DeleteUserDomain 删除用户域名
func (s *Store) DeleteUserDomain(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domain, ok := s.userDomains[id]
	if !ok {
		return ErrUserDomainNotFound
	}

	delete(s.userDomains, id)
	delete(s.byDomain, domain.Domain)

	return nil
}

// IncrementMailboxCount 增加邮箱计数
func (s *Store) IncrementMailboxCount(domainName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainID, ok := s.byDomain[domainName]
	if !ok {
		// 如果域名不存在，不报错（可能是系统默认域名）
		return nil
	}

	domain, ok := s.userDomains[domainID]
	if ok {
		domain.MailboxCount++
	}

	return nil
}

// DecrementMailboxCount 减少邮箱计数
func (s *Store) DecrementMailboxCount(domainName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	domainID, ok := s.byDomain[domainName]
	if !ok {
		// 如果域名不存在，不报错（可能是系统默认域名）
		return nil
	}

	domain, ok := s.userDomains[domainID]
	if ok && domain.MailboxCount > 0 {
		domain.MailboxCount--
	}

	return nil
}

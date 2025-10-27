package sql

import (
	"time"

	"tempmail/backend/internal/domain"
)

// ========== System Domain Repository ==========

// SaveSystemDomain 保存系统域名
func (s *Store) SaveSystemDomain(sysDomain *domain.SystemDomain) error {
	query := `
		INSERT INTO system_domains (id, domain, is_active, is_default, created_at, mailbox_count, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_active = VALUES(is_active),
			is_default = VALUES(is_default),
			notes = VALUES(notes)
	`
	_, err := s.db.Exec(query,
		sysDomain.ID,
		sysDomain.Domain,
		sysDomain.IsActive,
		sysDomain.IsDefault,
		sysDomain.CreatedAt,
		sysDomain.MailboxCount,
		sysDomain.Notes,
	)
	return err
}

// GetSystemDomain 根据ID获取系统域名
func (s *Store) GetSystemDomain(domainID string) (*domain.SystemDomain, error) {
	query := `
		SELECT id, domain, is_active, is_default, created_at, mailbox_count, notes
		FROM system_domains
		WHERE id = ?
	`
	var sysDomain domain.SystemDomain
	err := s.db.QueryRow(query, domainID).Scan(
		&sysDomain.ID,
		&sysDomain.Domain,
		&sysDomain.IsActive,
		&sysDomain.IsDefault,
		&sysDomain.CreatedAt,
		&sysDomain.MailboxCount,
		&sysDomain.Notes,
	)
	if err != nil {
		return nil, err
	}
	return &sysDomain, nil
}

// GetSystemDomainByDomain 根据域名获取系统域名
func (s *Store) GetSystemDomainByDomain(domainName string) (*domain.SystemDomain, error) {
	query := `
		SELECT id, domain, is_active, is_default, created_at, mailbox_count, notes
		FROM system_domains
		WHERE domain = ?
	`
	var sysDomain domain.SystemDomain
	err := s.db.QueryRow(query, domainName).Scan(
		&sysDomain.ID,
		&sysDomain.Domain,
		&sysDomain.IsActive,
		&sysDomain.IsDefault,
		&sysDomain.CreatedAt,
		&sysDomain.MailboxCount,
		&sysDomain.Notes,
	)
	if err != nil {
		return nil, err
	}
	return &sysDomain, nil
}

// ListSystemDomains 获取所有系统域名
func (s *Store) ListSystemDomains() ([]*domain.SystemDomain, error) {
	query := `
		SELECT id, domain, is_active, is_default, created_at, mailbox_count, notes
		FROM system_domains
		ORDER BY is_default DESC, domain ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sysDomains []*domain.SystemDomain
	for rows.Next() {
		var sysDomain domain.SystemDomain
		err := rows.Scan(
			&sysDomain.ID,
			&sysDomain.Domain,
			&sysDomain.IsActive,
			&sysDomain.IsDefault,
			&sysDomain.CreatedAt,
			&sysDomain.MailboxCount,
			&sysDomain.Notes,
		)
		if err != nil {
			return nil, err
		}
		sysDomains = append(sysDomains, &sysDomain)
	}
	return sysDomains, rows.Err()
}

// ListActiveSystemDomains 获取所有已激活的系统域名
func (s *Store) ListActiveSystemDomains() ([]*domain.SystemDomain, error) {
	query := `
		SELECT id, domain, is_active, is_default, created_at, mailbox_count, notes
		FROM system_domains
		WHERE is_active = true
		ORDER BY is_default DESC, domain ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sysDomains []*domain.SystemDomain
	for rows.Next() {
		var sysDomain domain.SystemDomain
		err := rows.Scan(
			&sysDomain.ID,
			&sysDomain.Domain,
			&sysDomain.IsActive,
			&sysDomain.IsDefault,
			&sysDomain.CreatedAt,
			&sysDomain.MailboxCount,
			&sysDomain.Notes,
		)
		if err != nil {
			return nil, err
		}
		sysDomains = append(sysDomains, &sysDomain)
	}
	return sysDomains, rows.Err()
}

// UpdateSystemDomain 更新系统域名
func (s *Store) UpdateSystemDomain(sysDomain *domain.SystemDomain) error {
	query := `
		UPDATE system_domains
		SET domain = ?, is_active = ?, is_default = ?, notes = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query,
		sysDomain.Domain,
		sysDomain.IsActive,
		sysDomain.IsDefault,
		sysDomain.Notes,
		sysDomain.ID,
	)
	return err
}

// DeleteSystemDomain 删除系统域名
func (s *Store) DeleteSystemDomain(domainID string) error {
	query := `DELETE FROM system_domains WHERE id = ?`
	_, err := s.db.Exec(query, domainID)
	return err
}

// SetDefaultSystemDomain 设置默认系统域名
func (s *Store) SetDefaultSystemDomain(domainID string) error {
	// 开启事务
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 清除所有域名的默认标志
	_, err = tx.Exec(`UPDATE system_domains SET is_default = false`)
	if err != nil {
		return err
	}

	// 设置指定域名为默认
	_, err = tx.Exec(`UPDATE system_domains SET is_default = true WHERE id = ?`, domainID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetDefaultSystemDomain 获取默认系统域名
func (s *Store) GetDefaultSystemDomain() (*domain.SystemDomain, error) {
	query := `
		SELECT id, domain, is_active, is_default, created_at, mailbox_count, notes
		FROM system_domains
		WHERE is_default = true
		LIMIT 1
	`
	var sysDomain domain.SystemDomain
	err := s.db.QueryRow(query).Scan(
		&sysDomain.ID,
		&sysDomain.Domain,
		&sysDomain.IsActive,
		&sysDomain.IsDefault,
		&sysDomain.CreatedAt,
		&sysDomain.MailboxCount,
		&sysDomain.Notes,
	)
	if err != nil {
		return nil, err
	}
	return &sysDomain, nil
}

// IncrementSystemDomainMailboxCount 增加系统域名邮箱计数
func (s *Store) IncrementSystemDomainMailboxCount(domainName string) error {
	query := `UPDATE system_domains SET mailbox_count = mailbox_count + 1 WHERE domain = ?`
	_, err := s.db.Exec(query, domainName)
	return err
}

// DecrementSystemDomainMailboxCount 减少系统域名邮箱计数
func (s *Store) DecrementSystemDomainMailboxCount(domainName string) error {
	query := `UPDATE system_domains SET mailbox_count = mailbox_count - 1 WHERE domain = ? AND mailbox_count > 0`
	_, err := s.db.Exec(query, domainName)
	return err
}

// DeleteUnverifiedSystemDomains 删除指定时间前创建且未验证的域名
func (s *Store) DeleteUnverifiedSystemDomains(before time.Time) (int, error) {
	// 注意：系统域名通常不需要验证机制，这里返回0
	// 如果未来需要实现验证机制，可在system_domains表添加verified字段
	return 0, nil
}

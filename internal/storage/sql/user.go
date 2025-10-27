package sql

import (
	"database/sql"
	"time"

	"tempmail/backend/internal/domain"
)

// ========== User Repository ==========

// CreateUser 创建新用户
func (s *Store) CreateUser(user *domain.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, role, tier, is_active, is_email_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Tier,
		user.IsActive,
		user.IsEmailVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// GetUserByID 根据ID获取用户
func (s *Store) GetUserByID(id string) (*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, tier, is_active, is_email_verified, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE id = ?
	`
	var user domain.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Tier,
		&user.IsActive,
		&user.IsEmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *Store) GetUserByEmail(email string) (*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, tier, is_active, is_email_verified, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE email = ?
	`
	var user domain.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Tier,
		&user.IsActive,
		&user.IsEmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *Store) GetUserByUsername(username string) (*domain.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, tier, is_active, is_email_verified, 
		       created_at, updated_at, last_login_at
		FROM users
		WHERE lower(username) = lower(?)
	`
	var user domain.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Tier,
		&user.IsActive,
		&user.IsEmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (s *Store) UpdateUser(user *domain.User) error {
	query := `
		UPDATE users
		SET email = ?, username = ?, password_hash = ?, role = ?, tier = ?, 
		    is_active = ?, is_email_verified = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Tier,
		user.IsActive,
		user.IsEmailVerified,
		time.Now(),
		user.ID,
	)
	return err
}

// UpdateLastLogin 更新用户最后登录时间
func (s *Store) UpdateLastLogin(userID string) error {
	query := `UPDATE users SET last_login_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), userID)
	return err
}

// ListUsers 列出用户（支持分页和过滤）
func (s *Store) ListUsers(page, pageSize int, search string, role *domain.UserRole, tier *domain.UserTier, isActive *bool) ([]domain.User, int, error) {
	// 构建查询条件
	where := "WHERE 1=1"
	args := []interface{}{}

	if search != "" {
		where += " AND (email LIKE ? OR username LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	if role != nil {
		where += " AND role = ?"
		args = append(args, *role)
	}

	if tier != nil {
		where += " AND tier = ?"
		args = append(args, *tier)
	}

	if isActive != nil {
		where += " AND is_active = ?"
		args = append(args, *isActive)
	}

	// 获取总数
	countQuery := "SELECT COUNT(*) FROM users " + where
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取用户列表
	offset := (page - 1) * pageSize
	query := `
		SELECT id, email, username, password_hash, role, tier, is_active, is_email_verified, 
		       created_at, updated_at, last_login_at
		FROM users
	` + where + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	args = append(args, pageSize, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var user domain.User
		var lastLoginAt sql.NullTime

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.Role,
			&user.Tier,
			&user.IsActive,
			&user.IsEmailVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLoginAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}

		users = append(users, user)
	}

	return users, total, rows.Err()
}

// DeleteUser 删除用户
func (s *Store) DeleteUser(userID string) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := s.db.Exec(query, userID)
	return err
}

// GetUserByAPIKey 根据API Key获取用户
func (s *Store) GetUserByAPIKey(apiKey string) (*domain.User, error) {
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, u.role, u.tier, 
		       u.is_active, u.is_email_verified, u.created_at, u.updated_at, u.last_login_at
		FROM users u
		INNER JOIN api_keys ak ON u.id = ak.user_id
		WHERE ak.key = ? AND u.is_active = true
	`
	var user domain.User
	var lastLoginAt sql.NullTime

	err := s.db.QueryRow(query, apiKey).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Tier,
		&user.IsActive,
		&user.IsEmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return &user, nil
}

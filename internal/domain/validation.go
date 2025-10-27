package domain

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

// 验证相关的错误定义
var (
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrEmailTooLong     = errors.New("email address too long")
	ErrLocalPartTooLong = errors.New("local part too long (max 64 chars)")
	ErrDomainTooLong    = errors.New("domain too long (max 253 chars)")
	ErrInvalidLocalPart = errors.New("invalid local part format")
	ErrInvalidDomain    = errors.New("invalid domain format")
	ErrPasswordTooShort = errors.New("password too short (min 8 chars)")
	ErrPasswordTooLong  = errors.New("password too long (max 128 chars)")
	ErrUsernameTooShort = errors.New("username too short (min 3 chars)")
	ErrUsernameTooLong  = errors.New("username too long (max 32 chars)")
	ErrInvalidUsername  = errors.New("invalid username format")
)

// 验证常量
const (
	// RFC 5322 邮箱地址长度限制
	MaxEmailLength     = 254 // 整个邮箱地址最大长度
	MaxLocalPartLength = 64  // 本地部分最大长度(@前面)
	MaxDomainLength    = 253 // 域名最大长度

	// 密码长度限制
	MinPasswordLength = 8
	MaxPasswordLength = 128

	// 用户名长度限制
	MinUsernameLength = 3
	MaxUsernameLength = 32
)

// 正则表达式
var (
	// 本地部分验证（更严格的规则，要求3-64字符）
	localPartRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)

	// 域名验证（支持子域名）
	domainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?(\.[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?)*$`)

	// 用户名验证（必须以字母开头）
	usernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*[a-zA-Z0-9]$|^[a-zA-Z]$`)
)

// EmailValidator 邮箱验证器
type EmailValidator struct{}

// NewEmailValidator 创建邮箱验证器
func NewEmailValidator() *EmailValidator {
	return &EmailValidator{}
}

// ValidateEmail 完整验证邮箱地址
func (v *EmailValidator) ValidateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	// 长度检查
	if len(email) > MaxEmailLength {
		return ErrEmailTooLong
	}

	// 使用标准库进行基础格式验证
	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}

	// 分离本地部分和域名
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ErrInvalidEmail
	}

	localPart := parts[0]
	domain := parts[1]

	// 验证本地部分
	if err := v.ValidateLocalPart(localPart); err != nil {
		return err
	}

	// 验证域名
	if err := v.ValidateDomain(domain); err != nil {
		return err
	}

	return nil
}

// ValidateLocalPart 验证邮箱本地部分
func (v *EmailValidator) ValidateLocalPart(localPart string) error {
	if localPart == "" {
		return ErrInvalidLocalPart
	}

	// 长度检查（最小3字符，最大64字符）
	if len(localPart) < 3 {
		return ErrInvalidLocalPart
	}
	if len(localPart) > MaxLocalPartLength {
		return ErrLocalPartTooLong
	}

	// 格式检查
	if !localPartRegex.MatchString(localPart) {
		return ErrInvalidLocalPart
	}

	// 不允许连续的特殊字符
	if strings.Contains(localPart, "..") || strings.Contains(localPart, ".-") ||
		strings.Contains(localPart, "-.") || strings.Contains(localPart, "--") ||
		strings.Contains(localPart, "__") || strings.Contains(localPart, "_.") ||
		strings.Contains(localPart, "._") {
		return ErrInvalidLocalPart
	}

	return nil
}

// ValidateDomain 验证域名
func (v *EmailValidator) ValidateDomain(domain string) error {
	if domain == "" {
		return ErrInvalidDomain
	}

	// 长度检查
	if len(domain) > MaxDomainLength {
		return ErrDomainTooLong
	}

	// 格式检查
	if !domainRegex.MatchString(domain) {
		return ErrInvalidDomain
	}

	// 检查每个标签的长度（不超过63字符）
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return ErrInvalidDomain
		}
	}

	return nil
}

// ValidatePasswordError 验证密码并返回错误
func ValidatePasswordError(password string) error {
	if len(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	return nil
}

// ValidateUsername 验证用户名
func ValidateUsername(username string) error {
	if len(username) < MinUsernameLength {
		return ErrUsernameTooShort
	}

	if len(username) > MaxUsernameLength {
		return ErrUsernameTooLong
	}

	if !usernameRegex.MatchString(username) {
		return ErrInvalidUsername
	}

	return nil
}
// 简化的验证函数，返回bool值用于测试
func ValidateEmail(email string) bool {
	// 基本格式检查
	if email == "" {
		return false
	}
	
	// 检查是否包含@
	if !strings.Contains(email, "@") {
		return false
	}
	
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	localPart := parts[0]
	domain := parts[1]
	
	// 基本长度检查
	if localPart == "" || domain == "" {
		return false
	}
	
	// 检查不允许的特殊字符 ($, 等)
	for _, r := range localPart {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == '+') {
			return false
		}
	}
	
	// 使用标准库进行验证
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	
	return true
}

func ValidateUsernameBool(username string) bool {
	return ValidateUsername(username) == nil
}

func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", r):
			hasSpecial = true
		}
	}
	
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func ValidateDomain(domain string) bool {
	if domain == "" {
		return false
	}
	
	// 必须包含点
	if !strings.Contains(domain, ".") {
		return false
	}
	
	// 不能以点开头或结尾
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}
	
	// 不能包含连续的点
	if strings.Contains(domain, "..") {
		return false
	}
	
	// 不能包含空格
	if strings.Contains(domain, " ") {
		return false
	}
	
	// 不能以破折号开头或结尾
	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		return false
	}
	
	// 检查每个标签
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if label == "" {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		// 只允许字母、数字和破折号
		for _, r := range label {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}
	
	return true
}

func ValidateTagName(name string) bool {
	if name == "" || len(strings.TrimSpace(name)) == 0 {
		return false
	}
	if len(name) > 100 {
		return false
	}
	// 不允许特殊字符和控制字符
	for _, r := range name {
		if r < 32 || r == '@' || r == '#' || r == '$' || r == '%' {
			return false
		}
	}
	return true
}

func ValidateColorCode(color string) bool {
	if len(color) != 4 && len(color) != 7 {
		return false
	}
	if !strings.HasPrefix(color, "#") {
		return false
	}
	hex := color[1:]
	for _, r := range hex {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func ValidateSubject(subject string) bool {
	if len(subject) > 255 {
		return false
	}
	// 不允许控制字符（包括tab和换行）
	for _, r := range subject {
		if r < 32 {
			return false
		}
	}
	return true
}

func ValidateMessageBody(body string) bool {
	if len(body) > 100000 {
		return false
	}
	return true
}

// User 验证方法
func (u *User) Validate() error {
	if ValidateUsername(u.Username) != nil {
		return errors.New("invalid username")
	}
	validator := NewEmailValidator()
	if validator.ValidateEmail(u.Email) != nil {
		return errors.New("invalid email")
	}
	if u.Role != RoleUser && u.Role != RoleAdmin {
		return errors.New("invalid role")
	}
	return nil
}

// CreateMailboxRequest 请求结构
type CreateMailboxRequest struct {
	UserID string `json:"userId"`
	Domain string `json:"domain"`
}

func (r *CreateMailboxRequest) Validate() error {
	if r.UserID == "" {
		return errors.New("user ID is required")
	}
	if !ValidateDomain(r.Domain) {
		return errors.New("invalid domain")
	}
	return nil
}

// CreateMessageRequest 请求结构
type CreateMessageRequest struct {
	MailboxID string   `json:"mailboxId"`
	From      string   `json:"from"`
	To        []string `json:"to"`
	Subject   string   `json:"subject"`
	Body      string   `json:"body"`
}

func (r *CreateMessageRequest) Validate() error {
	if r.MailboxID == "" {
		return errors.New("mailbox ID is required")
	}
	if !ValidateEmail(r.From) {
		return errors.New("invalid from email")
	}
	if len(r.To) == 0 {
		return errors.New("recipients are required")
	}
	for _, email := range r.To {
		if !ValidateEmail(email) {
			return errors.New("invalid recipient email")
		}
	}
	return nil
}

// 认证相关的请求结构
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type ChangePasswordRequest struct {
	UserID      string `json:"userId"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
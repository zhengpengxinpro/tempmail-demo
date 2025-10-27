package auth

import (
	"tempmail/backend/internal/auth/jwt"
	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
)

// JWTManager JWT管理器包装
type JWTManager struct {
	manager *jwt.Manager
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(cfg *config.JWTConfig) *JWTManager {
	manager := jwt.NewManager(cfg.Secret, cfg.Issuer, cfg.AccessExpiry, cfg.RefreshExpiry)
	return &JWTManager{manager: manager}
}

// TokenResponse 令牌响应
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// Claims JWT声明
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// GenerateTokens 生成令牌对
func (j *JWTManager) GenerateTokens(userID string, role string) (*TokenResponse, error) {
	tokenPair, err := j.manager.GenerateTokenPair(userID, "", role)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// ValidateToken 验证令牌
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	claims, err := j.manager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &Claims{
		UserID: claims.UserID,
		Role:   claims.Tier, // 使用Tier作为Role
	}, nil
}

// RefreshToken 刷新令牌
func (j *JWTManager) RefreshToken(refreshToken string) (*TokenResponse, error) {
	// 先验证刷新令牌
	claims, err := j.manager.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 生成新的令牌对
	tokenPair, err := j.manager.GenerateTokenPair(claims.UserID, claims.Email, claims.Tier)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// AuthService 认证服务包装
type AuthService struct {
	service    *Service
	jwtManager *JWTManager
}

// NewAuthService 创建认证服务
func NewAuthService(userRepo UserRepository, jwtManager *JWTManager) *AuthService {
	service := NewService(userRepo)
	return &AuthService{
		service:    service,
		jwtManager: jwtManager,
	}
}

// AuthResponse 认证响应
type AuthResponse struct {
	User         *domain.User `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	TokenType    string       `json:"tokenType"`
	ExpiresIn    int64        `json:"expiresIn"`
}

// Register 用户注册
func (a *AuthService) Register(req *domain.RegisterRequest) (*AuthResponse, error) {
	input := RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	}

	user, err := a.service.Register(input)
	if err != nil {
		return nil, err
	}

	// 生成令牌
	tokens, err := a.jwtManager.GenerateTokens(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// Login 用户登录
func (a *AuthService) Login(req *domain.LoginRequest) (*AuthResponse, error) {
	input := LoginInput{
		Identifier: req.Username, // 可以是用户名或邮箱
		Password:   req.Password,
	}

	user, err := a.service.Login(input)
	if err != nil {
		return nil, err
	}

	// 生成令牌
	tokens, err := a.jwtManager.GenerateTokens(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}

// RefreshToken 刷新令牌
func (a *AuthService) RefreshToken(req *domain.RefreshTokenRequest) (*TokenResponse, error) {
	return a.jwtManager.RefreshToken(req.RefreshToken)
}

// GetUserByID 根据ID获取用户
func (a *AuthService) GetUserByID(userID string) (*domain.User, error) {
	return a.service.GetUserByID(userID)
}

// ChangePassword 修改密码
func (a *AuthService) ChangePassword(req *domain.ChangePasswordRequest) error {
	return a.service.ChangePassword(req.UserID, req.OldPassword, req.NewPassword)
}

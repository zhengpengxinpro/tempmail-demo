package auth

import (
	"strings"
	"testing"
	"time"

	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage/memory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Register(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Test successful registration
	req := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}

	response, err := service.Register(req)
	require.NoError(t, err)
	assert.NotEmpty(t, response.User.ID)
	assert.Equal(t, "testuser", response.User.Username)
	assert.Equal(t, "test@example.com", response.User.Email)
	assert.Equal(t, domain.RoleUser, response.User.Role)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register first user
	req1 := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test1@example.com",
		Password: "Password123!",
	}
	_, err := service.Register(req1)
	require.NoError(t, err)

	// Try to register with same username but different email - should succeed
	// (current implementation only checks email uniqueness)
	req2 := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test2@example.com",
		Password: "Password123!",
	}

	_, err = service.Register(req2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username already exists")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Create first user
	user := &domain.User{
		ID:           "user-1",
		Username:     "testuser1",
		Email:        "test@example.com",
		PasswordHash: "somehash",
		Role:         domain.RoleUser,
		IsActive:     true,
	}
	err := store.CreateUser(user)
	require.NoError(t, err)

	// Try to register with same email
	req := &domain.RegisterRequest{
		Username: "testuser2",
		Email:    "test@example.com",
		Password: "Password123!",
	}

	_, err = service.Register(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
}

func TestAuthService_Login(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	_, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test successful login with username
	loginReq := &domain.LoginRequest{
		Username: "testuser",
		Password: "Password123!",
	}

	response, err := service.Login(loginReq)
	require.NoError(t, err)
	assert.Equal(t, "testuser", response.User.Username)
	assert.Equal(t, "test@example.com", response.User.Email)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
}

func TestAuthService_Login_WithEmail(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	_, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test successful login with email
	loginReq := &domain.LoginRequest{
		Username: "test@example.com", // Using email as username
		Password: "Password123!",
	}

	response, err := service.Login(loginReq)
	require.NoError(t, err)
	assert.Equal(t, "testuser", response.User.Username)
	assert.Equal(t, "test@example.com", response.User.Email)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	_, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test login with wrong password
	loginReq := &domain.LoginRequest{
		Username: "testuser",
		Password: "WrongPassword123!",
	}

	_, err = service.Login(loginReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Test login with non-existent user
	loginReq := &domain.LoginRequest{
		Username: "nonexistent",
		Password: "Password123!",
	}

	_, err := service.Login(loginReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestAuthService_RefreshToken(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	registerResponse, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test token refresh
	req := &domain.RefreshTokenRequest{
		RefreshToken: registerResponse.RefreshToken,
	}

	// Wait a moment to ensure token timestamps are different (JWT uses second precision)
	time.Sleep(1100 * time.Millisecond)

	response, err := service.RefreshToken(req)
	require.NoError(t, err)
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEqual(t, registerResponse.AccessToken, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Test refresh with invalid token
	req := &domain.RefreshTokenRequest{
		RefreshToken: "invalid-refresh-token",
	}

	_, err := service.RefreshToken(req)
	assert.Error(t, err)
}

func TestAuthService_GetUserByID(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	registerResponse, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test GetUserByID
	user, err := service.GetUserByID(registerResponse.User.ID)
	require.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, domain.RoleUser, user.Role)
}

func TestAuthService_GetUserByID_NotFound(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Test GetUserByID with non-existent user
	_, err := service.GetUserByID("non-existent-id")
	assert.Error(t, err)
}

func TestAuthService_ChangePassword(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	registerResponse, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test password change
	req := &domain.ChangePasswordRequest{
		UserID:      registerResponse.User.ID,
		OldPassword: "Password123!",
		NewPassword: "NewPassword123!",
	}

	err = service.ChangePassword(req)
	require.NoError(t, err)

	// Verify new password works
	loginReq := &domain.LoginRequest{
		Username: "testuser",
		Password: "NewPassword123!",
	}

	_, err = service.Login(loginReq)
	assert.NoError(t, err)

	// Verify old password doesn't work
	loginReq.Password = "Password123!"
	_, err = service.Login(loginReq)
	assert.Error(t, err)
}

func TestAuthService_ChangePassword_WrongOldPassword(t *testing.T) {
	store := memory.NewStore(24 * time.Hour)
	cfg := &config.JWTConfig{
		Secret:        strings.Repeat("a", 32),
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	jwtManager := NewJWTManager(cfg)
	service := NewAuthService(store, jwtManager)

	// Register a user first
	registerReq := &domain.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "Password123!",
	}
	registerResponse, err := service.Register(registerReq)
	require.NoError(t, err)

	// Test password change with wrong old password
	req := &domain.ChangePasswordRequest{
		UserID:      registerResponse.User.ID,
		OldPassword: "WrongPassword123!",
		NewPassword: "NewPassword123!",
	}

	err = service.ChangePassword(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid old password")
}

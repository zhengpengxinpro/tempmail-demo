package auth

import (
	"testing"
	"time"

	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateTokens(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager := NewJWTManager(cfg)
	
	tokens, err := manager.GenerateTokens("test-user-1", string(domain.RoleUser))
	require.NoError(t, err)
	
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, int64(15*60), tokens.ExpiresIn) // 15 minutes in seconds
}

func TestJWTManager_ValidateToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager := NewJWTManager(cfg)
	
	// Generate valid token
	tokens, err := manager.GenerateTokens("test-user-1", string(domain.RoleUser))
	require.NoError(t, err)
	
	// Validate token
	claims, err := manager.ValidateToken(tokens.AccessToken)
	require.NoError(t, err)
	
	assert.Equal(t, "test-user-1", claims.UserID)
	assert.Equal(t, string(domain.RoleUser), claims.Role)
}

func TestJWTManager_ValidateToken_Invalid(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager := NewJWTManager(cfg)
	
	// Test invalid token
	_, err := manager.ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestJWTManager_ValidateToken_Expired(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessExpiry:  1 * time.Millisecond, // Very short expiry
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager := NewJWTManager(cfg)
	
	// Generate token
	tokens, err := manager.GenerateTokens("test-user-1", string(domain.RoleUser))
	require.NoError(t, err)
	
	// Wait for expiration
	time.Sleep(10 * time.Millisecond)
	
	// Validate expired token
	_, err = manager.ValidateToken(tokens.AccessToken)
	assert.Error(t, err)
}

func TestJWTManager_RefreshToken(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager := NewJWTManager(cfg)
	
	// Generate initial tokens
	tokens, err := manager.GenerateTokens("test-user-1", string(domain.RoleUser))
	require.NoError(t, err)
	
	// Wait a moment to ensure different timestamps
	time.Sleep(1 * time.Second)
	
	// Refresh tokens
	newTokens, err := manager.RefreshToken(tokens.RefreshToken)
	require.NoError(t, err)
	
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	// Access token should be different due to different timestamps
	assert.NotEqual(t, tokens.AccessToken, newTokens.AccessToken)
	// Refresh token should also be different
	assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken)
}

func TestJWTManager_RefreshToken_Invalid(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:        "test-secret",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager := NewJWTManager(cfg)
	
	// Test invalid refresh token
	_, err := manager.RefreshToken("invalid-refresh-token")
	assert.Error(t, err)
}

func TestJWTManager_DifferentSecrets(t *testing.T) {
	cfg1 := &config.JWTConfig{
		Secret:        "secret-1",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	cfg2 := &config.JWTConfig{
		Secret:        "secret-2",
		Issuer:        "test",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	
	manager1 := NewJWTManager(cfg1)
	manager2 := NewJWTManager(cfg2)
	
	// Generate token with manager1
	tokens, err := manager1.GenerateTokens("test-user-1", string(domain.RoleUser))
	require.NoError(t, err)
	
	// Try to validate with manager2 (different secret)
	_, err = manager2.ValidateToken(tokens.AccessToken)
	assert.Error(t, err)
}
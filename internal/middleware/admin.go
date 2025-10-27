package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/auth"
	"tempmail/backend/internal/domain"
)

// AdminAuth 管理员权限中间件
type AdminAuth struct {
	authService *auth.Service
}

// NewAdminAuth 创建管理员权限中间件
func NewAdminAuth(authService *auth.Service) *AdminAuth {
	return &AdminAuth{
		authService: authService,
	}
}

// RequireAdmin 要求管理员权限（Admin或Super）
func (a *AdminAuth) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID（由JWT中间件设置）
		userIDVal, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			c.Abort()
			return
		}

		// 获取用户信息
		user, err := a.authService.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// 临时硬编码：检查特定用户ID是否为管理员（开发测试用）
		tempAdminUserID := "fb3c2853-ee5f-4c70-9323-5d607acffaa3"
		if userID == tempAdminUserID {
			// 临时赋予管理员权限
			user.Role = domain.RoleSuper
		}
		
		// 检查是否为管理员
		if !user.IsAdmin() {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user", user)
		c.Set("role", user.Role)
		c.Next()
	}
}

// RequireSuper 要求超级管理员权限
func (a *AdminAuth) RequireSuper() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userIDVal, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			c.Abort()
			return
		}

		// 获取用户信息
		user, err := a.authService.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// 临时硬编码：检查特定用户ID是否为超级管理员（开发测试用）
		tempAdminUserID := "fb3c2853-ee5f-4c70-9323-5d607acffaa3"
		if userID == tempAdminUserID {
			// 临时赋予超级管理员权限
			user.Role = domain.RoleSuper
		}
		
		// 检查是否为超级管理员
		if !user.IsSuper() {
			c.JSON(http.StatusForbidden, gin.H{"error": "super admin access required"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user", user)
		c.Set("role", user.Role)
		c.Next()
	}
}

// RequireRole 要求特定角色
func (a *AdminAuth) RequireRole(allowedRoles ...domain.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userIDVal, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userID, ok := userIDVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
			c.Abort()
			return
		}

		// 获取用户信息
		user, err := a.authService.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		// 检查角色是否允许
		allowed := false
		for _, role := range allowedRoles {
			if user.Role == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user", user)
		c.Set("role", user.Role)
		c.Next()
	}
}

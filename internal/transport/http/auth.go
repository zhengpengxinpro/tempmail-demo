package httptransport

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tempmail/backend/internal/auth"
	jwtpkg "tempmail/backend/internal/auth/jwt"
)

// AuthHandler 处理认证相关的 HTTP 请求
type AuthHandler struct {
	authService *auth.Service   // 认证业务服务
	jwtManager  *jwtpkg.Manager // JWT 令牌管理器
	log         *zap.Logger     // 结构化日志记录器
}

// NewAuthHandler 创建新的认证处理器实例
//
// 参数:
//   - authService: 认证业务服务
//   - jwtManager: JWT 令牌管理器
//
// 返回值:
//   - *AuthHandler: 认证处理器实例
func NewAuthHandler(authService *auth.Service, jwtManager *jwtpkg.Manager) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtManager:  jwtManager,
		log:         zap.NewNop(), // 临时使用空日志
	}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Username string `json:"username"`
}

type loginRequest struct {
    Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	ExpiresIn    int64        `json:"expiresIn"`
}

type userResponse struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	Username        string `json:"username,omitempty"`
	Tier            string `json:"tier"`
	IsActive        bool   `json:"isActive"`
	IsEmailVerified bool   `json:"isEmailVerified"`
}

// Register 处理用户注册请求
// @Summary 用户注册
// @Description 创建新用户账户，返回用户信息和认证令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body registerRequest true "注册信息"
// @Success 201 {object} authResponse "注册成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 409 {object} Response "邮箱已存在"
// @Failure 500 {object} Response "服务器内部错误"
// @Router /v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	// 注册用户
	user, err := h.authService.Register(auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	})

	if err != nil {
		switch err {
		case auth.ErrInvalidEmail:
			BadRequest(c, "邮箱格式无效")
		case auth.ErrInvalidPassword:
			BadRequest(c, err.Error())
		case auth.ErrEmailExists:
			Conflict(c, "该邮箱已被注册")
		default:
			h.log.Error("failed to register user", zap.Error(err))
			InternalError(c, "注册失败，请稍后重试")
		}
		return
	}

	// 生成令牌
	tokens, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Tier))
	if err != nil {
		h.log.Error("failed to generate tokens", zap.Error(err))
		InternalError(c, "生成令牌失败")
		return
	}

	h.log.Info("user registered",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	Created(c, authResponse{
		User: userResponse{
			ID:              user.ID,
			Email:           user.Email,
			Username:        user.Username,
			Tier:            string(user.Tier),
			IsActive:        user.IsActive,
			IsEmailVerified: user.IsEmailVerified,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Login 处理用户登录请求
// @Summary 用户登录
// @Description 使用邮箱和密码进行身份验证，成功后返回认证令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body loginRequest true "登录凭证"
// @Success 200 {object} authResponse "登录成功"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 401 {object} Response "邮箱或密码错误"
// @Failure 403 {object} Response "账户已被禁用"
// @Failure 500 {object} Response "服务器内部错误"
// @Router /v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	// 验证登录
    user, err := h.authService.Login(auth.LoginInput{
        Identifier: strings.TrimSpace(req.Username),
        Password:   req.Password,
    })

	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			Unauthorized(c, MsgInvalidCredentials)
		case auth.ErrUserInactive:
			Forbidden(c, "账户已被禁用")
		default:
			h.log.Error("failed to login", zap.Error(err))
			InternalError(c, "登录失败，请稍后重试")
		}
		return
	}

	// 生成令牌
	tokens, err := h.jwtManager.GenerateTokenPair(user.ID, user.Email, string(user.Tier))
	if err != nil {
		h.log.Error("failed to generate tokens", zap.Error(err))
		InternalError(c, "生成令牌失败")
		return
	}

	h.log.Info("user logged in",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	Success(c, authResponse{
		User: userResponse{
			ID:              user.ID,
			Email:           user.Email,
			Username:        user.Username,
			Tier:            string(user.Tier),
			IsActive:        user.IsActive,
			IsEmailVerified: user.IsEmailVerified,
		},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// Refresh 刷新访问令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌，避免重新登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body refreshRequest true "包含刷新令牌的请求"
// @Success 200 {object} object{accessToken=string,expiresIn=int} "新的访问令牌"
// @Failure 400 {object} Response "请求参数错误"
// @Failure 401 {object} Response "刷新令牌无效或已过期"
// @Failure 500 {object} Response "服务器内部错误"
// @Router /v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	// 刷新访问令牌
	accessToken, err := h.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		switch err {
		case jwtpkg.ErrInvalidToken:
			Unauthorized(c, "刷新令牌无效")
		case jwtpkg.ErrExpiredToken:
			Unauthorized(c, MsgTokenExpired)
		default:
			h.log.Error("failed to refresh token", zap.Error(err))
			InternalError(c, "刷新令牌失败")
		}
		return
	}

	Success(c, gin.H{
		"accessToken": accessToken,
		"expiresIn":   int64(15 * 60), // 15 分钟
	})
}

// Me 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取已认证用户的详细信息，需要有效的访问令牌
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} userResponse "用户信息"
// @Failure 401 {object} Response "未认证或令牌无效"
// @Failure 404 {object} Response "用户不存在"
// @Failure 500 {object} Response "服务器内部错误"
// @Router /v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	// 从上下文中获取用户 ID（由认证中间件设置）
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	user, err := h.authService.GetUserByID(userID.(string))
	if err != nil {
		if err == auth.ErrUserNotFound {
			NotFound(c, MsgUserNotFound)
			return
		}
		h.log.Error("failed to get user", zap.Error(err))
		InternalError(c, MsgUserGetFailed)
		return
	}

	Success(c, userResponse{
		ID:              user.ID,
		Email:           user.Email,
		Username:        user.Username,
		Tier:            string(user.Tier),
		IsActive:        user.IsActive,
		IsEmailVerified: user.IsEmailVerified,
	})
}

// AuthMiddleware JWT 认证中间件
//
// 该中间件用于验证请求中的 JWT 令牌，并将用户信息注入到上下文中
//
// 参数:
//   - jwtManager: JWT 令牌管理器
//
// 返回值:
//   - gin.HandlerFunc: Gin 中间件函数
//
// 上下文注入:
//   - userID: 用户 ID
//   - email: 用户邮箱
//   - tier: 用户等级
func AuthMiddleware(jwtManager *jwtpkg.Manager) gin.HandlerFunc {
	log := zap.NewNop() // 临时使用空日志

	return func(c *gin.Context) {
		// 从 Authorization 头获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			Unauthorized(c, "缺少认证令牌")
			c.Abort()
			return
		}

		// 解析 Bearer 令牌
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			Unauthorized(c, "认证令牌格式错误")
			c.Abort()
			return
		}

		token := parts[1]

		// 验证令牌
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			switch err {
			case jwtpkg.ErrExpiredToken:
				Unauthorized(c, MsgTokenExpired)
			case jwtpkg.ErrInvalidToken:
				Unauthorized(c, MsgTokenInvalid)
			default:
				log.Error("failed to validate token", zap.Error(err))
				Unauthorized(c, "令牌验证失败")
			}
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("tier", claims.Tier)

		c.Next()
	}
}

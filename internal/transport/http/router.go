package httptransport

import (
	"fmt"
	"net/http"
	"time"

	gincors "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	"go.uber.org/zap"

	"tempmail/backend/internal/auth"
	jwtpkg "tempmail/backend/internal/auth/jwt"
	"tempmail/backend/internal/config"
	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/middleware"
	"tempmail/backend/internal/service"
	"tempmail/backend/internal/storage"
	"tempmail/backend/internal/storage/memory"
	"tempmail/backend/internal/websocket"
)

// Handler 聚合所有 HTTP 处理逻辑。
type Handler struct {
	mailboxes *service.MailboxService
	messages  *service.MessageService
	aliases   *service.AliasService
	search    *service.SearchService
	webhook   *service.WebhookService
	tag       *service.TagService
}

// RouterDependencies 路由器依赖项
type RouterDependencies struct {
	Config              *config.Config
	MailboxService      *service.MailboxService
	MessageService      *service.MessageService
	AliasService        *service.AliasService
	SearchService       *service.SearchService       // 添加搜索服务
	WebhookService      *service.WebhookService      // 添加 Webhook 服务
	TagService          *service.TagService          // 添加标签服务
	AuthService         *auth.Service
	AdminService        *service.AdminService        // 添加管理服务
	UserDomainService   *service.UserDomainService   // 添加用户域名服务
	SystemDomainService *service.SystemDomainService // 添加系统域名服务
	APIKeyService       *service.APIKeyService       // 添加API Key服务
	ConfigService       *service.ConfigService       // 添加系统配置服务
	JWTManager          *jwtpkg.Manager
	WebSocketHub        *websocket.Hub // WebSocket Hub
	Store               storage.Store  // 添加存储接口
	Logger              *zap.Logger    // 添加日志记录器
}

// NewRouter 创建并返回 Gin 路由实例。
func NewRouter(deps RouterDependencies) *gin.Engine {
	router := gin.New()

	// 使用自定义中间件替代默认中间件
	router.Use(middleware.RecoveryHandler())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.SecurityHeaders())

	// 使用新的请求体大小限制中间件
	// 设置全局默认限制为 10MB
	router.Use(middleware.BodySizeLimit(10 * 1024 * 1024))

	// CORS 配置
	corsConfig := gincors.Config{
		AllowOrigins: deps.Config.CORS.AllowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Mailbox-Token"},
		ExposeHeaders: []string{
			"Content-Length",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// 如果允许所有来源，则需清空凭证支持。
	for _, origin := range corsConfig.AllowOrigins {
		if origin == "*" {
			corsConfig.AllowCredentials = false
			break
		}
	}
	router.Use(gincors.New(corsConfig))

	// 创建处理器
	handler := &Handler{
		mailboxes: deps.MailboxService,
		messages:  deps.MessageService,
		aliases:   deps.AliasService,
		search:    deps.SearchService,
		webhook:   deps.WebhookService,
		tag:       deps.TagService,
	}

	authHandler := NewAuthHandler(deps.AuthService, deps.JWTManager)
	adminHandler := NewAdminHandler(deps.AdminService, deps.SystemDomainService)                                                       // 创建管理处理器
	userDomainHandler := NewUserDomainHandler(deps.UserDomainService)                                                                  // 创建用户域名处理器
	apiKeyHandler := NewAPIKeyHandler(deps.APIKeyService)                                                                              // 创建API Key处理器
	configHandler := NewConfigHandler(deps.ConfigService)                                                                              // 创建系统配置处理器
	compatHandler := NewCompatHandler(deps.MailboxService, deps.MessageService, deps.AliasService, deps.Config.Mailbox.AllowedDomains) // 创建兼容API处理器
	publicHandler := NewPublicHandler(deps.SystemDomainService)                                                                        // 创建公开API处理器

	// 创建中间件
	mailboxAuth := middleware.NewMailboxAuth(deps.MailboxService)
	jwtAuth := middleware.NewJWTAuth(deps.JWTManager)
	adminAuth := middleware.NewAdminAuth(deps.AuthService)     // 创建管理员中间件
	apiKeyAuth := middleware.NewAPIKeyAuth(deps.APIKeyService) // 创建API Key中间件

	// 限流中间件（临时禁用 - 开发环境）
	// rateLimitStore := deps.Store.(storage.RateLimitRepository)
	// ipRateLimit := middleware.RateLimitByIP(rateLimitStore, deps.Logger, 100, 1*time.Minute)
	// _ = middleware.RateLimitByUser(rateLimitStore, deps.Logger, 200, 1*time.Minute) // 暂时不使用
	
	// 邮箱创建限流：放宽限制以支持测试和开发
	// 压力测试：临时禁用限流
	// mailboxRateLimit := middleware.MailboxRateLimit(deps.Store.(storage.RateLimitRepository), deps.Logger, 50, 1*time.Hour)
	// messageRateLimit := middleware.MessageRateLimit(deps.Store.(storage.RateLimitRepository), deps.Logger, 1000, 1*time.Hour)

	// 防滥用中间件（临时禁用 - 开发环境）
	// abuseConfig := middleware.AbuseConfig{
	// 	Store:                 deps.Store,
	// 	Logger:                deps.Logger,
	// 	MaxMailboxesPerIP:     10,
	// 	MaxMessagesPerMailbox: 1000,
	// 	MaxAliasesPerMailbox:  5,
	// 	BlockDuration:         1 * time.Hour,
	// }
	// abusePrevention := middleware.AbusePrevention(abuseConfig)
	// contentFilter := middleware.ContentFilter(abuseConfig)
	// userAgentFilter := middleware.UserAgentFilter(abuseConfig)

	// Swagger 文档
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// V1 API
	v1 := router.Group("/v1")
	{
		// ========== Public Routes（无需认证的公开API） ==========
		publicRoutes := v1.Group("/public")
		{
			publicRoutes.GET("/domains", publicHandler.GetAvailableDomains) // 获取可用域名列表
			publicRoutes.GET("/config", publicHandler.GetSystemConfig)      // 获取系统配置
		}

		// 应用全局限流和防滥用中间件（临时禁用 - 开发环境）
		// v1.Use(ipRateLimit)
		// v1.Use(abusePrevention)
		// v1.Use(contentFilter)
		// v1.Use(userAgentFilter)

		// ========== Auth Routes ==========
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/register", authHandler.Register)
			authRoutes.POST("/login", authHandler.Login)
			authRoutes.POST("/refresh", authHandler.Refresh)
			authRoutes.GET("/me", jwtAuth.RequireAuth(), authHandler.Me)
		}

		// ========== Mailbox Routes ==========
		mailboxRoutes := v1.Group("/mailboxes")
		{
			// 邮箱创建限流
			mailboxRoutes.POST("", jwtAuth.OptionalAuth(), handler.createMailbox)
			mailboxRoutes.GET("", jwtAuth.OptionalAuth(), handler.listMailboxes)

			// 需要邮箱Token的端点
			mailboxRoutes.GET("/:id", mailboxAuth.RequireMailboxToken(), handler.getMailbox)
			mailboxRoutes.DELETE("/:id", mailboxAuth.RequireMailboxToken(), handler.deleteMailbox)

			// 邮件相关端点（需要邮箱Token）
			mailboxRoutes.POST("/:id/messages", mailboxAuth.RequireMailboxToken(), handler.createMessage)
			mailboxRoutes.GET("/:id/messages", mailboxAuth.RequireMailboxToken(), handler.listMessages)
			mailboxRoutes.GET("/:id/messages/:messageId", mailboxAuth.RequireMailboxToken(), handler.getMessage)
			mailboxRoutes.POST("/:id/messages/:messageId/read", mailboxAuth.RequireMailboxToken(), handler.markMessageRead)

			// 附件下载端点
			mailboxRoutes.GET("/:id/messages/:messageId/attachments/:attachmentId", mailboxAuth.RequireMailboxToken(), handler.downloadAttachment)

			// 邮件搜索端点
			mailboxRoutes.GET("/:id/messages/search", mailboxAuth.RequireMailboxToken(), handler.searchMessages)

			// 别名管理端点
			mailboxRoutes.POST("/:id/aliases", mailboxAuth.RequireMailboxToken(), handler.createAlias)
			mailboxRoutes.GET("/:id/aliases", mailboxAuth.RequireMailboxToken(), handler.listAliases)
			mailboxRoutes.GET("/:id/aliases/:aliasId", mailboxAuth.RequireMailboxToken(), handler.getAlias)
			mailboxRoutes.DELETE("/:id/aliases/:aliasId", mailboxAuth.RequireMailboxToken(), handler.deleteAlias)
			mailboxRoutes.PATCH("/:id/aliases/:aliasId", mailboxAuth.RequireMailboxToken(), handler.toggleAlias)

			// 邮件标签端点（需要邮箱Token）
			mailboxRoutes.POST("/:id/messages/:messageId/tags", mailboxAuth.RequireMailboxToken(), handler.addMessageTag)
			mailboxRoutes.GET("/:id/messages/:messageId/tags", mailboxAuth.RequireMailboxToken(), handler.getMessageTags)
			mailboxRoutes.DELETE("/:id/messages/:messageId/tags/:tagId", mailboxAuth.RequireMailboxToken(), handler.removeMessageTag)
		}

		// ========== WebSocket Routes ==========
		if deps.WebSocketHub != nil {
			v1.GET("/ws", websocket.HandleWebSocket(deps.WebSocketHub))
		}

		// ========== Admin Routes ==========
		adminRoutes := v1.Group("/admin")
		adminRoutes.Use(jwtAuth.RequireAuth()) // 所有管理路由都需要认证
		{
			// 用户管理（需要管理员权限）
			adminRoutes.GET("/users", adminAuth.RequireAdmin(), adminHandler.ListUsers)
			adminRoutes.GET("/users/:id", adminAuth.RequireAdmin(), adminHandler.GetUser)
			adminRoutes.PATCH("/users/:id", adminAuth.RequireAdmin(), adminHandler.UpdateUser)
			adminRoutes.DELETE("/users/:id", adminAuth.RequireSuper(), adminHandler.DeleteUser) // 超级管理员才能删除用户

			// 用户配额管理
			adminRoutes.GET("/users/:id/quota", adminAuth.RequireAdmin(), adminHandler.GetUserQuota)
			adminRoutes.PUT("/users/:id/quota", adminAuth.RequireAdmin(), adminHandler.UpdateUserQuota)

			// 系统域名管理
			adminRoutes.GET("/domains", adminAuth.RequireAdmin(), adminHandler.ListSystemDomains)            // 获取域名列表
			adminRoutes.POST("/domains", adminAuth.RequireSuper(), adminHandler.AddSystemDomain)            // 添加域名
			adminRoutes.POST("/domains/recover", adminAuth.RequireSuper(), adminHandler.RecoverSystemDomain) // 找回域名
			adminRoutes.GET("/domains/:id", adminAuth.RequireAdmin(), adminHandler.GetSystemDomain)          // 获取域名详情
			adminRoutes.POST("/domains/:id/verify", adminAuth.RequireAdmin(), adminHandler.VerifySystemDomain) // 验证域名
			adminRoutes.GET("/domains/:id/instructions", adminAuth.RequireAdmin(), adminHandler.GetSystemDomainInstructions) // 配置说明
			adminRoutes.PATCH("/domains/:id/toggle", adminAuth.RequireAdmin(), adminHandler.ToggleSystemDomainStatus)        // 切换状态
			adminRoutes.POST("/domains/:id/set-default", adminAuth.RequireSuper(), adminHandler.SetDefaultSystemDomain)      // 设置默认域名
			adminRoutes.DELETE("/domains/:id", adminAuth.RequireSuper(), adminHandler.DeleteSystemDomain)    // 删除域名

			// 系统统计
			adminRoutes.GET("/statistics", adminAuth.RequireAdmin(), adminHandler.GetStatistics)

			// 系统配置管理（需要管理员权限）
			adminRoutes.GET("/config", adminAuth.RequireAdmin(), configHandler.GetSystemConfig)           // 获取系统配置
			adminRoutes.PUT("/config", adminAuth.RequireSuper(), configHandler.UpdateSystemConfig)        // 更新系统配置（超级管理员）
			adminRoutes.POST("/config/reset", adminAuth.RequireSuper(), configHandler.ResetSystemConfig) // 重置系统配置（超级管理员）
		}

		// ========== User Domain Routes ==========
		userDomainRoutes := v1.Group("/user/domains")
		userDomainRoutes.Use(jwtAuth.RequireAuth()) // 所有用户域名路由都需要认证
		{
			userDomainRoutes.POST("", userDomainHandler.AddDomain)                            // 添加域名
			userDomainRoutes.GET("", userDomainHandler.ListDomains)                           // 域名列表
			userDomainRoutes.GET("/:id", userDomainHandler.GetDomain)                         // 域名详情
			userDomainRoutes.POST("/:id/verify", userDomainHandler.VerifyDomain)              // 验证域名
			userDomainRoutes.GET("/:id/instructions", userDomainHandler.GetSetupInstructions) // 配置说明
			userDomainRoutes.PATCH("/:id", userDomainHandler.UpdateDomainMode)                // 更新模式
			userDomainRoutes.DELETE("/:id", userDomainHandler.DeleteDomain)                   // 删除域名
		}

		// ========== Webhook Routes ==========
		if deps.WebhookService != nil {
			webhookRoutes := v1.Group("/webhooks")
			webhookRoutes.Use(jwtAuth.RequireAuth()) // 需要认证
			{
				webhookRoutes.POST("", handler.createWebhook)                       // 创建 Webhook
				webhookRoutes.GET("", handler.listWebhooks)                         // 列出 Webhooks
				webhookRoutes.GET("/:id", handler.getWebhook)                       // 获取 Webhook
				webhookRoutes.PATCH("/:id", handler.updateWebhook)                  // 更新 Webhook
				webhookRoutes.DELETE("/:id", handler.deleteWebhook)                 // 删除 Webhook
				webhookRoutes.GET("/:id/deliveries", handler.getWebhookDeliveries) // 获取投递记录
			}
		}

		// ========== Tag Routes ==========
		if deps.TagService != nil {
			tagRoutes := v1.Group("/tags")
			tagRoutes.Use(jwtAuth.RequireAuth()) // 需要认证
			{
				tagRoutes.POST("", handler.createTag)              // 创建标签
				tagRoutes.GET("", handler.listTags)                // 列出标签
				tagRoutes.GET("/:id", handler.getTag)              // 获取标签
				tagRoutes.PATCH("/:id", handler.updateTag)         // 更新标签
				tagRoutes.DELETE("/:id", handler.deleteTag)        // 删除标签
				tagRoutes.GET("/:id/messages", handler.listMessagesByTag) // 按标签列出邮件
			}
		}

		// ========== API Key Routes ==========
		apiKeyRoutes := v1.Group("/api-keys")
		apiKeyRoutes.Use(jwtAuth.RequireAuth()) // 所有API Key路由都需要JWT认证
		{
			apiKeyRoutes.POST("", apiKeyHandler.CreateAPIKey)       // 创建API Key
			apiKeyRoutes.GET("", apiKeyHandler.ListAPIKeys)         // 列出API Keys
			apiKeyRoutes.GET("/:id", apiKeyHandler.GetAPIKey)       // 获取API Key详情
			apiKeyRoutes.DELETE("/:id", apiKeyHandler.DeleteAPIKey) // 删除API Key
		}
	}

	// ========== Compatibility API (兼容层) ==========
	// 提供兼容 mail.ry.edu.kg API 格式的端点
	apiRoutes := router.Group("/api")
	apiRoutes.Use(apiKeyAuth.RequireAPIKey()) // 所有API路由都需要API Key认证
	{
		apiRoutes.GET("/config", compatHandler.GetConfig)                      // 获取系统配置
		apiRoutes.POST("/emails/generate", compatHandler.GenerateEmail)        // 生成临时邮箱
		apiRoutes.GET("/emails", compatHandler.ListEmails)                     // 获取邮箱列表
		apiRoutes.GET("/emails/:emailId", compatHandler.ListMessages)          // 获取邮件列表
		apiRoutes.GET("/emails/:emailId/:messageId", compatHandler.GetMessage) // 获取单封邮件
	}

	return router
}

type createMailboxRequest struct {
	Prefix    string `json:"prefix"`
	Domain    string `json:"domain"`
	ExpiresIn string `json:"expiresIn"`
}

type mailboxResponse struct {
	ID        string     `json:"id"`
	Address   string     `json:"address"`
	LocalPart string     `json:"localPart"`
	Domain    string     `json:"domain"`
	Token     string     `json:"token"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Unread    int        `json:"unread"`
	Total     int        `json:"total"`
}

type mailboxListResponse struct {
	Items []mailboxResponse `json:"items"`
	Count int               `json:"count"`
}

// createMailbox godoc
// @Summary 创建临时邮箱
// @Description 创建一个新的临时邮箱地址
// @Tags Mailboxes
// @Accept json
// @Produce json
// @Param request body createMailboxRequest true "邮箱参数"
// @Success 201 {object} mailboxResponse
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes [post]
func (h *Handler) createMailbox(c *gin.Context) {
	var req createMailboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		d, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			BadRequest(c, MsgInvalidDuration)
			return
		}
		t := time.Now().Add(d)
		expiresAt = &t
	}

	// 提取用户ID（如果已认证）
	var userID *string
	if userIDVal, exists := c.Get("userID"); exists {
		if uid, ok := userIDVal.(string); ok {
			userID = &uid
		}
	}

	mailbox, err := h.mailboxes.Create(service.CreateMailboxInput{
		Prefix:    req.Prefix,
		Domain:    req.Domain,
		IPSource:  c.ClientIP(),
		UserID:    userID, // 关联用户ID（游客模式为nil）
		ExpiresAt: expiresAt,
	})
	if err != nil {
		switch err {
		case service.ErrDomainNotAllowed, service.ErrPrefixInvalid:
			BadRequest(c, GetErrorMessage(err))
		default:
			InternalError(c, MsgMailboxCreateFailed)
		}
		return
	}

	Created(c, toMailboxResponse(mailbox))
}

// listMailboxes godoc
// @Summary 获取邮箱列表
// @Description 返回当前用户的临时邮箱列表（认证用户）或所有邮箱（游客）
// @Tags Mailboxes
// @Produce json
// @Success 200 {object} mailboxListResponse
// @Router /v1/mailboxes [get]
func (h *Handler) listMailboxes(c *gin.Context) {
	var mailboxes []domain.Mailbox

	// 如果用户已认证，只返回该用户的邮箱
	if userIDVal, exists := c.Get("userID"); exists {
		if userID, ok := userIDVal.(string); ok {
			mailboxes = h.mailboxes.ListByUserID(userID)
		} else {
			mailboxes = h.mailboxes.List()
		}
	} else {
		// 未认证用户：返回所有邮箱（包括游客邮箱）
		mailboxes = h.mailboxes.List()
	}

	responses := make([]mailboxResponse, 0, len(mailboxes))
	for i := range mailboxes {
		responses = append(responses, toMailboxResponse(&mailboxes[i]))
	}

	Success(c, mailboxListResponse{
		Items: responses,
		Count: len(responses),
	})
}

// getMailbox godoc
// @Summary 获取邮箱详情
// @Description 根据邮箱 ID 查看详细信息
// @Tags Mailboxes
// @Produce json
// @Param id path string true "邮箱ID"
// @Success 200 {object} mailboxResponse
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id} [get]
func (h *Handler) getMailbox(c *gin.Context) {
	// mailbox 已经由中间件验证并存储在上下文中
	mailboxInterface, _ := c.Get("mailbox")
	mailbox := mailboxInterface.(*domain.Mailbox)
	Success(c, toMailboxResponse(mailbox))
}

// deleteMailbox godoc
// @Summary 删除临时邮箱
// @Description 删除指定 ID 的邮箱及其邮件
// @Tags Mailboxes
// @Param id path string true "邮箱ID"
// @Success 204
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id} [delete]
func (h *Handler) deleteMailbox(c *gin.Context) {
	mailboxID := c.Param("id")
	err := h.mailboxes.Delete(mailboxID)
	if err != nil {
		if err == memory.ErrMailboxNotFound {
			NotFound(c, MsgMailboxNotFound)
		} else {
			InternalError(c, MsgMailboxDeleteFailed)
		}
		return
	}
	NoContent(c)
}

type createMessageRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
	Raw     string `json:"raw"`
	IsRead  bool   `json:"isRead"`
}

type attachmentInfo struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type messageResponse struct {
	ID          string           `json:"id"`
	MailboxID   string           `json:"mailboxId"`
	From        string           `json:"from"`
	To          string           `json:"to"`
	Subject     string           `json:"subject"`
	Text        string           `json:"text"`
	HTML        string           `json:"html"`
	IsRead      bool             `json:"isRead"`
	CreatedAt   time.Time        `json:"createdAt"`
	ReceivedAt  time.Time        `json:"receivedAt"`
	Attachments []attachmentInfo `json:"attachments,omitempty"` // 附件列表（不包含内容）
}

type messageListResponse struct {
	Items []messageResponse `json:"items"`
	Count int               `json:"count"`
}

// createMessage godoc
// @Summary 写入邮件
// @Description 在指定邮箱下新增一封邮件
// @Tags Messages
// @Accept json
// @Produce json
// @Param id path string true "邮箱ID"
// @Param request body createMessageRequest true "邮件内容"
// @Success 201 {object} messageResponse
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/messages [post]
func (h *Handler) createMessage(c *gin.Context) {
	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	mailboxID := c.Param("id")
	message, err := h.messages.Create(service.CreateMessageInput{
		MailboxID: mailboxID,
		From:      req.From,
		To:        req.To,
		Subject:   req.Subject,
		Text:      req.Text,
		HTML:      req.HTML,
		Raw:       req.Raw,
		IsRead:    req.IsRead,
	})
	if err != nil {
		if err == memory.ErrMailboxNotFound {
			NotFound(c, MsgMailboxNotFound)
			return
		}
		InternalError(c, MsgMessageCreateFailed)
		return
	}

	Created(c, toMessageResponse(message))
}

// listMessages godoc
// @Summary 获取邮件列表
// @Description 返回邮箱内的全部邮件
// @Tags Messages
// @Produce json
// @Param id path string true "邮箱ID"
// @Success 200 {object} messageListResponse
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/messages [get]
func (h *Handler) listMessages(c *gin.Context) {
	messages, err := h.messages.List(c.Param("id"))
	if err != nil {
		if err == memory.ErrMailboxNotFound {
			NotFound(c, MsgMailboxNotFound)
			return
		}
		InternalError(c, MsgMessageListFailed)
		return
	}

	responses := make([]messageResponse, 0, len(messages))
	for i := range messages {
		msg := messages[i]
		responses = append(responses, toMessageResponse(&msg))
	}

	Success(c, messageListResponse{
		Items: responses,
		Count: len(responses),
	})
}

// getMessage godoc
// @Summary 获取邮件详情
// @Description 查看单封邮件内容
// @Tags Messages
// @Produce json
// @Param id path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Success 200 {object} messageResponse
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/messages/{messageId} [get]
func (h *Handler) getMessage(c *gin.Context) {
	msg, err := h.messages.Get(c.Param("id"), c.Param("messageId"))
	if err != nil {
		if err == memory.ErrMessageNotFound {
			NotFound(c, MsgMessageNotFound)
			return
		}
		InternalError(c, MsgInternalError)
		return
	}

	Success(c, toMessageResponse(msg))
}

// markMessageRead godoc
// @Summary 标记邮件已读
// @Description 将指定邮件更新为已读状态
// @Tags Messages
// @Param id path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Success 204
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/messages/{messageId}/read [post]
func (h *Handler) markMessageRead(c *gin.Context) {
	err := h.messages.MarkRead(c.Param("id"), c.Param("messageId"))
	if err != nil {
		if err == memory.ErrMessageNotFound {
			NotFound(c, MsgMessageNotFound)
		} else {
			InternalError(c, MsgMessageMarkReadFailed)
		}
		return
	}
	NoContent(c)
}

// toMailboxResponse 转换实体为响应体。
func toMailboxResponse(mailbox *domain.Mailbox) mailboxResponse {
	return mailboxResponse{
		ID:        mailbox.ID,
		Address:   mailbox.Address,
		LocalPart: mailbox.LocalPart,
		Domain:    mailbox.Domain,
		Token:     mailbox.Token,
		CreatedAt: mailbox.CreatedAt,
		ExpiresAt: mailbox.ExpiresAt,
		Unread:    mailbox.Unread,
		Total:     mailbox.TotalCount,
	}
}

// toMessageResponse 转换邮件实体为响应体。
func toMessageResponse(message *domain.Message) messageResponse {
	// 转换附件信息（不包含内容）
	attachments := make([]attachmentInfo, 0, len(message.Attachments))
	for _, att := range message.Attachments {
		attachments = append(attachments, attachmentInfo{
			ID:          att.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
		})
	}

	return messageResponse{
		ID:          message.ID,
		MailboxID:   message.MailboxID,
		From:        message.From,
		To:          message.To,
		Subject:     message.Subject,
		Text:        message.Text,
		HTML:        message.HTML,
		IsRead:      message.IsRead,
		CreatedAt:   message.CreatedAt,
		ReceivedAt:  message.ReceivedAt,
		Attachments: attachments,
	}
}

// downloadAttachment godoc
// @Summary 下载附件
// @Description 下载邮件的附件文件
// @Tags Messages
// @Produce application/octet-stream
// @Param id path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Param attachmentId path string true "附件ID"
// @Success 200 {file} binary
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/messages/{messageId}/attachments/{attachmentId} [get]
func (h *Handler) downloadAttachment(c *gin.Context) {
	mailboxID := c.Param("id")
	messageID := c.Param("messageId")
	attachmentID := c.Param("attachmentId")

	// 获取附件
	attachment, err := h.messages.GetAttachment(mailboxID, messageID, attachmentID)
	if err != nil {
		if err == memory.ErrMessageNotFound {
			NotFound(c, MsgMessageNotFound)
			return
		}
		NotFound(c, MsgAttachmentNotFound)
		return
	}

	// 附件下载不使用统一响应格式，直接返回二进制流
	c.Header("Content-Type", attachment.ContentType)
	c.Header("Content-Disposition", "attachment; filename=\""+attachment.Filename+"\"")
	c.Header("Content-Length", fmt.Sprintf("%d", attachment.Size))
	c.Data(http.StatusOK, attachment.ContentType, attachment.Content)
}

// searchMessages godoc
// @Summary 搜索邮件
// @Description 在指定邮箱中搜索邮件
// @Tags Messages
// @Accept json
// @Produce json
// @Param id path string true "邮箱ID"
// @Param q query string false "搜索关键词（搜索主题、发件人、内容）"
// @Param from query string false "发件人筛选"
// @Param subject query string false "主题筛选"
// @Param startDate query string false "开始日期 (RFC3339格式)"
// @Param endDate query string false "结束日期 (RFC3339格式)"
// @Param isRead query boolean false "是否已读"
// @Param hasAttachment query boolean false "是否有附件"
// @Param page query int false "页码（默认1）"
// @Param pageSize query int false "每页数量（默认20，最大100）"
// @Success 200 {object} Response{data=domain.MessageSearchResult}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/messages/search [get]
func (h *Handler) searchMessages(c *gin.Context) {
	mailboxID := c.Param("id")

	// 解析查询参数
	var input struct {
		Query         string `form:"q"`
		From          string `form:"from"`
		Subject       string `form:"subject"`
		StartDate     string `form:"startDate"`
		EndDate       string `form:"endDate"`
		IsRead        *bool  `form:"isRead"`
		HasAttachment *bool  `form:"hasAttachment"`
		Page          int    `form:"page"`
		PageSize      int    `form:"pageSize"`
	}

	if err := c.ShouldBindQuery(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "无效的查询参数",
		})
		return
	}

	// 解析时间参数
	var startDate, endDate *time.Time
	if input.StartDate != "" {
		t, err := time.Parse(time.RFC3339, input.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "开始日期格式无效，请使用 RFC3339 格式",
			})
			return
		}
		startDate = &t
	}
	if input.EndDate != "" {
		t, err := time.Parse(time.RFC3339, input.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "结束日期格式无效，请使用 RFC3339 格式",
			})
			return
		}
		endDate = &t
	}

	// 执行搜索
	result, err := h.search.SearchMessages(service.SearchMessagesInput{
		MailboxID:     mailboxID,
		Query:         input.Query,
		From:          input.From,
		Subject:       input.Subject,
		StartDate:     startDate,
		EndDate:       endDate,
		IsRead:        input.IsRead,
		HasAttachment: input.HasAttachment,
		Page:          input.Page,
		PageSize:      input.PageSize,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "搜索失败",
		})
		return
	}

	Success(c, result)
}

// ========== Alias Handlers ==========

// createAlias godoc
// @Summary 创建邮箱别名
// @Description 为邮箱创建一个新的别名地址
// @Tags Aliases
// @Accept json
// @Produce json
// @Param id path string true "邮箱ID"
// @Param body body object true "别名信息"
// @Success 201 {object} domain.MailboxAlias
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/aliases [post]
func (h *Handler) createAlias(c *gin.Context) {
	mailboxID := c.Param("id")

	var req struct {
		Address string `json:"address" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	alias, err := h.aliases.Create(service.CreateAliasInput{
		MailboxID: mailboxID,
		Address:   req.Address,
	})

	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Created(c, alias)
}

// listAliases godoc
// @Summary 列出邮箱别名
// @Description 获取邮箱的所有别名列表
// @Tags Aliases
// @Produce json
// @Param id path string true "邮箱ID"
// @Success 200 {object} object{items=[]domain.MailboxAlias,count=int}
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/aliases [get]
func (h *Handler) listAliases(c *gin.Context) {
	mailboxID := c.Param("id")

	aliases, err := h.aliases.List(mailboxID)
	if err != nil {
		InternalError(c, MsgAliasListFailed)
		return
	}

	Success(c, gin.H{
		"items": aliases,
		"count": len(aliases),
	})
}

// getAlias godoc
// @Summary 获取别名详情
// @Description 获取指定别名的详细信息
// @Tags Aliases
// @Produce json
// @Param id path string true "邮箱ID"
// @Param aliasId path string true "别名ID"
// @Success 200 {object} domain.MailboxAlias
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/aliases/{aliasId} [get]
func (h *Handler) getAlias(c *gin.Context) {
	aliasID := c.Param("aliasId")

	alias, err := h.aliases.Get(aliasID)
	if err != nil {
		NotFound(c, MsgAliasNotFound)
		return
	}

	Success(c, alias)
}

// deleteAlias godoc
// @Summary 删除别名
// @Description 删除指定的邮箱别名
// @Tags Aliases
// @Produce json
// @Param id path string true "邮箱ID"
// @Param aliasId path string true "别名ID"
// @Success 204
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/aliases/{aliasId} [delete]
func (h *Handler) deleteAlias(c *gin.Context) {
	mailboxID := c.Param("id")
	aliasID := c.Param("aliasId")

	if err := h.aliases.Delete(mailboxID, aliasID); err != nil {
		InternalError(c, MsgAliasDeleteFailed)
		return
	}

	NoContent(c)
}

// toggleAlias godoc
// @Summary 切换别名状态
// @Description 启用或禁用邮箱别名
// @Tags Aliases
// @Accept json
// @Produce json
// @Param id path string true "邮箱ID"
// @Param aliasId path string true "别名ID"
// @Param body body object true "状态信息"
// @Success 200 {object} domain.MailboxAlias
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /v1/mailboxes/{id}/aliases/{aliasId} [patch]
func (h *Handler) toggleAlias(c *gin.Context) {
	mailboxID := c.Param("id")
	aliasID := c.Param("aliasId")

	var req struct {
		IsActive bool `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	if err := h.aliases.Toggle(mailboxID, aliasID, req.IsActive); err != nil {
		InternalError(c, MsgAliasToggleFailed)
		return
	}

	// 返回更新后的别名
	alias, err := h.aliases.Get(aliasID)
	if err != nil {
		NotFound(c, MsgAliasNotFound)
		return
	}

	Success(c, alias)
}

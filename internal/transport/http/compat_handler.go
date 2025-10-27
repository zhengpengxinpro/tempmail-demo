package httptransport

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/service"
	"tempmail/backend/internal/storage/memory"
)

// 注意：兼容API全部使用旧格式（直接返回数据），与 mail.ry.edu.kg 完全兼容

// errorResponse 兼容API错误响应（旧格式）
type errorResponse struct {
	Error string `json:"error"`
}

// CompatHandler 兼容API处理器
type CompatHandler struct {
	mailboxes *service.MailboxService
	messages  *service.MessageService
	aliases   *service.AliasService
	config    *compatConfig
}

// compatConfig 兼容API配置
type compatConfig struct {
	AllowedDomains []string
}

// NewCompatHandler 创建兼容API处理器
func NewCompatHandler(
	mailboxService *service.MailboxService,
	messageService *service.MessageService,
	aliasService *service.AliasService,
	allowedDomains []string,
) *CompatHandler {
	return &CompatHandler{
		mailboxes: mailboxService,
		messages:  messageService,
		aliases:   aliasService,
		config: &compatConfig{
			AllowedDomains: allowedDomains,
		},
	}
}

// ========== 响应结构体 ==========

type configResponse struct {
	Domains []string `json:"domains"` // 可用的邮箱域名列表
}

type generateEmailRequest struct {
	Name       string `json:"name"`       // 邮箱前缀
	ExpiryTime int64  `json:"expiryTime"` // 过期时间（毫秒）：3600000(1h), 86400000(1d), 604800000(7d), 0(永久)
	Domain     string `json:"domain"`     // 邮箱域名
}

type generateEmailResponse struct {
	EmailID   string    `json:"emailId"`   // 邮箱ID
	Email     string    `json:"email"`     // 完整邮箱地址
	Name      string    `json:"name"`      // 邮箱前缀
	Domain    string    `json:"domain"`    // 域名
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	ExpiresAt time.Time `json:"expiresAt"` // 过期时间
	Token     string    `json:"token"`     // 访问令牌
}

type emailListResponse struct {
	Emails     []emailItem `json:"emails"`
	NextCursor *string     `json:"nextCursor,omitempty"` // 下一页游标
}

type emailItem struct {
	EmailID   string    `json:"emailId"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type compatMessageListResponse struct {
	Messages   []messageItem `json:"messages"`
	NextCursor *string       `json:"nextCursor,omitempty"` // 下一页游标
}

type messageItem struct {
	MessageID     string    `json:"messageId"`
	From          string    `json:"from"`
	Subject       string    `json:"subject"`
	CreatedAt     time.Time `json:"createdAt"`
	IsRead        bool      `json:"isRead"`
	HasAttachment bool      `json:"hasAttachment"`
}

type messageDetailResponse struct {
	MessageID   string           `json:"messageId"`
	EmailID     string           `json:"emailId"`
	From        string           `json:"from"`
	To          string           `json:"to"`
	Subject     string           `json:"subject"`
	Text        string           `json:"text"`
	HTML        string           `json:"html"`
	CreatedAt   time.Time        `json:"createdAt"`
	IsRead      bool             `json:"isRead"`
	Attachments []attachmentItem `json:"attachments,omitempty"`
}

type attachmentItem struct {
	AttachmentID string `json:"attachmentId"`
	Filename     string `json:"filename"`
	ContentType  string `json:"contentType"`
	Size         int64  `json:"size"`
}

// ========== API 处理器 ==========

// GetConfig 获取系统配置
// @Summary 获取系统配置
// @Description 获取系统配置，包括可用的邮箱域名列表
// @Tags Compat
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} configResponse
// @Router /api/config [get]
func (h *CompatHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, configResponse{
		Domains: h.config.AllowedDomains,
	})
}

// GenerateEmail 生成临时邮箱
// @Summary 生成临时邮箱
// @Description 创建一个新的临时邮箱
// @Tags Compat
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body generateEmailRequest true "邮箱参数"
// @Success 200 {object} generateEmailResponse
// @Failure 400 {object} errorResponse
// @Router /api/emails/generate [post]
func (h *CompatHandler) GenerateEmail(c *gin.Context) {
	var req generateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request"})
		return
	}

	// 计算过期时间
	var expiresAt *time.Time
	if req.ExpiryTime > 0 {
		t := time.Now().Add(time.Duration(req.ExpiryTime) * time.Millisecond)
		expiresAt = &t
	}

	// 提取用户ID（从API Key中间件设置）
	var userID *string
	if userIDVal, exists := c.Get("userID"); exists {
		if uid, ok := userIDVal.(string); ok {
			userID = &uid
		}
	}

	// 创建邮箱
	mailbox, err := h.mailboxes.Create(service.CreateMailboxInput{
		Prefix:    req.Name,
		Domain:    req.Domain,
		IPSource:  c.ClientIP(),
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		switch err {
		case service.ErrDomainNotAllowed:
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid domain"})
		case service.ErrPrefixInvalid:
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid name"})
		default:
			c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to create mailbox"})
		}
		return
	}

	// 构造响应
	resp := generateEmailResponse{
		EmailID:   mailbox.ID,
		Email:     mailbox.Address,
		Name:      mailbox.LocalPart,
		Domain:    mailbox.Domain,
		CreatedAt: mailbox.CreatedAt,
		Token:     mailbox.Token,
	}

	if mailbox.ExpiresAt != nil {
		resp.ExpiresAt = *mailbox.ExpiresAt
	} else {
		// 如果没有设置过期时间，返回一个很远的时间（100年后）
		resp.ExpiresAt = time.Now().Add(100 * 365 * 24 * time.Hour)
	}

	c.JSON(http.StatusOK, resp)
}

// ListEmails 获取邮箱列表
// @Summary 获取邮箱列表
// @Description 获取用户的邮箱列表，支持分页
// @Tags Compat
// @Produce json
// @Security ApiKeyAuth
// @Param cursor query string false "分页游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} emailListResponse
// @Router /api/emails [get]
func (h *CompatHandler) ListEmails(c *gin.Context) {
	// 获取用户ID
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}
	userID := userIDVal.(string)

	// 获取分页参数
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 获取邮箱列表
	mailboxes := h.mailboxes.ListByUserID(userID)

	// 简单分页实现（基于偏移量）
	offset := 0
	if cursor != "" {
		offset, _ = strconv.Atoi(cursor)
	}

	// 分页
	end := offset + limit
	if end > len(mailboxes) {
		end = len(mailboxes)
	}

	items := make([]emailItem, 0)
	if offset < len(mailboxes) {
		for i := offset; i < end; i++ {
			mb := &mailboxes[i]
			item := emailItem{
				EmailID:   mb.ID,
				Email:     mb.Address,
				Name:      mb.LocalPart,
				Domain:    mb.Domain,
				CreatedAt: mb.CreatedAt,
			}
			if mb.ExpiresAt != nil {
				item.ExpiresAt = *mb.ExpiresAt
			} else {
				item.ExpiresAt = time.Now().Add(100 * 365 * 24 * time.Hour)
			}
			items = append(items, item)
		}
	}

	// 构造响应
	resp := emailListResponse{
		Emails: items,
	}

	// 如果还有更多数据，返回下一页游标
	if end < len(mailboxes) {
		nextCursor := strconv.Itoa(end)
		resp.NextCursor = &nextCursor
	}

	c.JSON(http.StatusOK, resp)
}

// ListMessages 获取邮件列表
// @Summary 获取邮件列表
// @Description 获取指定邮箱的邮件列表，支持分页
// @Tags Compat
// @Produce json
// @Security ApiKeyAuth
// @Param emailId path string true "邮箱ID"
// @Param cursor query string false "分页游标"
// @Param limit query int false "每页数量" default(20)
// @Success 200 {object} compatMessageListResponse
// @Failure 404 {object} errorResponse
// @Router /api/emails/{emailId} [get]
func (h *CompatHandler) ListMessages(c *gin.Context) {
	emailID := c.Param("emailId")

	// 获取分页参数
	cursor := c.Query("cursor")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 获取邮件列表
	messages, err := h.messages.List(emailID)
	if err != nil {
		if err == memory.ErrMailboxNotFound {
			c.JSON(http.StatusNotFound, errorResponse{Error: "mailbox not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to list messages"})
		return
	}

	// 简单分页实现
	offset := 0
	if cursor != "" {
		offset, _ = strconv.Atoi(cursor)
	}

	end := offset + limit
	if end > len(messages) {
		end = len(messages)
	}

	items := make([]messageItem, 0)
	if offset < len(messages) {
		for i := offset; i < end; i++ {
			msg := &messages[i]
			items = append(items, messageItem{
				MessageID:     msg.ID,
				From:          msg.From,
				Subject:       msg.Subject,
				CreatedAt:     msg.CreatedAt,
				IsRead:        msg.IsRead,
				HasAttachment: len(msg.Attachments) > 0,
			})
		}
	}

	resp := compatMessageListResponse{
		Messages: items,
	}

	// 如果还有更多数据，返回下一页游标
	if end < len(messages) {
		nextCursor := strconv.Itoa(end)
		resp.NextCursor = &nextCursor
	}

	c.JSON(http.StatusOK, resp)
}

// GetMessage 获取单封邮件
// @Summary 获取单封邮件
// @Description 获取邮件的详细内容
// @Tags Compat
// @Produce json
// @Security ApiKeyAuth
// @Param emailId path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Success 200 {object} messageDetailResponse
// @Failure 404 {object} errorResponse
// @Router /api/emails/{emailId}/{messageId} [get]
func (h *CompatHandler) GetMessage(c *gin.Context) {
	emailID := c.Param("emailId")
	messageID := c.Param("messageId")

	// 获取邮件
	msg, err := h.messages.Get(emailID, messageID)
	if err != nil {
		if err == memory.ErrMessageNotFound {
			c.JSON(http.StatusNotFound, errorResponse{Error: "message not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "failed to get message"})
		return
	}

	// 自动标记为已读
	_ = h.messages.MarkRead(emailID, messageID)

	// 转换附件信息
	attachments := make([]attachmentItem, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, attachmentItem{
			AttachmentID: att.ID,
			Filename:     att.Filename,
			ContentType:  att.ContentType,
			Size:         att.Size,
		})
	}

	resp := messageDetailResponse{
		MessageID:   msg.ID,
		EmailID:     msg.MailboxID,
		From:        msg.From,
		To:          msg.To,
		Subject:     msg.Subject,
		Text:        msg.Text,
		HTML:        msg.HTML,
		CreatedAt:   msg.CreatedAt,
		IsRead:      true, // 已标记为已读
		Attachments: attachments,
	}

	c.JSON(http.StatusOK, resp)
}

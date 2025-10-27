package httptransport

import (
	"time"

	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/service"
)

// APIKeyHandler API Key管理处理器
type APIKeyHandler struct {
	apiKeyService *service.APIKeyService
}

// NewAPIKeyHandler 创建API Key处理器
func NewAPIKeyHandler(apiKeyService *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// createAPIKeyRequest 创建API Key请求
type createAPIKeyRequest struct {
	Name      string `json:"name" binding:"required"` // API Key名称/描述
	ExpiresIn string `json:"expiresIn,omitempty"`     // 过期时间（如 "720h" 表示30天）
}

// apiKeyResponse API Key响应
type apiKeyResponse struct {
	ID         string     `json:"id"`
	Key        string     `json:"key"`
	Name       string     `json:"name"`
	IsActive   bool       `json:"isActive"`
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

// CreateAPIKey godoc
// @Summary 创建API Key
// @Description 为当前用户创建一个新的API Key用于第三方对接
// @Tags APIKeys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createAPIKeyRequest true "API Key参数"
// @Success 201 {object} apiKeyResponse
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 500 {object} Response
// @Router /v1/api-keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	var req createAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	// 解析过期时间
	var expiresIn *time.Duration
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			BadRequest(c, "过期时间格式错误")
			return
		}
		expiresIn = &duration
	}

	// 创建API Key
	apiKey, err := h.apiKeyService.CreateAPIKey(service.CreateAPIKeyInput{
		UserID:    userID.(string),
		Name:      req.Name,
		ExpiresIn: expiresIn,
	})

	if err != nil {
		InternalError(c, MsgAPIKeyCreateFailed)
		return
	}

	Created(c, apiKeyResponse{
		ID:         apiKey.ID,
		Key:        apiKey.Key,
		Name:       apiKey.Name,
		IsActive:   apiKey.IsActive,
		CreatedAt:  apiKey.CreatedAt,
		ExpiresAt:  apiKey.ExpiresAt,
		LastUsedAt: apiKey.LastUsedAt,
	})
}

// ListAPIKeys godoc
// @Summary 获取API Key列表
// @Description 获取当前用户的所有API Key
// @Tags APIKeys
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object{items=[]apiKeyResponse,count=int}
// @Failure 401 {object} Response
// @Failure 500 {object} Response
// @Router /v1/api-keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	// 获取API Key列表
	apiKeys, err := h.apiKeyService.ListAPIKeys(userID.(string))
	if err != nil {
		InternalError(c, MsgAPIKeyListFailed)
		return
	}

	// 转换响应（隐藏实际的key值，安全起见）
	items := make([]apiKeyResponse, 0, len(apiKeys))
	for _, key := range apiKeys {
		items = append(items, apiKeyResponse{
			ID:         key.ID,
			Key:        maskAPIKey(key.Key), // 脱敏显示
			Name:       key.Name,
			IsActive:   key.IsActive,
			CreatedAt:  key.CreatedAt,
			ExpiresAt:  key.ExpiresAt,
			LastUsedAt: key.LastUsedAt,
		})
	}

	Success(c, gin.H{
		"items": items,
		"count": len(items),
	})
}

// GetAPIKey godoc
// @Summary 获取API Key详情
// @Description 获取指定API Key的详细信息
// @Tags APIKeys
// @Produce json
// @Security BearerAuth
// @Param id path string true "API Key ID"
// @Success 200 {object} apiKeyResponse
// @Failure 401 {object} Response
// @Failure 404 {object} Response
// @Router /v1/api-keys/{id} [get]
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	keyID := c.Param("id")

	// 获取API Key
	apiKey, err := h.apiKeyService.GetAPIKey(keyID)
	if err != nil {
		NotFound(c, MsgAPIKeyNotFound)
		return
	}

	// 验证所有权
	if apiKey.UserID != userID.(string) {
		Forbidden(c, MsgPermissionDenied)
		return
	}

	Success(c, apiKeyResponse{
		ID:         apiKey.ID,
		Key:        maskAPIKey(apiKey.Key), // 脱敏显示
		Name:       apiKey.Name,
		IsActive:   apiKey.IsActive,
		CreatedAt:  apiKey.CreatedAt,
		ExpiresAt:  apiKey.ExpiresAt,
		LastUsedAt: apiKey.LastUsedAt,
	})
}

// DeleteAPIKey godoc
// @Summary 删除API Key
// @Description 删除指定的API Key
// @Tags APIKeys
// @Security BearerAuth
// @Param id path string true "API Key ID"
// @Success 204
// @Failure 401 {object} Response
// @Failure 404 {object} Response
// @Router /v1/api-keys/{id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	keyID := c.Param("id")

	// 删除API Key
	err := h.apiKeyService.DeleteAPIKey(userID.(string), keyID)
	if err != nil {
		if err == service.ErrAPIKeyNotFound {
			NotFound(c, MsgAPIKeyNotFound)
			return
		}
		Forbidden(c, err.Error())
		return
	}

	NoContent(c)
}

// maskAPIKey 脱敏显示API Key
// 只显示前8个字符和后4个字符，中间用*代替
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	return key[:8] + "****" + key[len(key)-4:]
}

package httptransport

import (
	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/service"
)

// ConfigHandler 系统配置API处理器
type ConfigHandler struct {
	configService *service.ConfigService
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(configService *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// GetSystemConfig godoc
// @Summary 获取系统配置
// @Description 获取当前系统配置（需要管理员权限）
// @Tags Admin - Config
// @Produce json
// @Success 200 {object} Response{data=domain.SystemConfig}
// @Failure 500 {object} Response
// @Router /v1/admin/config [get]
func (h *ConfigHandler) GetSystemConfig(c *gin.Context) {
	config, err := h.configService.GetSystemConfig()
	if err != nil {
		InternalError(c, "获取系统配置失败")
		return
	}

	Success(c, config)
}

// UpdateSystemConfigRequest 更新系统配置请求
type UpdateSystemConfigRequest struct {
	SMTP      *domain.SMTPConfig      `json:"smtp,omitempty"`
	Mailbox   *domain.MailboxConfig   `json:"mailbox,omitempty"`
	RateLimit *domain.RateLimitConfig `json:"rateLimit,omitempty"`
	Security  *domain.SecurityConfig  `json:"security,omitempty"`
}

// UpdateSystemConfig godoc
// @Summary 更新系统配置
// @Description 更新系统配置（需要超级管理员权限）
// @Tags Admin - Config
// @Accept json
// @Produce json
// @Param request body UpdateSystemConfigRequest true "配置信息"
// @Success 200 {object} Response{data=domain.SystemConfig}
// @Failure 400 {object} Response
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /v1/admin/config [put]
func (h *ConfigHandler) UpdateSystemConfig(c *gin.Context) {
	userID := c.GetString("userID")

	var req UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	input := service.UpdateSystemConfigInput{
		SMTP:      req.SMTP,
		Mailbox:   req.Mailbox,
		RateLimit: req.RateLimit,
		Security:  req.Security,
		UpdatedBy: userID,
	}

	config, err := h.configService.UpdateSystemConfig(input)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMsg(c, "系统配置更新成功", config)
}

// ResetSystemConfig godoc
// @Summary 重置系统配置
// @Description 将系统配置重置为默认值（需要超级管理员权限）
// @Tags Admin - Config
// @Produce json
// @Success 200 {object} Response{data=domain.SystemConfig}
// @Failure 403 {object} Response
// @Failure 500 {object} Response
// @Router /v1/admin/config/reset [post]
func (h *ConfigHandler) ResetSystemConfig(c *gin.Context) {
	userID := c.GetString("userID")

	config, err := h.configService.ResetSystemConfig(userID)
	if err != nil {
		InternalError(c, "重置系统配置失败")
		return
	}

	SuccessWithMsg(c, "系统配置已重置为默认值", config)
}

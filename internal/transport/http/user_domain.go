package httptransport

import (
	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/service"
)

// UserDomainHandler 用户域名处理器
type UserDomainHandler struct {
	service *service.UserDomainService
}

// NewUserDomainHandler 创建用户域名处理器
func NewUserDomainHandler(service *service.UserDomainService) *UserDomainHandler {
	return &UserDomainHandler{
		service: service,
	}
}

// AddUserDomainRequest 添加用户域名请求
type AddUserDomainRequest struct {
	Domain string `json:"domain" binding:"required"`
	Mode   string `json:"mode" binding:"required,oneof=shared exclusive catch_all whitelist"`
}

// AddDomain godoc
// @Summary 添加用户自定义域名
// @Description 用户添加自己的域名，可选择共享或独享模式
// @Tags User Domains
// @Accept json
// @Produce json
// @Param request body AddUserDomainRequest true "域名信息"
// @Success 201 {object} domain.UserDomain
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 409 {object} Response
// @Router /v1/user/domains [post]
func (h *UserDomainHandler) AddDomain(c *gin.Context) {
	// 获取当前用户ID
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	var req AddUserDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	// 转换模式
	mode := domain.DomainMode(req.Mode)

	input := service.AddDomainInput{
		UserID: userID,
		Domain: req.Domain,
		Mode:   mode,
	}

	userDomain, err := h.service.AddDomain(input)
	if err != nil {
		switch err {
		case service.ErrInvalidDomain:
			BadRequest(c, GetErrorMessage(service.ErrInvalidDomain))
		case service.ErrDomainAlreadyExists:
			Conflict(c, GetErrorMessage(service.ErrDomainAlreadyExists))
		default:
			InternalError(c, MsgDomainAddFailed)
		}
		return
	}

	Created(c, userDomain)
}

// ListDomains godoc
// @Summary 获取用户的域名列表
// @Description 获取当前用户添加的所有域名
// @Tags User Domains
// @Produce json
// @Success 200 {array} domain.UserDomain
// @Failure 401 {object} Response
// @Router /v1/user/domains [get]
func (h *UserDomainHandler) ListDomains(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	domains, err := h.service.ListUserDomains(userID)
	if err != nil {
		InternalError(c, MsgDomainListFailed)
		return
	}

	// 如果没有域名，返回空数组而不是 null
	if domains == nil {
		domains = []*domain.UserDomain{}
	}

	Success(c, domains)
}

// GetDomain godoc
// @Summary 获取域名详情
// @Description 获取指定域名的详细信息
// @Tags User Domains
// @Produce json
// @Param id path string true "域名ID"
// @Success 200 {object} domain.UserDomain
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Router /v1/user/domains/{id} [get]
func (h *UserDomainHandler) GetDomain(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	domainID := c.Param("id")

	userDomain, err := h.service.GetUserDomain(domainID, userID)
	if err != nil {
		switch err {
		case service.ErrDomainNotFound:
			NotFound(c, GetErrorMessage(service.ErrDomainNotFound))
		case service.ErrNotDomainOwner:
			Forbidden(c, "无权操作此域名")
		default:
			InternalError(c, MsgDomainGetFailed)
		}
		return
	}

	Success(c, userDomain)
}

// VerifyDomain godoc
// @Summary 验证域名所有权
// @Description 通过 DNS TXT 记录验证域名所有权
// @Tags User Domains
// @Produce json
// @Param id path string true "域名ID"
// @Success 200 {object} domain.UserDomain
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 422 {object} Response
// @Router /v1/user/domains/{id}/verify [post]
func (h *UserDomainHandler) VerifyDomain(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	domainID := c.Param("id")

	userDomain, err := h.service.VerifyDomain(domainID, userID)
	if err != nil {
		switch err {
		case service.ErrDomainNotFound:
			NotFound(c, GetErrorMessage(service.ErrDomainNotFound))
		case service.ErrNotDomainOwner:
			Forbidden(c, "无权操作此域名")
		case service.ErrDomainVerifyFailed:
			UnprocessableEntity(c, GetErrorMessage(service.ErrDomainVerifyFailed))
		default:
			InternalError(c, MsgDomainVerifyFailed)
		}
		return
	}

	Success(c, userDomain)
}

// GetSetupInstructions godoc
// @Summary 获取域名配置说明
// @Description 获取域名的 DNS 配置说明（MX 记录、TXT 记录）
// @Tags User Domains
// @Produce json
// @Param id path string true "域名ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Router /v1/user/domains/{id}/instructions [get]
func (h *UserDomainHandler) GetSetupInstructions(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	domainID := c.Param("id")

	instructions, err := h.service.GetDomainSetupInstructions(domainID, userID)
	if err != nil {
		switch err {
		case service.ErrDomainNotFound:
			NotFound(c, GetErrorMessage(service.ErrDomainNotFound))
		case service.ErrNotDomainOwner:
			Forbidden(c, "无权操作此域名")
		default:
			InternalError(c, MsgDomainInstructionsFailed)
		}
		return
	}

	Success(c, instructions)
}

// UpdateDomainModeRequest 更新域名模式请求
type UpdateDomainModeRequest struct {
	Mode string `json:"mode" binding:"required,oneof=shared exclusive catch_all whitelist"`
}

// UpdateDomainMode godoc
// @Summary 更新域名模式
// @Description 更新域名的共享/独享模式
// @Tags User Domains
// @Accept json
// @Produce json
// @Param id path string true "域名ID"
// @Param request body UpdateDomainModeRequest true "模式信息"
// @Success 200 {object} domain.UserDomain
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Router /v1/user/domains/{id} [patch]
func (h *UserDomainHandler) UpdateDomainMode(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	domainID := c.Param("id")

	var req UpdateDomainModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	mode := domain.DomainMode(req.Mode)

	userDomain, err := h.service.UpdateDomainMode(domainID, userID, mode)
	if err != nil {
		switch err {
		case service.ErrDomainNotFound:
			NotFound(c, GetErrorMessage(service.ErrDomainNotFound))
		case service.ErrNotDomainOwner:
			Forbidden(c, "无权操作此域名")
		default:
			InternalError(c, MsgDomainUpdateFailed)
		}
		return
	}

	Success(c, userDomain)
}

// DeleteDomain godoc
// @Summary 删除域名
// @Description 删除用户自定义域名
// @Tags User Domains
// @Param id path string true "域名ID"
// @Success 204
// @Failure 401 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Router /v1/user/domains/{id} [delete]
func (h *UserDomainHandler) DeleteDomain(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		Unauthorized(c, MsgAuthRequired)
		return
	}

	domainID := c.Param("id")

	err := h.service.DeleteUserDomain(domainID, userID)
	if err != nil {
		switch err {
		case service.ErrDomainNotFound:
			NotFound(c, GetErrorMessage(service.ErrDomainNotFound))
		case service.ErrNotDomainOwner:
			Forbidden(c, "无权操作此域名")
		default:
			// 如果是 "cannot delete domain with active mailboxes" 错误
			if err.Error() == "cannot delete domain with active mailboxes" {
				Conflict(c, MsgDomainHasMailboxes)
			} else {
				InternalError(c, MsgDomainDeleteFailed)
			}
		}
		return
	}

	NoContent(c)
}

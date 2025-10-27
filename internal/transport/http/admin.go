package httptransport

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/service"
)

// AdminHandler 管理API处理器
type AdminHandler struct {
	adminService        *service.AdminService
	systemDomainService *service.SystemDomainService
}

// NewAdminHandler 创建管理处理器
func NewAdminHandler(adminService *service.AdminService, systemDomainService *service.SystemDomainService) *AdminHandler {
	return &AdminHandler{
		adminService:        adminService,
		systemDomainService: systemDomainService,
	}
}

// ========== 用户管理 ==========

// ListUsers godoc
// @Summary 获取用户列表
// @Description 获取系统中的用户列表（需要管理员权限）
// @Tags Admin
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(20)
// @Param search query string false "搜索关键词（邮箱/用户名）"
// @Param role query string false "角色过滤（user/admin/super）"
// @Param tier query string false "等级过滤（free/basic/pro/enterprise）"
// @Param isActive query bool false "激活状态过滤"
// @Success 200 {object} service.ListUsersOutput
// @Failure 403 {object} Response
// @Router /v1/admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	search := c.Query("search")

	// 解析可选过滤参数
	var role *domain.UserRole
	if r := c.Query("role"); r != "" {
		roleVal := domain.UserRole(r)
		role = &roleVal
	}

	var tier *domain.UserTier
	if t := c.Query("tier"); t != "" {
		tierVal := domain.UserTier(t)
		tier = &tierVal
	}

	var isActive *bool
	if a := c.Query("isActive"); a != "" {
		active := a == "true"
		isActive = &active
	}

	input := service.ListUsersInput{
		Page:     page,
		PageSize: pageSize,
		Search:   search,
		Role:     role,
		Tier:     tier,
		IsActive: isActive,
	}

	result, err := h.adminService.ListUsers(input)
	if err != nil {
		InternalError(c, MsgUserListFailed)
		return
	}

	Success(c, result)
}

// GetUser godoc
// @Summary 获取用户详情
// @Description 获取指定用户的详细信息（需要管理员权限）
// @Tags Admin
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} domain.User
// @Failure 404 {object} Response
// @Router /v1/admin/users/{id} [get]
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.adminService.GetUser(userID)
	if err != nil {
		if err == service.ErrAdminUserNotFound {
			NotFound(c, MsgUserNotFound)
		} else {
			InternalError(c, MsgUserGetFailed)
		}
		return
	}

	Success(c, user)
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Role            *domain.UserRole `json:"role,omitempty"`
	Tier            *domain.UserTier `json:"tier,omitempty"`
	IsActive        *bool            `json:"isActive,omitempty"`
	IsEmailVerified *bool            `json:"isEmailVerified,omitempty"`
}

// UpdateUser godoc
// @Summary 更新用户信息
// @Description 更新用户的角色、等级或状态（需要管理员权限）
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body UpdateUserRequest true "更新内容"
// @Success 200 {object} domain.User
// @Failure 400 {object} Response
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Router /v1/admin/users/{id} [patch]
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	operatorID := c.GetString("userID") // 从JWT中间件获取

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	input := service.UpdateUserInput{
		UserID:          userID,
		Role:            req.Role,
		Tier:            req.Tier,
		IsActive:        req.IsActive,
		IsEmailVerified: req.IsEmailVerified,
		OperatorID:      operatorID,
	}

	user, err := h.adminService.UpdateUser(input)
	if err != nil {
		switch err {
		case service.ErrAdminUserNotFound:
			NotFound(c, MsgUserNotFound)
		case service.ErrCannotModifySelf:
			Forbidden(c, "不能修改自己的账户")
		case service.ErrCannotModifySuper:
			Forbidden(c, "不能修改超级管理员账户")
		case service.ErrInsufficientPermission:
			Forbidden(c, MsgPermissionDenied)
		default:
			InternalError(c, MsgUserUpdateFailed)
		}
		return
	}

	Success(c, user)
}

// DeleteUser godoc
// @Summary 删除用户
// @Description 删除用户及其所有数据（需要超级管理员权限）
// @Tags Admin
// @Param id path string true "用户ID"
// @Success 204
// @Failure 403 {object} Response
// @Failure 404 {object} Response
// @Router /v1/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	operatorID := c.GetString("userID")

	err := h.adminService.DeleteUser(userID, operatorID)
	if err != nil {
		switch err {
		case service.ErrAdminUserNotFound:
			NotFound(c, MsgUserNotFound)
		case service.ErrCannotModifySelf:
			Forbidden(c, "不能删除自己的账户")
		case service.ErrCannotModifySuper:
			Forbidden(c, "不能删除超级管理员账户")
		default:
			InternalError(c, MsgUserDeleteFailed)
		}
		return
	}

	NoContent(c)
}

// ========== 系统域名管理 ==========

// ListSystemDomains godoc
// @Summary 获取系统域名列表
// @Description 获取所有系统域名（需要管理员权限）
// @Tags Admin - System Domains
// @Produce json
// @Success 200 {array} domain.SystemDomain
// @Failure 500 {object} Response
// @Router /v1/admin/domains [get]
func (h *AdminHandler) ListSystemDomains(c *gin.Context) {
	domains, err := h.systemDomainService.ListSystemDomains()
	if err != nil {
		InternalError(c, MsgDomainListFailedAdmin)
		return
	}

	// 如果没有域名，返回空数组
	if domains == nil {
		domains = []*domain.SystemDomain{}
	}

	Success(c, domains)
}

// AddSystemDomainRequest 添加系统域名请求
type AddSystemDomainRequest struct {
	Domain string `json:"domain" binding:"required"`
	Notes  string `json:"notes"`
}

// AddSystemDomain godoc
// @Summary 添加系统域名
// @Description 添加新的系统域名，生成 DNS 验证令牌（需要超级管理员权限）
// @Tags Admin - System Domains
// @Accept json
// @Produce json
// @Param request body AddSystemDomainRequest true "域名信息"
// @Success 201 {object} domain.SystemDomain
// @Failure 400 {object} Response
// @Failure 409 {object} Response
// @Router /v1/admin/domains [post]
func (h *AdminHandler) AddSystemDomain(c *gin.Context) {
	userID := c.GetString("userID") // 获取当前管理员用户ID

	var req AddSystemDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	input := service.AddSystemDomainInput{
		Domain:    req.Domain,
		CreatedBy: userID,
		Notes:     req.Notes,
	}

	sysDomain, err := h.systemDomainService.AddSystemDomain(input)
	if err != nil {
		switch err {
		case service.ErrInvalidSystemDomain:
			BadRequest(c, "无效的域名格式")
		case service.ErrSystemDomainAlreadyExists:
			Conflict(c, MsgDomainAlreadyExists)
		default:
			InternalError(c, MsgDomainAddFailedAdmin)
		}
		return
	}

	Created(c, sysDomain)
}

// GetSystemDomain godoc
// @Summary 获取系统域名详情
// @Description 获取指定系统域名的详细信息（需要管理员权限）
// @Tags Admin - System Domains
// @Produce json
// @Param id path string true "域名ID"
// @Success 200 {object} domain.SystemDomain
// @Failure 404 {object} Response
// @Router /v1/admin/domains/{id} [get]
func (h *AdminHandler) GetSystemDomain(c *gin.Context) {
	domainID := c.Param("id")

	sysDomain, err := h.systemDomainService.GetSystemDomain(domainID)
	if err != nil {
		if err == service.ErrSystemDomainNotFound {
			NotFound(c, MsgDomainNotFoundAdmin)
		} else {
			InternalError(c, "获取域名详情失败")
		}
		return
	}

	Success(c, sysDomain)
}

// VerifySystemDomain godoc
// @Summary 验证系统域名
// @Description 通过 DNS TXT 记录验证域名所有权（需要管理员权限）
// @Tags Admin - System Domains
// @Produce json
// @Param id path string true "域名ID"
// @Success 200 {object} domain.SystemDomain
// @Failure 404 {object} Response
// @Failure 422 {object} Response
// @Router /v1/admin/domains/{id}/verify [post]
func (h *AdminHandler) VerifySystemDomain(c *gin.Context) {
	domainID := c.Param("id")

	sysDomain, err := h.systemDomainService.VerifySystemDomain(domainID)
	if err != nil {
		switch err {
		case service.ErrSystemDomainNotFound:
			NotFound(c, MsgDomainNotFoundAdmin)
		case service.ErrSystemDomainVerifyFailed:
			UnprocessableEntity(c, "DNS 验证失败，请检查 TXT 记录是否正确配置")
		default:
			InternalError(c, "验证域名失败")
		}
		return
	}

	Success(c, sysDomain)
}

// GetSystemDomainInstructions godoc
// @Summary 获取系统域名配置说明
// @Description 获取域名的 DNS 配置说明（MX 记录、TXT 记录）
// @Tags Admin - System Domains
// @Produce json
// @Param id path string true "域名ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} Response
// @Router /v1/admin/domains/{id}/instructions [get]
func (h *AdminHandler) GetSystemDomainInstructions(c *gin.Context) {
	domainID := c.Param("id")

	instructions, err := h.systemDomainService.GetSetupInstructions(domainID)
	if err != nil {
		if err == service.ErrSystemDomainNotFound {
			NotFound(c, MsgDomainNotFoundAdmin)
		} else {
			InternalError(c, "获取配置说明失败")
		}
		return
	}

	Success(c, instructions)
}

// ToggleSystemDomainStatusRequest 切换域名状态请求
type ToggleSystemDomainStatusRequest struct {
	IsActive bool `json:"isActive"`
}

// ToggleSystemDomainStatus godoc
// @Summary 切换系统域名状态
// @Description 启用或禁用系统域名（需要管理员权限）
// @Tags Admin - System Domains
// @Accept json
// @Produce json
// @Param id path string true "域名ID"
// @Param request body ToggleSystemDomainStatusRequest true "状态信息"
// @Success 200 {object} domain.SystemDomain
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /v1/admin/domains/{id}/toggle [patch]
func (h *AdminHandler) ToggleSystemDomainStatus(c *gin.Context) {
	domainID := c.Param("id")

	var req ToggleSystemDomainStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	sysDomain, err := h.systemDomainService.ToggleDomainStatus(domainID, req.IsActive)
	if err != nil {
		switch err {
		case service.ErrSystemDomainNotFound:
			NotFound(c, MsgDomainNotFoundAdmin)
		case service.ErrSystemDomainNotVerified:
			BadRequest(c, "只有已验证的域名才能启用")
		default:
			InternalError(c, "切换域名状态失败")
		}
		return
	}

	Success(c, sysDomain)
}

// SetDefaultSystemDomain godoc
// @Summary 设置默认系统域名
// @Description 将指定域名设为默认域名（需要超级管理员权限）
// @Tags Admin - System Domains
// @Param id path string true "域名ID"
// @Success 200
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /v1/admin/domains/{id}/set-default [post]
func (h *AdminHandler) SetDefaultSystemDomain(c *gin.Context) {
	domainID := c.Param("id")

	err := h.systemDomainService.SetDefaultDomain(domainID)
	if err != nil {
		switch err {
		case service.ErrSystemDomainNotFound:
			NotFound(c, MsgDomainNotFoundAdmin)
		case service.ErrSystemDomainNotVerified:
			BadRequest(c, "只有已验证且激活的域名才能设为默认")
		default:
			InternalError(c, "设置默认域名失败")
		}
		return
	}

	SuccessWithMsg(c, "默认域名设置成功", nil)
}

// RecoverSystemDomainRequest 找回系统域名请求
type RecoverSystemDomainRequest struct {
	Domain string `json:"domain" binding:"required"`
}

// RecoverSystemDomain godoc
// @Summary 找回系统域名
// @Description 通过 DNS 验证找回已删除或验证失败的域名（需要超级管理员权限）
// @Tags Admin - System Domains
// @Accept json
// @Produce json
// @Param request body RecoverSystemDomainRequest true "域名信息"
// @Success 200 {object} domain.SystemDomain
// @Failure 400 {object} Response
// @Failure 422 {object} Response
// @Router /v1/admin/domains/recover [post]
func (h *AdminHandler) RecoverSystemDomain(c *gin.Context) {
	userID := c.GetString("userID")

	var req RecoverSystemDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	sysDomain, err := h.systemDomainService.RecoverSystemDomain(req.Domain, userID)
	if err != nil {
		switch err {
		case service.ErrInvalidSystemDomain:
			BadRequest(c, "无效的域名格式")
		default:
			UnprocessableEntity(c, err.Error())
		}
		return
	}

	Success(c, sysDomain)
}

// DeleteSystemDomain godoc
// @Summary 删除系统域名
// @Description 删除系统域名（需要超级管理员权限）
// @Tags Admin - System Domains
// @Param id path string true "域名ID"
// @Success 204
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 409 {object} Response
// @Router /v1/admin/domains/{id} [delete]
func (h *AdminHandler) DeleteSystemDomain(c *gin.Context) {
	domainID := c.Param("id")

	err := h.systemDomainService.DeleteSystemDomain(domainID)
	if err != nil {
		switch err {
		case service.ErrSystemDomainNotFound:
			NotFound(c, MsgDomainNotFoundAdmin)
		case service.ErrCannotDeleteDefaultDomain:
			BadRequest(c, "不能删除默认域名")
		case service.ErrSystemDomainHasMailboxes:
			Conflict(c, MsgDomainHasMailboxes)
		default:
			InternalError(c, MsgDomainRemoveFailed)
		}
		return
	}

	NoContent(c)
}

// ========== 统计信息 ==========

// GetStatistics godoc
// @Summary 获取系统统计
// @Description 获取系统使用统计信息（需要管理员权限）
// @Tags Admin
// @Produce json
// @Success 200 {object} domain.SystemStatistics
// @Router /v1/admin/statistics [get]
func (h *AdminHandler) GetStatistics(c *gin.Context) {
	stats, err := h.adminService.GetStatistics()
	if err != nil {
		InternalError(c, MsgStatisticsGetFailed)
		return
	}

	Success(c, stats)
}

// GetUserQuota godoc
// @Summary 获取用户配额
// @Description 获取用户的配额信息（需要管理员权限）
// @Tags Admin
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} domain.Quota
// @Failure 404 {object} Response
// @Router /v1/admin/users/{id}/quota [get]
func (h *AdminHandler) GetUserQuota(c *gin.Context) {
	userID := c.Param("id")

	quota, err := h.adminService.GetUserQuota(userID)
	if err != nil {
		if err == service.ErrAdminUserNotFound {
			NotFound(c, MsgUserNotFound)
		} else {
			InternalError(c, MsgQuotaGetFailed)
		}
		return
	}

	Success(c, quota)
}

// UpdateUserQuotaRequest 更新用户配额请求
type UpdateUserQuotaRequest struct {
	MaxMailboxes            int `json:"maxMailboxes"`
	MaxMessagesPerMailbox   int `json:"maxMessagesPerMailbox"`
	MaxAPIRequestsPerMinute int `json:"maxApiRequestsPerMinute"`
	MaxConcurrentRequests   int `json:"maxConcurrentRequests"`
}

// UpdateUserQuota godoc
// @Summary 更新用户配额
// @Description 自定义用户配额限制（需要管理员权限）
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body UpdateUserQuotaRequest true "配额信息"
// @Success 200
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /v1/admin/users/{id}/quota [put]
func (h *AdminHandler) UpdateUserQuota(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateUserQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, MsgInvalidRequest)
		return
	}

	quota := domain.Quota{
		UserID:                  userID,
		MaxMailboxes:            req.MaxMailboxes,
		MaxMessagesPerMailbox:   req.MaxMessagesPerMailbox,
		MaxAPIRequestsPerMinute: req.MaxAPIRequestsPerMinute,
		MaxConcurrentRequests:   req.MaxConcurrentRequests,
	}

	err := h.adminService.UpdateUserQuota(userID, quota)
	if err != nil {
		if err == service.ErrAdminUserNotFound {
			NotFound(c, MsgUserNotFound)
		} else {
			InternalError(c, MsgQuotaUpdateFailed)
		}
		return
	}

	SuccessWithMsg(c, "配额更新成功", nil)
}

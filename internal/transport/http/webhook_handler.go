package httptransport

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/service"
)

// ========== Webhook Handlers ==========

// createWebhook godoc
// @Summary 创建 Webhook
// @Description 创建一个新的 Webhook 配置
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhook body service.CreateWebhookInput true "Webhook 信息"
// @Success 200 {object} Response{data=domain.Webhook}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /v1/webhooks [post]
func (h *Handler) createWebhook(c *gin.Context) {
	var input service.CreateWebhookInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "无效的请求参数")
		return
	}

	// 从 JWT 中获取用户 ID
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "未授权")
		return
	}
	input.UserID = userID.(string)

	webhook, err := h.webhook.CreateWebhook(input)
	if err != nil {
		InternalError(c, "创建 Webhook 失败")
		return
	}

	Success(c, webhook)
}

// listWebhooks godoc
// @Summary 列出 Webhooks
// @Description 列出当前用户的所有 Webhooks
// @Tags Webhooks
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=[]domain.Webhook}
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /v1/webhooks [get]
func (h *Handler) listWebhooks(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "未授权")
		return
	}

	webhooks, err := h.webhook.ListWebhooks(userID.(string))
	if err != nil {
		InternalError(c, "获取 Webhook 列表失败")
		return
	}

	Success(c, webhooks)
}

// getWebhook godoc
// @Summary 获取 Webhook
// @Description 获取指定 Webhook 的详细信息
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID"
// @Success 200 {object} Response{data=domain.Webhook}
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /v1/webhooks/{id} [get]
func (h *Handler) getWebhook(c *gin.Context) {
	id := c.Param("id")

	webhook, err := h.webhook.GetWebhook(id)
	if err != nil {
		NotFound(c, "Webhook 不存在")
		return
	}

	// 验证权限
	userID, _ := c.Get("userID")
	if webhook.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	Success(c, webhook)
}

// updateWebhook godoc
// @Summary 更新 Webhook
// @Description 更新 Webhook 配置
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID"
// @Param webhook body service.UpdateWebhookInput true "更新信息"
// @Success 200 {object} Response{data=domain.Webhook}
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /v1/webhooks/{id} [patch]
func (h *Handler) updateWebhook(c *gin.Context) {
	id := c.Param("id")

	// 验证权限
	webhook, err := h.webhook.GetWebhook(id)
	if err != nil {
		NotFound(c, "Webhook 不存在")
		return
	}

	userID, _ := c.Get("userID")
	if webhook.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	var input service.UpdateWebhookInput
	if err := c.ShouldBindJSON(&input); err != nil {
		BadRequest(c, "无效的请求参数")
		return
	}

	updated, err := h.webhook.UpdateWebhook(id, input)
	if err != nil {
		InternalError(c, "更新 Webhook 失败")
		return
	}

	Success(c, updated)
}

// deleteWebhook godoc
// @Summary 删除 Webhook
// @Description 删除指定的 Webhook
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID"
// @Success 200 {object} Response
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /v1/webhooks/{id} [delete]
func (h *Handler) deleteWebhook(c *gin.Context) {
	id := c.Param("id")

	// 验证权限
	webhook, err := h.webhook.GetWebhook(id)
	if err != nil {
		NotFound(c, "Webhook 不存在")
		return
	}

	userID, _ := c.Get("userID")
	if webhook.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	if err := h.webhook.DeleteWebhook(id); err != nil {
		InternalError(c, "删除 Webhook 失败")
		return
	}

	SuccessWithMsg(c, "Webhook 已删除", nil)
}

// getWebhookDeliveries godoc
// @Summary 获取投递记录
// @Description 获取 Webhook 的投递记录
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param id path string true "Webhook ID"
// @Param limit query int false "记录数量（默认20，最大100）"
// @Success 200 {object} Response{data=[]domain.WebhookDelivery}
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Security BearerAuth
// @Router /v1/webhooks/{id}/deliveries [get]
func (h *Handler) getWebhookDeliveries(c *gin.Context) {
	id := c.Param("id")

	// 验证权限
	webhook, err := h.webhook.GetWebhook(id)
	if err != nil {
		NotFound(c, "Webhook 不存在")
		return
	}

	userID, _ := c.Get("userID")
	if webhook.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	// 获取 limit 参数
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	deliveries, err := h.webhook.GetDeliveries(id, limit)
	if err != nil {
		InternalError(c, "获取投递记录失败")
		return
	}

	Success(c, deliveries)
}

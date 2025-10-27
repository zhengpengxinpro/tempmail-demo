package httptransport

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tempmail/backend/internal/service"
)

// ========== Tag Handlers ==========

// createTag godoc
// @Summary 创建标签
// @Description 创建一个新的邮件标签
// @Tags Tags
// @Accept json
// @Produce json
// @Param tag body service.CreateTagInput true "标签信息"
// @Success 200 {object} Response{data=domain.Tag}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/tags [post]
func (h *Handler) createTag(c *gin.Context) {
	var input service.CreateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "无效的请求参数",
		})
		return
	}

	// 从 JWT 中获取用户 ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorResponse{
			Error: "未授权",
		})
		return
	}
	input.UserID = userID.(string)

	tag, err := h.tag.CreateTag(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: err.Error(),
		})
		return
	}

	Success(c, tag)
}

// listTags godoc
// @Summary 列出标签
// @Description 列出当前用户的所有标签
// @Tags Tags
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=[]domain.TagWithCount}
// @Failure 401 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/tags [get]
func (h *Handler) listTags(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorResponse{
			Error: "未授权",
		})
		return
	}

	tags, err := h.tag.ListTags(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "获取标签列表失败",
		})
		return
	}

	Success(c, tags)
}

// getTag godoc
// @Summary 获取标签
// @Description 获取指定标签的详细信息
// @Tags Tags
// @Accept json
// @Produce json
// @Param id path string true "标签ID"
// @Success 200 {object} Response{data=domain.Tag}
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/tags/{id} [get]
func (h *Handler) getTag(c *gin.Context) {
	id := c.Param("id")

	tag, err := h.tag.GetTag(id)
	if err != nil {
		NotFound(c, "标签不存在")
		return
	}

	// 验证权限
	userID, _ := c.Get("userID")
	if tag.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	Success(c, tag)
}

// updateTag godoc
// @Summary 更新标签
// @Description 更新标签信息
// @Tags Tags
// @Accept json
// @Produce json
// @Param id path string true "标签ID"
// @Param tag body service.UpdateTagInput true "更新信息"
// @Success 200 {object} Response{data=domain.Tag}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/tags/{id} [patch]
func (h *Handler) updateTag(c *gin.Context) {
	id := c.Param("id")

	// 验证权限
	tag, err := h.tag.GetTag(id)
	if err != nil {
		NotFound(c, "标签不存在")
		return
	}

	userID, _ := c.Get("userID")
	if tag.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	var input service.UpdateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "无效的请求参数",
		})
		return
	}

	updated, err := h.tag.UpdateTag(id, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: err.Error(),
		})
		return
	}

	Success(c, updated)
}

// deleteTag godoc
// @Summary 删除标签
// @Description 删除指定的标签
// @Tags Tags
// @Accept json
// @Produce json
// @Param id path string true "标签ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/tags/{id} [delete]
func (h *Handler) deleteTag(c *gin.Context) {
	id := c.Param("id")

	// 验证权限
	tag, err := h.tag.GetTag(id)
	if err != nil {
		NotFound(c, "标签不存在")
		return
	}

	userID, _ := c.Get("userID")
	if tag.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	if err := h.tag.DeleteTag(id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "删除标签失败",
		})
		return
	}

	SuccessWithMsg(c, "标签已删除", nil)
}

// addMessageTag godoc
// @Summary 为邮件添加标签
// @Description 为指定邮件添加标签
// @Tags Tags
// @Accept json
// @Produce json
// @Param mailboxId path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Param data body object{tagId:string} true "标签ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Security BearerAuth
// @Router /v1/mailboxes/{mailboxId}/messages/{messageId}/tags [post]
func (h *Handler) addMessageTag(c *gin.Context) {
	messageID := c.Param("messageId")

	var input struct {
		TagID string `json:"tagId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error: "无效的请求参数",
		})
		return
	}

	// 验证标签权限
	tag, err := h.tag.GetTag(input.TagID)
	if err != nil {
		NotFound(c, "标签不存在")
		return
	}

	userID, _ := c.Get("userID")
	if tag.UserID != userID.(string) {
		Forbidden(c, "无权访问该标签")
		return
	}

	if err := h.tag.AddMessageTag(messageID, input.TagID); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "添加标签失败",
		})
		return
	}

	SuccessWithMsg(c, "标签已添加", nil)
}

// removeMessageTag godoc
// @Summary 移除邮件标签
// @Description 从邮件中移除指定标签
// @Tags Tags
// @Accept json
// @Produce json
// @Param mailboxId path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Param tagId path string true "标签ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/mailboxes/{mailboxId}/messages/{messageId}/tags/{tagId} [delete]
func (h *Handler) removeMessageTag(c *gin.Context) {
	messageID := c.Param("messageId")
	tagID := c.Param("tagId")

	// 验证标签权限
	tag, err := h.tag.GetTag(tagID)
	if err != nil {
		NotFound(c, "标签不存在")
		return
	}

	userID, _ := c.Get("userID")
	if tag.UserID != userID.(string) {
		Forbidden(c, "无权访问该标签")
		return
	}

	if err := h.tag.RemoveMessageTag(messageID, tagID); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "移除标签失败",
		})
		return
	}

	SuccessWithMsg(c, "标签已移除", nil)
}

// getMessageTags godoc
// @Summary 获取邮件标签
// @Description 获取指定邮件的所有标签
// @Tags Tags
// @Accept json
// @Produce json
// @Param mailboxId path string true "邮箱ID"
// @Param messageId path string true "邮件ID"
// @Success 200 {object} Response{data=[]domain.Tag}
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/mailboxes/{mailboxId}/messages/{messageId}/tags [get]
func (h *Handler) getMessageTags(c *gin.Context) {
	messageID := c.Param("messageId")

	tags, err := h.tag.GetMessageTags(messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "获取标签失败",
		})
		return
	}

	Success(c, tags)
}

// listMessagesByTag godoc
// @Summary 按标签列出邮件
// @Description 列出指定标签下的所有邮件
// @Tags Tags
// @Accept json
// @Produce json
// @Param id path string true "标签ID"
// @Success 200 {object} Response{data=[]domain.Message}
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Security BearerAuth
// @Router /v1/tags/{id}/messages [get]
func (h *Handler) listMessagesByTag(c *gin.Context) {
	tagID := c.Param("id")

	// 验证标签权限
	tag, err := h.tag.GetTag(tagID)
	if err != nil {
		NotFound(c, "标签不存在")
		return
	}

	userID, _ := c.Get("userID")
	if tag.UserID != userID.(string) {
		Forbidden(c, "无权访问")
		return
	}

	messages, err := h.tag.ListMessagesByTag(tagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "获取邮件列表失败",
		})
		return
	}

	Success(c, messages)
}

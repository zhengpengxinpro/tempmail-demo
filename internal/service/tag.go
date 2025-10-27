package service

import (
	"fmt"

	"github.com/google/uuid"

	"tempmail/backend/internal/domain"
)

// TagService 标签服务
type TagService struct {
	store domain.Store
}

// NewTagService 创建标签服务
func NewTagService(store domain.Store) *TagService {
	return &TagService{
		store: store,
	}
}

// CreateTagInput 创建标签输入
type CreateTagInput struct {
	UserID      string `json:"-"` // 从JWT中获取，不需要客户端提供
	Name        string `json:"name" binding:"required,min=1,max=20"`
	Color       string `json:"color" binding:"required"`
	Description string `json:"description" binding:"omitempty,max=100"`
}

// UpdateTagInput 更新标签输入
type UpdateTagInput struct {
	Name        string `json:"name" binding:"omitempty,min=1,max=20"`
	Color       string `json:"color" binding:"omitempty"`
	Description string `json:"description" binding:"omitempty,max=100"`
}

// CreateTag 创建标签
//
// 参数:
//   - input: 创建标签输入
//
// 返回值:
//   - *domain.Tag: 创建的标签
//   - error: 错误信息
func (s *TagService) CreateTag(input CreateTagInput) (*domain.Tag, error) {
	// 检查是否已存在同名标签
	existing, _ := s.store.GetTagByName(input.UserID, input.Name)
	if existing != nil {
		return nil, fmt.Errorf("tag already exists")
	}

	tag := &domain.Tag{
		ID:          uuid.New().String(),
		UserID:      input.UserID,
		Name:        input.Name,
		Color:       input.Color,
		Description: input.Description,
	}

	if err := s.store.CreateTag(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// GetTag 获取标签
//
// 参数:
//   - id: 标签ID
//
// 返回值:
//   - *domain.Tag: 标签详情
//   - error: 错误信息
func (s *TagService) GetTag(id string) (*domain.Tag, error) {
	return s.store.GetTag(id)
}

// ListTags 列出用户的所有标签
//
// 参数:
//   - userID: 用户ID
//
// 返回值:
//   - []domain.TagWithCount: 标签列表（含计数）
//   - error: 错误信息
func (s *TagService) ListTags(userID string) ([]domain.TagWithCount, error) {
	return s.store.ListTags(userID)
}

// UpdateTag 更新标签
//
// 参数:
//   - id: 标签ID
//   - input: 更新输入
//
// 返回值:
//   - *domain.Tag: 更新后的标签
//   - error: 错误信息
func (s *TagService) UpdateTag(id string, input UpdateTagInput) (*domain.Tag, error) {
	tag, err := s.store.GetTag(id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if input.Name != "" {
		// 检查名称冲突
		existing, _ := s.store.GetTagByName(tag.UserID, input.Name)
		if existing != nil && existing.ID != id {
			return nil, fmt.Errorf("tag name already exists")
		}
		tag.Name = input.Name
	}
	if input.Color != "" {
		tag.Color = input.Color
	}
	if input.Description != "" {
		tag.Description = input.Description
	}

	if err := s.store.UpdateTag(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// DeleteTag 删除标签
//
// 参数:
//   - id: 标签ID
//
// 返回值:
//   - error: 错误信息
func (s *TagService) DeleteTag(id string) error {
	return s.store.DeleteTag(id)
}

// AddMessageTag 为邮件添加标签
//
// 参数:
//   - messageID: 邮件ID
//   - tagID: 标签ID
//
// 返回值:
//   - error: 错误信息
func (s *TagService) AddMessageTag(messageID, tagID string) error {
	return s.store.AddMessageTag(messageID, tagID)
}

// RemoveMessageTag 移除邮件标签
//
// 参数:
//   - messageID: 邮件ID
//   - tagID: 标签ID
//
// 返回值:
//   - error: 错误信息
func (s *TagService) RemoveMessageTag(messageID, tagID string) error {
	return s.store.RemoveMessageTag(messageID, tagID)
}

// GetMessageTags 获取邮件的所有标签
//
// 参数:
//   - messageID: 邮件ID
//
// 返回值:
//   - []domain.Tag: 标签列表
//   - error: 错误信息
func (s *TagService) GetMessageTags(messageID string) ([]domain.Tag, error) {
	return s.store.GetMessageTags(messageID)
}

// ListMessagesByTag 列出标签下的所有邮件
//
// 参数:
//   - tagID: 标签ID
//
// 返回值:
//   - []domain.Message: 邮件列表
//   - error: 错误信息
func (s *TagService) ListMessagesByTag(tagID string) ([]domain.Message, error) {
	return s.store.ListMessagesByTag(tagID)
}

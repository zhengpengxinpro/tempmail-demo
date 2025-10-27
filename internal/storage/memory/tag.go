package memory

import (
	"fmt"
	"time"

	"tempmail/backend/internal/domain"
)

// CreateTag 创建标签
func (s *Store) CreateTag(tag *domain.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在同名标签
	for _, existingTag := range s.tags {
		if existingTag.UserID == tag.UserID && existingTag.Name == tag.Name {
			return fmt.Errorf("tag already exists")
		}
	}

	tag.CreatedAt = time.Now()
	tag.UpdatedAt = time.Now()
	s.tags[tag.ID] = tag

	// 按用户索引
	if s.tagsByUser[tag.UserID] == nil {
		s.tagsByUser[tag.UserID] = make(map[string]*domain.Tag)
	}
	s.tagsByUser[tag.UserID][tag.ID] = tag

	return nil
}

// GetTag 获取标签
func (s *Store) GetTag(id string) (*domain.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag, exists := s.tags[id]
	if !exists {
		return nil, fmt.Errorf("tag not found")
	}

	return tag, nil
}

// GetTagByName 根据名称获取标签
func (s *Store) GetTagByName(userID, name string) (*domain.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userTags := s.tagsByUser[userID]
	if userTags == nil {
		return nil, fmt.Errorf("tag not found")
	}

	for _, tag := range userTags {
		if tag.Name == name {
			return tag, nil
		}
	}

	return nil, fmt.Errorf("tag not found")
}

// ListTags 列出用户的所有标签
func (s *Store) ListTags(userID string) ([]domain.TagWithCount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userTags := s.tagsByUser[userID]
	if userTags == nil {
		return []domain.TagWithCount{}, nil
	}

	result := make([]domain.TagWithCount, 0, len(userTags))
	for _, tag := range userTags {
		// 计算该标签下的邮件数量
		count := 0
		for _, mt := range s.messageTags {
			if mt.TagID == tag.ID {
				count++
			}
		}

		result = append(result, domain.TagWithCount{
			Tag:          *tag,
			MessageCount: count,
		})
	}

	return result, nil
}

// UpdateTag 更新标签
func (s *Store) UpdateTag(tag *domain.Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.tags[tag.ID]
	if !exists {
		return fmt.Errorf("tag not found")
	}

	// 检查名称冲突
	for _, t := range s.tags {
		if t.UserID == tag.UserID && t.Name == tag.Name && t.ID != tag.ID {
			return fmt.Errorf("tag name already exists")
		}
	}

	tag.CreatedAt = existing.CreatedAt
	tag.UpdatedAt = time.Now()
	s.tags[tag.ID] = tag
	s.tagsByUser[tag.UserID][tag.ID] = tag

	return nil
}

// DeleteTag 删除标签
func (s *Store) DeleteTag(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tag, exists := s.tags[id]
	if !exists {
		return fmt.Errorf("tag not found")
	}

	// 删除所有关联
	newMessageTags := make(map[string]*domain.MessageTag)
	for key, mt := range s.messageTags {
		if mt.TagID != id {
			newMessageTags[key] = mt
		}
	}
	s.messageTags = newMessageTags

	// 删除标签
	delete(s.tags, id)
	delete(s.tagsByUser[tag.UserID], id)

	return nil
}

// AddMessageTag 为邮件添加标签
func (s *Store) AddMessageTag(messageID, tagID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查标签是否存在
	if _, exists := s.tags[tagID]; !exists {
		return fmt.Errorf("tag not found")
	}

	// 检查是否已存在
	key := messageID + ":" + tagID
	if _, exists := s.messageTags[key]; exists {
		return nil // 已存在，不报错
	}

	messageTag := &domain.MessageTag{
		MessageID: messageID,
		TagID:     tagID,
		CreatedAt: time.Now(),
	}

	s.messageTags[key] = messageTag

	// 按邮件索引
	if s.tagsByMessage[messageID] == nil {
		s.tagsByMessage[messageID] = make(map[string]*domain.MessageTag)
	}
	s.tagsByMessage[messageID][tagID] = messageTag

	return nil
}

// RemoveMessageTag 移除邮件标签
func (s *Store) RemoveMessageTag(messageID, tagID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := messageID + ":" + tagID
	delete(s.messageTags, key)

	if s.tagsByMessage[messageID] != nil {
		delete(s.tagsByMessage[messageID], tagID)
	}

	return nil
}

// GetMessageTags 获取邮件的所有标签
func (s *Store) GetMessageTags(messageID string) ([]domain.Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messageTags := s.tagsByMessage[messageID]
	if messageTags == nil {
		return []domain.Tag{}, nil
	}

	result := make([]domain.Tag, 0, len(messageTags))
	for _, mt := range messageTags {
		if tag, exists := s.tags[mt.TagID]; exists {
			result = append(result, *tag)
		}
	}

	return result, nil
}

// ListMessagesByTag 列出标签下的所有邮件
func (s *Store) ListMessagesByTag(tagID string) ([]domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 收集所有带该标签的邮件ID
	messageIDs := make(map[string]bool)
	for _, mt := range s.messageTags {
		if mt.TagID == tagID {
			messageIDs[mt.MessageID] = true
		}
	}

	// 获取邮件详情
	result := make([]domain.Message, 0)
	for _, mailboxMessages := range s.messages {
		for _, msg := range mailboxMessages {
			if messageIDs[msg.ID] {
				result = append(result, *msg)
			}
		}
	}

	return result, nil
}

// DeleteMessageTags 删除邮件的所有标签
func (s *Store) DeleteMessageTags(messageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 删除所有关联
	messageTags := s.tagsByMessage[messageID]
	if messageTags != nil {
		for tagID := range messageTags {
			key := messageID + ":" + tagID
			delete(s.messageTags, key)
		}
		delete(s.tagsByMessage, messageID)
	}

	return nil
}

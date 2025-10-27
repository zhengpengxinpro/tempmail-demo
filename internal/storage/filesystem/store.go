package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tempmail/backend/internal/domain"
)

// Store 文件系统存储实现
type Store struct {
	basePath      string         // 邮件存储根目录
	platformUtils *PlatformUtils // 平台兼容性工具
}

// NewStore 创建文件系统存储实例
func NewStore(basePath string) (*Store, error) {
	// 创建平台工具
	platformUtils := NewPlatformUtils()

	// 验证基础路径
	if err := platformUtils.ValidatePath(basePath); err != nil {
		return nil, fmt.Errorf("invalid base path: %w", err)
	}

	// 标准化路径
	normalizedPath := platformUtils.NormalizePath(basePath)

	// 确保基础目录存在
	if err := os.MkdirAll(normalizedPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Store{
		basePath:      normalizedPath,
		platformUtils: platformUtils,
	}, nil
}

// ========== 邮件存储 ==========

// SaveMessageRaw 保存邮件原始内容到文件
func (s *Store) SaveMessageRaw(mailboxID, messageID string, rawContent []byte) (string, error) {
	// 创建邮件目录: /data/mails/{mailboxID}/{YYYY-MM-DD}/{messageID}/
	messagePath := s.getMessagePath(mailboxID, messageID)
	if err := os.MkdirAll(messagePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create message directory: %w", err)
	}

	// 保存原始邮件: raw.eml
	rawFile := filepath.Join(messagePath, "raw.eml")
	if err := os.WriteFile(rawFile, rawContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write raw message: %w", err)
	}

	relPath, err := filepath.Rel(s.basePath, rawFile)
	if err != nil {
		return rawFile, nil
	}

	return relPath, nil
}

// GetMessageRaw 读取邮件原始内容
func (s *Store) GetMessageRaw(mailboxID, messageID string) ([]byte, error) {
	rawFile := filepath.Join(s.getMessagePath(mailboxID, messageID), "raw.eml")

	content, err := os.ReadFile(rawFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("message raw content not found")
		}
		return nil, fmt.Errorf("failed to read raw message: %w", err)
	}

	return content, nil
}

// SaveMessageMetadata 保存邮件元数据（JSON）
func (s *Store) SaveMessageMetadata(mailboxID, messageID string, message *domain.Message) (string, error) {
	messagePath := s.getMessagePath(mailboxID, messageID)
	if err := os.MkdirAll(messagePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create message directory: %w", err)
	}

	// 保存元数据: metadata.json
	metaFile := filepath.Join(messagePath, "metadata.json")

	// 创建附件元数据列表（不包含 Content）
	attachmentMetas := make([]struct {
		ID          string `json:"id"`
		MessageID   string `json:"messageId"`
		Filename    string `json:"filename"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
		StoragePath string `json:"storagePath,omitempty"`
	}, len(message.Attachments))

	for i, att := range message.Attachments {
		attachmentMetas[i] = struct {
			ID          string `json:"id"`
			MessageID   string `json:"messageId"`
			Filename    string `json:"filename"`
			ContentType string `json:"contentType"`
			Size        int64  `json:"size"`
			StoragePath string `json:"storagePath,omitempty"`
		}{
			ID:          att.ID,
			MessageID:   att.MessageID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			StoragePath: att.StoragePath,
		}
	}

	// 创建元数据结构（不包含 Raw，但包含 Attachments 元数据）
	meta := struct {
		ID          string    `json:"id"`
		MailboxID   string    `json:"mailboxId"`
		From        string    `json:"from"`
		To          string    `json:"to"`
		Subject     string    `json:"subject"`
		Text        string    `json:"text"`
		HTML        string    `json:"html"`
		CreatedAt   time.Time `json:"createdAt"`
		ReceivedAt  time.Time `json:"receivedAt"`
		IsRead      bool      `json:"isRead"`
		HasRaw      bool      `json:"hasRaw"`
		HasHTML     bool      `json:"hasHtml"`
		HasText     bool      `json:"hasText"`
		Attachments []struct {
			ID          string `json:"id"`
			MessageID   string `json:"messageId"`
			Filename    string `json:"filename"`
			ContentType string `json:"contentType"`
			Size        int64  `json:"size"`
			StoragePath string `json:"storagePath,omitempty"`
		} `json:"attachments,omitempty"`
	}{
		ID:          message.ID,
		MailboxID:   message.MailboxID,
		From:        message.From,
		To:          message.To,
		Subject:     message.Subject,
		Text:        message.Text,
		HTML:        message.HTML,
		CreatedAt:   message.CreatedAt,
		ReceivedAt:  message.ReceivedAt,
		IsRead:      message.IsRead,
		HasRaw:      message.HasRaw,
		HasHTML:     message.HasHTML,
		HasText:     message.HasText,
		Attachments: attachmentMetas,
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaFile, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write metadata: %w", err)
	}

	relPath, err := filepath.Rel(s.basePath, metaFile)
	if err != nil {
		return metaFile, nil
	}

	return relPath, nil
}

// GetMessageMetadata 读取邮件元数据
func (s *Store) GetMessageMetadata(mailboxID, messageID string) (*domain.Message, error) {
	metaFile := filepath.Join(s.getMessagePath(mailboxID, messageID), "metadata.json")

	data, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("message metadata not found")
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var message domain.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &message, nil
}

// ========== 附件存储 ==========

// SaveAttachment 保存邮件附件
func (s *Store) SaveAttachment(mailboxID, messageID, attachmentID string, attachment *domain.Attachment) (string, error) {
	// 创建附件目录: /data/mails/{mailboxID}/{YYYY-MM-DD}/{messageID}/attachments/
	attachPath := filepath.Join(s.getMessagePath(mailboxID, messageID), "attachments")
	if err := os.MkdirAll(attachPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create attachment directory: %w", err)
	}

	// 生成安全的文件名（使用 SHA256 前 16 位 + 原始文件名）
	safeFilename := s.generateSafeFilename(attachmentID, attachment.Filename)
	attachFile := filepath.Join(attachPath, safeFilename)

	// 保存附件内容
	if err := os.WriteFile(attachFile, attachment.Content, 0644); err != nil {
		return "", fmt.Errorf("failed to write attachment: %w", err)
	}

	// 保存附件元数据
	metaFile := filepath.Join(attachPath, safeFilename+".meta.json")
	meta := struct {
		ID          string `json:"id"`
		Filename    string `json:"filename"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
		SavedAt     string `json:"savedAt"`
	}{
		ID:          attachmentID,
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		Size:        attachment.Size,
		SavedAt:     time.Now().Format(time.RFC3339),
	}

	metaData, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaFile, metaData, 0644); err != nil {
		return "", fmt.Errorf("failed to write attachment metadata: %w", err)
	}

	// 返回相对存储路径（便于持久化到数据库）
	relPath, err := filepath.Rel(s.basePath, attachFile)
	if err != nil {
		// 如果计算相对路径失败，仍返回绝对路径
		return attachFile, nil
	}

	return relPath, nil
}

// GetAttachment 读取邮件附件
func (s *Store) GetAttachment(mailboxID, messageID, attachmentID string) (*domain.Attachment, error) {
	// 先读取邮件元数据以获取附件信息
	metadata, err := s.GetMessageMetadata(mailboxID, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message metadata: %w", err)
	}

	// 查找指定附件的元数据
	var attachmentMeta *domain.Attachment
	for _, att := range metadata.Attachments {
		if att.ID == attachmentID {
			attachmentMeta = att
			break
		}
	}
	if attachmentMeta == nil {
		return nil, fmt.Errorf("attachment not found in metadata")
	}

	// 读取附件文件内容
	safeFilename := s.generateSafeFilename(attachmentID, attachmentMeta.Filename)
	attachFile := filepath.Join(s.getMessagePath(mailboxID, messageID), "attachments", safeFilename)

	content, err := os.ReadFile(attachFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("attachment file not found")
		}
		return nil, fmt.Errorf("failed to read attachment: %w", err)
	}

	// 填充内容并返回完整的附件对象
	attachmentMeta.Content = content
	return attachmentMeta, nil
}

// ========== 清理操作 ==========

// DeleteMessage 删除邮件及其所有文件
func (s *Store) DeleteMessage(mailboxID, messageID string) error {
	messagePath := s.getMessagePath(mailboxID, messageID)
	return os.RemoveAll(messagePath)
}

// DeleteMailbox 删除邮箱的所有邮件
func (s *Store) DeleteMailbox(mailboxID string) error {
	mailboxPath := filepath.Join(s.basePath, "mails", mailboxID)
	return os.RemoveAll(mailboxPath)
}

// CleanupExpired 清理过期的邮件（基于目录的修改时间）
func (s *Store) CleanupExpired(retentionDays int) (int, error) {
	count := 0
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	mailsPath := filepath.Join(s.basePath, "mails")

	// 遍历所有邮箱目录
	mailboxDirs, err := os.ReadDir(mailsPath)
	if err != nil {
		return 0, err
	}

	for _, mailboxDir := range mailboxDirs {
		if !mailboxDir.IsDir() {
			continue
		}

		mailboxPath := filepath.Join(mailsPath, mailboxDir.Name())

		// 遍历日期目录
		dateDirs, err := os.ReadDir(mailboxPath)
		if err != nil {
			continue
		}

		for _, dateDir := range dateDirs {
			if !dateDir.IsDir() {
				continue
			}

			datePath := filepath.Join(mailboxPath, dateDir.Name())

			// 遍历邮件目录
			messageDirs, err := os.ReadDir(datePath)
			if err != nil {
				continue
			}

			for _, messageDir := range messageDirs {
				if !messageDir.IsDir() {
					continue
				}

				messagePath := filepath.Join(datePath, messageDir.Name())

				// 检查目录的修改时间
				info, err := os.Stat(messagePath)
				if err != nil {
					continue
				}

				if info.ModTime().Before(cutoffTime) {
					if err := os.RemoveAll(messagePath); err == nil {
						count++
					}
				}
			}

			// 如果日期目录为空，删除它
			if entries, _ := os.ReadDir(datePath); len(entries) == 0 {
				os.Remove(datePath)
			}
		}

		// 如果邮箱目录为空，删除它
		if entries, _ := os.ReadDir(mailboxPath); len(entries) == 0 {
			os.Remove(mailboxPath)
		}
	}

	return count, nil
}

// ========== 辅助方法 ==========

// getMessagePath 获取邮件存储路径
// 格式: /data/mails/{mailboxID}/{YYYY-MM-DD}/{messageID}/
func (s *Store) getMessagePath(mailboxID, messageID string) string {
	today := time.Now().Format("2006-01-02")
	return filepath.Join(s.basePath, "mails", mailboxID, today, messageID)
}

// generateSafeFilename 生成安全的文件名
func (s *Store) generateSafeFilename(attachmentID, originalFilename string) string {
	// 使用附件ID的前8位作为前缀，避免文件名冲突
	// 如果 ID 不足 8 位，使用全部
	prefix := attachmentID
	if len(attachmentID) > 8 {
		prefix = attachmentID[:8]
	}

	// 使用平台工具清理文件名
	safeFilename := s.platformUtils.SanitizeFilename(originalFilename)

	return fmt.Sprintf("%s_%s", prefix, safeFilename)
}

// GetStorageStats 获取存储统计信息
func (s *Store) GetStorageStats() (map[string]interface{}, error) {
	mailsPath := filepath.Join(s.basePath, "mails")

	var totalSize int64
	var messageCount int
	var attachmentCount int

	err := filepath.Walk(mailsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过错误，继续遍历
		}

		if !info.IsDir() {
			totalSize += info.Size()

			if filepath.Ext(path) == ".eml" {
				messageCount++
			}

			if filepath.Base(filepath.Dir(path)) == "attachments" &&
				filepath.Ext(path) != ".json" {
				attachmentCount++
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_size_bytes": totalSize,
		"total_size_mb":    float64(totalSize) / 1024 / 1024,
		"message_count":    messageCount,
		"attachment_count": attachmentCount,
		"base_path":        s.basePath,
	}, nil
}

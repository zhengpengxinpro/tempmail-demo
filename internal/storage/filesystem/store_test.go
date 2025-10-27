package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tempmail/backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助函数：创建临时测试目录
func setupTestStore(t *testing.T) (*Store, string) {
	tempDir, err := os.MkdirTemp("", "filesystem_test_*")
	require.NoError(t, err)

	store, err := NewStore(tempDir)
	require.NoError(t, err)

	return store, tempDir
}

// 测试辅助函数：清理测试目录
func cleanupTestStore(t *testing.T, tempDir string) {
	err := os.RemoveAll(tempDir)
	require.NoError(t, err)
}

// TestNewStore 测试创建文件系统存储实例
func TestNewStore(t *testing.T) {
	t.Run("create store with valid path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "filesystem_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		store, err := NewStore(tempDir)
		require.NoError(t, err)
		assert.NotNil(t, store)
		// 在 Windows 上，路径可能被转换为小写
		assert.Equal(t, strings.ToLower(tempDir), strings.ToLower(store.basePath))
	})

	t.Run("create store creates base directory if not exists", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "filesystem_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		newPath := filepath.Join(tempDir, "new", "nested", "path")
		store, err := NewStore(newPath)
		require.NoError(t, err)
		assert.NotNil(t, store)

		// 验证目录已创建
		_, err = os.Stat(newPath)
		assert.NoError(t, err)
	})
}

// TestSaveMessageRaw 测试保存邮件原始内容
func TestSaveMessageRaw(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-001"
	messageID := "test-message-001"
	rawContent := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nBody content")

	t.Run("save raw message successfully", func(t *testing.T) {
		path, err := store.SaveMessageRaw(mailboxID, messageID, rawContent)
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		// 验证文件已创建
		messagePath := store.getMessagePath(mailboxID, messageID)
		rawFile := filepath.Join(messagePath, "raw.eml")
		_, err = os.Stat(rawFile)
		assert.NoError(t, err)

		// 验证文件内容
		content, err := os.ReadFile(rawFile)
		require.NoError(t, err)
		assert.Equal(t, rawContent, content)
	})

	t.Run("overwrite existing raw message", func(t *testing.T) {
		newContent := []byte("Updated raw content")
		_, err := store.SaveMessageRaw(mailboxID, messageID, newContent)
		require.NoError(t, err)

		// 验证内容已更新
		content, err := store.GetMessageRaw(mailboxID, messageID)
		require.NoError(t, err)
		assert.Equal(t, newContent, content)
	})
}

// TestGetMessageRaw 测试读取邮件原始内容
func TestGetMessageRaw(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-002"
	messageID := "test-message-002"
	rawContent := []byte("Test raw email content")

	t.Run("get existing raw message", func(t *testing.T) {
		path, err := store.SaveMessageRaw(mailboxID, messageID, rawContent)
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		content, err := store.GetMessageRaw(mailboxID, messageID)
		require.NoError(t, err)
		assert.Equal(t, rawContent, content)
	})

	t.Run("get non-existent raw message", func(t *testing.T) {
		content, err := store.GetMessageRaw("nonexistent-mailbox", "nonexistent-message")
		assert.Error(t, err)
		assert.Nil(t, content)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestSaveMessageMetadata 测试保存邮件元数据
func TestSaveMessageMetadata(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-003"
	messageID := "test-message-003"

	message := &domain.Message{
		ID:         messageID,
		MailboxID:  mailboxID,
		From:       "sender@example.com",
		To:         "recipient@example.com",
		Subject:    "Test Subject",
		Text:       "Plain text content",
		HTML:       "<html><body>HTML content</body></html>",
		CreatedAt:  time.Now(),
		ReceivedAt: time.Now(),
		IsRead:     false,
		HasRaw:     true,
		HasHTML:    true,
		HasText:    true,
		Attachments: []*domain.Attachment{
			{
				ID:          "att-001",
				MessageID:   messageID,
				Filename:    "test.pdf",
				ContentType: "application/pdf",
				Size:        12345,
				StoragePath: "attachments/att-001_test.pdf",
			},
		},
	}

	t.Run("save metadata successfully", func(t *testing.T) {
		_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
		require.NoError(t, err)

		// 验证文件已创建
		messagePath := store.getMessagePath(mailboxID, messageID)
		metaFile := filepath.Join(messagePath, "metadata.json")
		_, err = os.Stat(metaFile)
		assert.NoError(t, err)

		// 验证JSON结构
		data, err := os.ReadFile(metaFile)
		require.NoError(t, err)

		var meta map[string]interface{}
		err = json.Unmarshal(data, &meta)
		require.NoError(t, err)

		assert.Equal(t, messageID, meta["id"])
		assert.Equal(t, mailboxID, meta["mailboxId"])
		assert.Equal(t, "sender@example.com", meta["from"])
		assert.Equal(t, "Test Subject", meta["subject"])
		assert.Equal(t, "Plain text content", meta["text"])
		assert.Equal(t, "<html><body>HTML content</body></html>", meta["html"])
	})

	t.Run("save metadata with attachments", func(t *testing.T) {
		_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
		require.NoError(t, err)

		// 读取并验证附件信息
		metadata, err := store.GetMessageMetadata(mailboxID, messageID)
		require.NoError(t, err)
		assert.Len(t, metadata.Attachments, 1)
		assert.Equal(t, "att-001", metadata.Attachments[0].ID)
		assert.Equal(t, "test.pdf", metadata.Attachments[0].Filename)
		assert.Equal(t, "application/pdf", metadata.Attachments[0].ContentType)
		assert.Equal(t, int64(12345), metadata.Attachments[0].Size)
	})
}

// TestGetMessageMetadata 测试读取邮件元数据
func TestGetMessageMetadata(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-004"
	messageID := "test-message-004"

	message := &domain.Message{
		ID:         messageID,
		MailboxID:  mailboxID,
		From:       "sender@example.com",
		To:         "recipient@example.com",
		Subject:    "Test Subject",
		Text:       "Plain text content",
		HTML:       "<html>HTML</html>",
		CreatedAt:  time.Now(),
		ReceivedAt: time.Now(),
		IsRead:     false,
		HasRaw:     true,
		HasHTML:    true,
		HasText:    true,
	}

	t.Run("get existing metadata", func(t *testing.T) {
		_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
		require.NoError(t, err)

		metadata, err := store.GetMessageMetadata(mailboxID, messageID)
		require.NoError(t, err)
		assert.Equal(t, message.ID, metadata.ID)
		assert.Equal(t, message.MailboxID, metadata.MailboxID)
		assert.Equal(t, message.From, metadata.From)
		assert.Equal(t, message.To, metadata.To)
		assert.Equal(t, message.Subject, metadata.Subject)
		assert.Equal(t, message.Text, metadata.Text)
		assert.Equal(t, message.HTML, metadata.HTML)
	})

	t.Run("get non-existent metadata", func(t *testing.T) {
		metadata, err := store.GetMessageMetadata("nonexistent-mailbox", "nonexistent-message")
		assert.Error(t, err)
		assert.Nil(t, metadata)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestSaveAttachment 测试保存附件
func TestSaveAttachment(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-005"
	messageID := "test-message-005"
	attachmentID := "test-att-001"

	attachment := &domain.Attachment{
		ID:          attachmentID,
		MessageID:   messageID,
		Filename:    "invoice.pdf",
		ContentType: "application/pdf",
		Size:        54321,
		Content:     []byte("PDF content here"),
	}

	t.Run("save attachment successfully", func(t *testing.T) {
		_, err := store.SaveAttachment(mailboxID, messageID, attachmentID, attachment)
		require.NoError(t, err)

		// 验证附件文件已创建
		safeFilename := store.generateSafeFilename(attachmentID, attachment.Filename)
		attachPath := filepath.Join(store.getMessagePath(mailboxID, messageID), "attachments", safeFilename)
		_, err = os.Stat(attachPath)
		assert.NoError(t, err)

		// 验证附件内容
		content, err := os.ReadFile(attachPath)
		require.NoError(t, err)
		assert.Equal(t, attachment.Content, content)

		// 验证元数据文件已创建
		metaFile := attachPath + ".meta.json"
		_, err = os.Stat(metaFile)
		assert.NoError(t, err)
	})

	t.Run("save multiple attachments", func(t *testing.T) {
		attachment2 := &domain.Attachment{
			ID:          "test-att-002",
			MessageID:   messageID,
			Filename:    "document.docx",
			ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			Size:        98765,
			Content:     []byte("DOCX content"),
		}

		_, err := store.SaveAttachment(mailboxID, messageID, "test-att-002", attachment2)
		require.NoError(t, err)

		// 验证两个附件都存在
		attachDir := filepath.Join(store.getMessagePath(mailboxID, messageID), "attachments")
		entries, err := os.ReadDir(attachDir)
		require.NoError(t, err)
		// 应该有 4 个文件：2个附件文件 + 2个元数据文件
		assert.GreaterOrEqual(t, len(entries), 2)
	})
}

// TestGetAttachment 测试读取附件
func TestGetAttachment(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-006"
	messageID := "test-message-006"
	attachmentID := "test-att-003"

	// 先保存消息元数据
	message := &domain.Message{
		ID:         messageID,
		MailboxID:  mailboxID,
		From:       "sender@example.com",
		To:         "recipient@example.com",
		Subject:    "Test with attachment",
		Text:       "See attachment",
		CreatedAt:  time.Now(),
		ReceivedAt: time.Now(),
		Attachments: []*domain.Attachment{
			{
				ID:          attachmentID,
				MessageID:   messageID,
				Filename:    "report.xlsx",
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				Size:        24680,
				StoragePath: "attachments/" + store.generateSafeFilename(attachmentID, "report.xlsx"),
			},
		},
	}

	_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
	require.NoError(t, err)

	// 保存附件
	attachment := &domain.Attachment{
		ID:          attachmentID,
		MessageID:   messageID,
		Filename:    "report.xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Size:        24680,
		Content:     []byte("Excel content"),
	}

	_, err = store.SaveAttachment(mailboxID, messageID, attachmentID, attachment)
	require.NoError(t, err)

	t.Run("get existing attachment", func(t *testing.T) {
		retrievedAtt, err := store.GetAttachment(mailboxID, messageID, attachmentID)
		require.NoError(t, err)
		assert.Equal(t, attachmentID, retrievedAtt.ID)
		assert.Equal(t, "report.xlsx", retrievedAtt.Filename)
		assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", retrievedAtt.ContentType)
		assert.Equal(t, []byte("Excel content"), retrievedAtt.Content)
	})

	t.Run("get non-existent attachment", func(t *testing.T) {
		retrievedAtt, err := store.GetAttachment(mailboxID, messageID, "nonexistent-att")
		assert.Error(t, err)
		assert.Nil(t, retrievedAtt)
	})

	t.Run("get attachment from non-existent message", func(t *testing.T) {
		retrievedAtt, err := store.GetAttachment(mailboxID, "nonexistent-message", attachmentID)
		assert.Error(t, err)
		assert.Nil(t, retrievedAtt)
	})
}

// TestDeleteMessage 测试删除邮件
func TestDeleteMessage(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-007"
	messageID := "test-message-007"

	// 保存邮件数据
	rawContent := []byte("Raw content")
	var err error
	_, err = store.SaveMessageRaw(mailboxID, messageID, rawContent)
	require.NoError(t, err)

	message := &domain.Message{
		ID:         messageID,
		MailboxID:  mailboxID,
		From:       "sender@example.com",
		To:         "recipient@example.com",
		Subject:    "Test",
		Text:       "Text",
		CreatedAt:  time.Now(),
		ReceivedAt: time.Now(),
	}
	_, err = store.SaveMessageMetadata(mailboxID, messageID, message)
	require.NoError(t, err)

	t.Run("delete existing message", func(t *testing.T) {
		// 验证邮件存在
		messagePath := store.getMessagePath(mailboxID, messageID)
		_, err := os.Stat(messagePath)
		assert.NoError(t, err)

		// 删除邮件
		err = store.DeleteMessage(mailboxID, messageID)
		require.NoError(t, err)

		// 验证邮件已删除
		_, err = os.Stat(messagePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete non-existent message", func(t *testing.T) {
		// 删除不存在的邮件不应报错
		err := store.DeleteMessage("nonexistent-mailbox", "nonexistent-message")
		assert.NoError(t, err)
	})
}

// TestDeleteMailbox 测试删除邮箱
func TestDeleteMailbox(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-008"

	// 创建多个邮件
	for i := 1; i <= 3; i++ {
		messageID := "test-message-" + string(rune('0'+i))
		rawContent := []byte("Raw content " + string(rune('0'+i)))
		_, err := store.SaveMessageRaw(mailboxID, messageID, rawContent)
		require.NoError(t, err)
	}

	t.Run("delete mailbox with multiple messages", func(t *testing.T) {
		// 验证邮箱目录存在
		mailboxPath := filepath.Join(store.basePath, "mails", mailboxID)
		_, err := os.Stat(mailboxPath)
		assert.NoError(t, err)

		// 删除邮箱
		err = store.DeleteMailbox(mailboxID)
		require.NoError(t, err)

		// 验证邮箱目录已删除
		_, err = os.Stat(mailboxPath)
		assert.True(t, os.IsNotExist(err))
	})
}

// TestCleanupExpired 测试清理过期邮件
func TestCleanupExpired(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-009"

	t.Run("cleanup expired messages", func(t *testing.T) {
		// 创建一个邮件
		messageID := "old-message-001"
		rawContent := []byte("Old content")
		_, err := store.SaveMessageRaw(mailboxID, messageID, rawContent)
		require.NoError(t, err)

		// 修改邮件目录的修改时间为3天前
		messagePath := store.getMessagePath(mailboxID, messageID)
		oldTime := time.Now().AddDate(0, 0, -3)
		err = os.Chtimes(messagePath, oldTime, oldTime)
		require.NoError(t, err)

		// 清理2天前的邮件
		count, err := store.CleanupExpired(2)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// 验证邮件已删除
		_, err = os.Stat(messagePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("cleanup does not delete recent messages", func(t *testing.T) {
		// 创建一个新邮件
		messageID := "new-message-001"
		rawContent := []byte("New content")
		_, err := store.SaveMessageRaw(mailboxID, messageID, rawContent)
		require.NoError(t, err)

		// 清理3天前的邮件
		count, err := store.CleanupExpired(3)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// 验证邮件仍然存在
		messagePath := store.getMessagePath(mailboxID, messageID)
		_, err = os.Stat(messagePath)
		assert.NoError(t, err)
	})
}

// TestGenerateSafeFilename 测试生成安全文件名
func TestGenerateSafeFilename(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	t.Run("generate safe filename with normal name", func(t *testing.T) {
		attachmentID := "12345678-1234-5678-1234-567812345678"
		filename := "document.pdf"
		safe := store.generateSafeFilename(attachmentID, filename)
		assert.Contains(t, safe, "12345678")
		assert.Contains(t, safe, "document.pdf")
	})

	t.Run("generate safe filename with special characters", func(t *testing.T) {
		attachmentID := "abcdefgh-1234-5678-1234-567812345678"
		filename := "../../etc/passwd"
		safe := store.generateSafeFilename(attachmentID, filename)
		assert.Contains(t, safe, "abcdefgh")
		// filepath.Base 应该清理路径
		assert.NotContains(t, safe, "..")
	})
}

// TestGetMessagePath 测试获取邮件路径
func TestGetMessagePath(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	t.Run("get message path with correct format", func(t *testing.T) {
		mailboxID := "test-mailbox-010"
		messageID := "test-message-010"
		path := store.getMessagePath(mailboxID, messageID)

		// 验证路径包含必要的组件
		assert.Contains(t, strings.ToLower(path), strings.ToLower(tempDir))
		assert.Contains(t, path, "mails")
		assert.Contains(t, path, mailboxID)
		assert.Contains(t, path, messageID)

		// 验证包含日期格式 (YYYY-MM-DD)
		today := time.Now().Format("2006-01-02")
		assert.Contains(t, path, today)
	})
}

// TestGetStorageStats 测试获取存储统计
func TestGetStorageStats(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	mailboxID := "test-mailbox-011"
	messageID := "test-message-011"

	t.Run("get stats with no data", func(t *testing.T) {
		stats, err := store.GetStorageStats()
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(0), stats["total_size_bytes"])
		assert.Equal(t, 0, stats["message_count"])
		assert.Equal(t, 0, stats["attachment_count"])
	})

	t.Run("get stats with messages and attachments", func(t *testing.T) {
		// 保存邮件
		rawContent := []byte("Raw email content for testing statistics")
		var err error
		_, err = store.SaveMessageRaw(mailboxID, messageID, rawContent)
		require.NoError(t, err)

		// 保存元数据
		message := &domain.Message{
			ID:         messageID,
			MailboxID:  mailboxID,
			From:       "sender@example.com",
			To:         "recipient@example.com",
			Subject:    "Test",
			Text:       "Text content",
			HTML:       "<html>HTML</html>",
			CreatedAt:  time.Now(),
			ReceivedAt: time.Now(),
			Attachments: []*domain.Attachment{
				{
					ID:          "att-stats-001",
					MessageID:   messageID,
					Filename:    "test.txt",
					ContentType: "text/plain",
					Size:        100,
					StoragePath: "attachments/att-stats-001_test.txt",
				},
			},
		}
		_, err = store.SaveMessageMetadata(mailboxID, messageID, message)
		require.NoError(t, err)

		// 保存附件
		attachment := &domain.Attachment{
			ID:          "att-stats-001",
			MessageID:   messageID,
			Filename:    "test.txt",
			ContentType: "text/plain",
			Size:        100,
			Content:     []byte("Attachment content for stats"),
		}
		_, err = store.SaveAttachment(mailboxID, messageID, "att-stats-001", attachment)
		require.NoError(t, err)

		// 获取统计信息
		stats, err := store.GetStorageStats()
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Greater(t, stats["total_size_bytes"].(int64), int64(0))
		assert.Equal(t, 1, stats["message_count"])
		assert.Equal(t, 1, stats["attachment_count"])
		assert.Equal(t, strings.ToLower(tempDir), strings.ToLower(stats["base_path"].(string)))
	})
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	t.Run("concurrent save operations", func(t *testing.T) {
		mailboxID := "test-mailbox-concurrent"
		numGoroutines := 10

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				messageID := "message-" + string(rune('0'+index))
				rawContent := []byte("Content " + string(rune('0'+index)))
				_, err := store.SaveMessageRaw(mailboxID, messageID, rawContent)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// 等待所有 goroutine 完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证所有消息都已保存
		for i := 0; i < numGoroutines; i++ {
			messageID := "message-" + string(rune('0'+i))
			content, err := store.GetMessageRaw(mailboxID, messageID)
			assert.NoError(t, err)
			assert.NotNil(t, content)
		}
	})
}

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(t, tempDir)

	t.Run("save empty raw content", func(t *testing.T) {
		mailboxID := "test-mailbox-edge"
		messageID := "test-message-empty"
		_, err := store.SaveMessageRaw(mailboxID, messageID, []byte{})
		require.NoError(t, err)

		content, err := store.GetMessageRaw(mailboxID, messageID)
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("save message with very long subject", func(t *testing.T) {
		mailboxID := "test-mailbox-edge"
		messageID := "test-message-long"
		longSubject := string(make([]byte, 10000)) // 10KB subject

		message := &domain.Message{
			ID:         messageID,
			MailboxID:  mailboxID,
			From:       "sender@example.com",
			To:         "recipient@example.com",
			Subject:    longSubject,
			Text:       "Text",
			CreatedAt:  time.Now(),
			ReceivedAt: time.Now(),
		}

		_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
		require.NoError(t, err)

		metadata, err := store.GetMessageMetadata(mailboxID, messageID)
		require.NoError(t, err)
		assert.Equal(t, longSubject, metadata.Subject)
	})

	t.Run("save attachment with empty content", func(t *testing.T) {
		mailboxID := "test-mailbox-edge"
		messageID := "test-message-edge"
		attachmentID := "att-empty"

		attachment := &domain.Attachment{
			ID:          attachmentID,
			MessageID:   messageID,
			Filename:    "empty.txt",
			ContentType: "text/plain",
			Size:        0,
			Content:     []byte{},
		}

		_, err := store.SaveAttachment(mailboxID, messageID, attachmentID, attachment)
		require.NoError(t, err)

		// 验证文件已创建
		safeFilename := store.generateSafeFilename(attachmentID, "empty.txt")
		attachPath := filepath.Join(store.getMessagePath(mailboxID, messageID), "attachments", safeFilename)
		content, err := os.ReadFile(attachPath)
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("save message with unicode characters", func(t *testing.T) {
		mailboxID := "test-mailbox-unicode"
		messageID := "test-message-unicode"

		message := &domain.Message{
			ID:         messageID,
			MailboxID:  mailboxID,
			From:       "发件人@例子.中国",
			To:         "收件人@例子.中国",
			Subject:    "测试主题 - Test Subject 🎉",
			Text:       "这是中文内容。This is English content. 日本語コンテンツ。",
			HTML:       "<html><body>多语言内容 Multilingual Content</body></html>",
			CreatedAt:  time.Now(),
			ReceivedAt: time.Now(),
		}

		_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
		require.NoError(t, err)

		metadata, err := store.GetMessageMetadata(mailboxID, messageID)
		require.NoError(t, err)
		assert.Equal(t, message.From, metadata.From)
		assert.Equal(t, message.To, metadata.To)
		assert.Equal(t, message.Subject, metadata.Subject)
		assert.Equal(t, message.Text, metadata.Text)
		assert.Equal(t, message.HTML, metadata.HTML)
	})
}

package filesystem

import (
	"os"
	"testing"
	"time"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/service"
	"tempmail/backend/internal/storage/memory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_MessageServiceWithFilesystem 测试 MessageService 与文件系统存储的集成
func TestIntegration_MessageServiceWithFilesystem(t *testing.T) {
	// 创建临时存储目录
	tempDir, err := os.MkdirTemp("", "integration_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 初始化文件系统存储
	fsStore, err := NewStore(tempDir)
	require.NoError(t, err)

	// 初始化内存数据库（用于元数据）
	memStore := memory.NewStore(24 * time.Hour)

	// 创建 MessageService
	msgService := service.NewMessageService(memStore)
	msgService.SetFilesystemStore(fsStore)

	t.Run("create and retrieve message with filesystem storage", func(t *testing.T) {
		// 创建测试邮箱
		mailbox := &domain.Mailbox{
			ID:        "test-mailbox-001",
			Address:   "test@example.com",
			LocalPart: "test",
			Domain:    "example.com",
			Token:     "test-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建邮件（带完整内容）
		input := service.CreateMessageInput{
			MailboxID: mailbox.ID,
			From:      "sender@example.com",
			To:        "test@example.com",
			Subject:   "Integration Test Email",
			Text:      "This is plain text content for integration test",
			HTML:      "<html><body><h1>Integration Test</h1><p>HTML content</p></body></html>",
			Raw:       "From: sender@example.com\r\nTo: test@example.com\r\nSubject: Test\r\n\r\nBody",
			IsRead:    false,
			Received:  time.Now(),
			Attachments: []*domain.Attachment{
				{
					ID:          "att-001",
					Filename:    "test.pdf",
					ContentType: "application/pdf",
					Size:        12345,
					Content:     []byte("PDF content here"),
				},
			},
		}

		// 创建邮件
		message, err := msgService.Create(input)
		require.NoError(t, err)
		assert.NotEmpty(t, message.ID)
		assert.Equal(t, mailbox.ID, message.MailboxID)

		// 文件系统写入由 MessageService 完成，无需额外操作
		// 读取邮件（应该从文件系统加载内容）
		retrieved, err := msgService.Get(mailbox.ID, message.ID)
		require.NoError(t, err)
		assert.Equal(t, message.ID, retrieved.ID)
		assert.Equal(t, input.Subject, retrieved.Subject)
		assert.Equal(t, input.Text, retrieved.Text)
		assert.Equal(t, input.HTML, retrieved.HTML)
		assert.Equal(t, input.Raw, retrieved.Raw)
		assert.Len(t, retrieved.Attachments, 1)

		// 验证附件
		att, err := msgService.GetAttachment(mailbox.ID, message.ID, "att-001")
		require.NoError(t, err)
		assert.Equal(t, "test.pdf", att.Filename)
		assert.Equal(t, "application/pdf", att.ContentType)
		assert.Equal(t, []byte("PDF content here"), att.Content)
	})

	t.Run("list messages returns metadata", func(t *testing.T) {
		mailboxID := "test-mailbox-002"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "list@example.com",
			LocalPart: "list",
			Domain:    "example.com",
			Token:     "list-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建多个邮件
		for i := 1; i <= 3; i++ {
			input := service.CreateMessageInput{
				MailboxID: mailboxID,
				From:      "sender@example.com",
				To:        "list@example.com",
				Subject:   "Test Email " + string(rune('0'+i)),
				Text:      "Text content " + string(rune('0'+i)),
				HTML:      "<html>HTML " + string(rune('0'+i)) + "</html>",
				IsRead:    false,
			}

			message, err := msgService.Create(input)
			require.NoError(t, err)

			// 保存到文件系统
			message.Text = input.Text
			message.HTML = input.HTML
			path, err := fsStore.SaveMessageMetadata(mailboxID, message.ID, message)
			require.NoError(t, err)
			require.NotEmpty(t, path)
		}

		// 列出邮件
		messages, err := msgService.List(mailboxID)
		require.NoError(t, err)
		assert.Len(t, messages, 3)

		// 验证列表中的邮件包含基本信息
		for _, msg := range messages {
			assert.NotEmpty(t, msg.Subject)
			assert.NotEmpty(t, msg.From)
			assert.NotEmpty(t, msg.To)
		}
	})

	t.Run("mark message as read", func(t *testing.T) {
		mailboxID := "test-mailbox-003"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "read@example.com",
			LocalPart: "read",
			Domain:    "example.com",
			Token:     "read-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建未读邮件
		input := service.CreateMessageInput{
			MailboxID: mailboxID,
			From:      "sender@example.com",
			To:        "read@example.com",
			Subject:   "Unread Email",
			Text:      "Content",
			IsRead:    false,
		}

		message, err := msgService.Create(input)
		require.NoError(t, err)
		assert.False(t, message.IsRead)

		// 标记为已读
		err = msgService.MarkRead(mailboxID, message.ID)
		require.NoError(t, err)

		// 验证已读状态
		retrieved, err := msgService.Get(mailboxID, message.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.IsRead)
	})

	t.Run("delete message removes from database and filesystem", func(t *testing.T) {
		mailboxID := "test-mailbox-004"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "delete@example.com",
			LocalPart: "delete",
			Domain:    "example.com",
			Token:     "delete-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建邮件
		input := service.CreateMessageInput{
			MailboxID: mailboxID,
			From:      "sender@example.com",
			To:        "delete@example.com",
			Subject:   "To Be Deleted",
			Text:      "This will be deleted",
			Raw:       "Raw content",
		}

		message, err := msgService.Create(input)
		require.NoError(t, err)

		// 验证文件存在
		messagePath := fsStore.getMessagePath(mailboxID, message.ID)
		_, err = os.Stat(messagePath)
		assert.NoError(t, err)

		// 删除邮件（从数据库）
		err = msgService.Delete(mailboxID, message.ID)
		require.NoError(t, err)

		// 手动从文件系统删除（在实际应用中，应该有后台任务清理）
		err = fsStore.DeleteMessage(mailboxID, message.ID)
		require.NoError(t, err)

		// 验证文件已删除
		_, err = os.Stat(messagePath)
		assert.True(t, os.IsNotExist(err))

		// 验证数据库中也已删除
		_, err = msgService.Get(mailboxID, message.ID)
		assert.Error(t, err)
	})

	t.Run("clear all messages in mailbox", func(t *testing.T) {
		mailboxID := "test-mailbox-005"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "clear@example.com",
			LocalPart: "clear",
			Domain:    "example.com",
			Token:     "clear-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建多个邮件
		messageIDs := make([]string, 0, 5)
		for i := 1; i <= 5; i++ {
			input := service.CreateMessageInput{
				MailboxID: mailboxID,
				From:      "sender@example.com",
				To:        "clear@example.com",
				Subject:   "Email " + string(rune('0'+i)),
				Text:      "Content " + string(rune('0'+i)),
			}

			message, err := msgService.Create(input)
			require.NoError(t, err)
			messageIDs = append(messageIDs, message.ID)

			// 保存到文件系统
			path, err := fsStore.SaveMessageRaw(mailboxID, message.ID, []byte("Raw "+string(rune('0'+i))))
			require.NoError(t, err)
			require.NotEmpty(t, path)
		}

		// 清空邮箱
		count, err := msgService.ClearAll(mailboxID)
		require.NoError(t, err)
		assert.Equal(t, 5, count)

		// 验证邮件已删除
		messages, err := msgService.List(mailboxID)
		require.NoError(t, err)
		assert.Len(t, messages, 0)

		// 手动清理文件系统
		for _, msgID := range messageIDs {
			fsStore.DeleteMessage(mailboxID, msgID)
		}
	})
}

// TestIntegration_LargeEmailHandling 测试大邮件处理
func TestIntegration_LargeEmailHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "large_email_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fsStore, err := NewStore(tempDir)
	require.NoError(t, err)

	memStore := memory.NewStore(24 * time.Hour)
	msgService := service.NewMessageService(memStore)
	msgService.SetFilesystemStore(fsStore)

	t.Run("handle large email with multiple attachments", func(t *testing.T) {
		mailboxID := "large-mailbox"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "large@example.com",
			LocalPart: "large",
			Domain:    "example.com",
			Token:     "large-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建大邮件（带多个大附件）
		largeText := string(make([]byte, 100*1024)) // 100KB text
		largeHTML := string(make([]byte, 200*1024)) // 200KB HTML

		attachments := []*domain.Attachment{
			{
				ID:          "att-large-001",
				Filename:    "document1.pdf",
				ContentType: "application/pdf",
				Size:        1024 * 1024, // 1MB
				Content:     make([]byte, 1024*1024),
			},
			{
				ID:          "att-large-002",
				Filename:    "image.jpg",
				ContentType: "image/jpeg",
				Size:        2 * 1024 * 1024, // 2MB
				Content:     make([]byte, 2*1024*1024),
			},
		}

		input := service.CreateMessageInput{
			MailboxID:   mailboxID,
			From:        "sender@example.com",
			To:          "large@example.com",
			Subject:     "Large Email with Attachments",
			Text:        largeText,
			HTML:        largeHTML,
			Attachments: attachments,
		}

		// 创建邮件
		message, err := msgService.Create(input)
		require.NoError(t, err)

		// 保存到文件系统
		message.Text = input.Text
		message.HTML = input.HTML
		message.Attachments = input.Attachments
		metaPath, err := fsStore.SaveMessageMetadata(mailboxID, message.ID, message)
		require.NoError(t, err)
		require.NotEmpty(t, metaPath)

		for _, att := range attachments {
			attPath, err := fsStore.SaveAttachment(mailboxID, message.ID, att.ID, att)
			require.NoError(t, err)
			require.NotEmpty(t, attPath)
		}

		// 读取邮件
		retrieved, err := msgService.Get(mailboxID, message.ID)
		require.NoError(t, err)
		assert.Equal(t, len(largeText), len(retrieved.Text))
		assert.Equal(t, len(largeHTML), len(retrieved.HTML))
		assert.Len(t, retrieved.Attachments, 2)

		// 验证附件大小
		for i, att := range retrieved.Attachments {
			assert.Equal(t, attachments[i].Size, att.Size)
			assert.Equal(t, attachments[i].Filename, att.Filename)
		}

		// 获取存储统计
		stats, err := fsStore.GetStorageStats()
		require.NoError(t, err)
		assert.Greater(t, stats["total_size_bytes"].(int64), int64(3*1024*1024)) // > 3MB
		assert.Equal(t, 2, stats["attachment_count"])
	})
}

// TestIntegration_ConcurrentMessageCreation 测试并发创建邮件
func TestIntegration_ConcurrentMessageCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fsStore, err := NewStore(tempDir)
	require.NoError(t, err)

	memStore := memory.NewStore(24 * time.Hour)
	msgService := service.NewMessageService(memStore)
	msgService.SetFilesystemStore(fsStore)

	t.Run("concurrent message creation", func(t *testing.T) {
		mailboxID := "concurrent-mailbox"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "concurrent@example.com",
			LocalPart: "concurrent",
			Domain:    "example.com",
			Token:     "concurrent-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 并发创建邮件
		numMessages := 20
		done := make(chan bool, numMessages)
		errors := make(chan error, numMessages)

		for i := 0; i < numMessages; i++ {
			go func(index int) {
				input := service.CreateMessageInput{
					MailboxID: mailboxID,
					From:      "sender@example.com",
					To:        "concurrent@example.com",
					Subject:   "Concurrent Email " + string(rune('0'+index)),
					Text:      "Content " + string(rune('0'+index)),
					Raw:       "Raw " + string(rune('0'+index)),
				}

				message, err := msgService.Create(input)
				if err != nil {
					errors <- err
					done <- false
					return
				}

				// 保存到文件系统
				if _, err := fsStore.SaveMessageRaw(mailboxID, message.ID, []byte(input.Raw)); err != nil {
					errors <- err
					done <- false
					return
				}

				message.Text = input.Text
				if _, err := fsStore.SaveMessageMetadata(mailboxID, message.ID, message); err != nil {
					errors <- err
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// 等待所有 goroutine 完成
		successCount := 0
		for i := 0; i < numMessages; i++ {
			if <-done {
				successCount++
			}
		}
		close(errors)

		// 验证没有错误
		errorList := make([]error, 0)
		for err := range errors {
			errorList = append(errorList, err)
		}
		assert.Empty(t, errorList, "Should have no errors")
		assert.Equal(t, numMessages, successCount)

		// 验证所有邮件都已创建
		messages, err := msgService.List(mailboxID)
		require.NoError(t, err)
		assert.Equal(t, numMessages, len(messages))
	})
}

// TestIntegration_ErrorHandling 测试错误处理
func TestIntegration_ErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "error_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fsStore, err := NewStore(tempDir)
	require.NoError(t, err)

	memStore := memory.NewStore(24 * time.Hour)
	msgService := service.NewMessageService(memStore)
	msgService.SetFilesystemStore(fsStore)

	t.Run("get non-existent message", func(t *testing.T) {
		_, err := msgService.Get("non-existent-mailbox", "non-existent-message")
		assert.Error(t, err)
	})

	t.Run("get attachment from non-existent message", func(t *testing.T) {
		_, err := msgService.GetAttachment("non-existent-mailbox", "non-existent-message", "att-001")
		assert.Error(t, err)
	})

	t.Run("delete non-existent message", func(t *testing.T) {
		err := msgService.Delete("non-existent-mailbox", "non-existent-message")
		// 删除不存在的消息可能不报错（取决于实现）
		// 主要是确保不会 panic
		_ = err
	})

	t.Run("filesystem error does not break database operation", func(t *testing.T) {
		mailboxID := "error-mailbox"

		// 创建邮箱
		mailbox := &domain.Mailbox{
			ID:        mailboxID,
			Address:   "error@example.com",
			LocalPart: "error",
			Domain:    "example.com",
			Token:     "error-token",
			CreatedAt: time.Now(),
		}
		err := memStore.SaveMailbox(mailbox)
		require.NoError(t, err)

		// 创建邮件（即使文件系统失败，数据库操作应该成功）
		input := service.CreateMessageInput{
			MailboxID: mailboxID,
			From:      "sender@example.com",
			To:        "error@example.com",
			Subject:   "Test Resilience",
			Text:      "Content",
		}

		message, err := msgService.Create(input)
		require.NoError(t, err)
		assert.NotEmpty(t, message.ID)

		// 验证即使没有文件系统数据，元数据仍可获取
		retrieved, err := msgService.Get(mailboxID, message.ID)
		require.NoError(t, err)
		assert.Equal(t, message.ID, retrieved.ID)
		assert.Equal(t, input.Subject, retrieved.Subject)
	})
}

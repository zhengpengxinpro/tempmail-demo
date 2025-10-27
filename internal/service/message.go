package service

import (
	"time"

	"github.com/google/uuid"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/storage"
)

// FilesystemStore 文件系统存储接口
type FilesystemStore interface {
	SaveMessageRaw(mailboxID, messageID string, rawContent []byte) (string, error)
	SaveMessageMetadata(mailboxID, messageID string, message *domain.Message) (string, error)
	SaveAttachment(mailboxID, messageID, attachmentID string, attachment *domain.Attachment) (string, error)
	GetMessageRaw(mailboxID, messageID string) ([]byte, error)
	GetMessageMetadata(mailboxID, messageID string) (*domain.Message, error)
	GetAttachment(mailboxID, messageID, attachmentID string) (*domain.Attachment, error)
}

// MessageService 封装邮件处理逻辑。
type MessageService struct {
	repo    storage.MessageRepository
	fsStore FilesystemStore // 文件系统存储（可选）
}

// NewMessageService 创建邮件业务服务。
func NewMessageService(repo storage.MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}

// SetFilesystemStore 设置文件系统存储
func (s *MessageService) SetFilesystemStore(fsStore FilesystemStore) {
	s.fsStore = fsStore
}

// CreateMessageInput 定义创建邮件的输入。
type CreateMessageInput struct {
	MailboxID   string
	From        string
	To          string
	Subject     string
	Text        string
	HTML        string
	Raw         string
	IsRead      bool
	Received    time.Time
	Attachments []*domain.Attachment // 附件列表
}

// Create 新建一封邮件。
func (s *MessageService) Create(input CreateMessageInput) (*domain.Message, error) {
	now := time.Now().UTC()
	if input.Received.IsZero() {
		input.Received = now
	}

	message := &domain.Message{
		ID:         uuid.NewString(),
		MailboxID:  input.MailboxID,
		From:       input.From,
		To:         input.To,
		Subject:    input.Subject,
		IsRead:     input.IsRead,
		CreatedAt:  now,
		ReceivedAt: input.Received,
		// 设置文件系统标记
		HasRaw:  input.Raw != "",
		HasHTML: input.HTML != "",
		HasText: input.Text != "",
		// 内容字段不存数据库
		Text:        input.Text,
		HTML:        input.HTML,
		Raw:         input.Raw,
		Attachments: input.Attachments,
	}

	// 先保存元数据到数据库
	if err := s.repo.SaveMessage(message); err != nil {
		return nil, err
	}

	if s.fsStore != nil {
		if err := s.persistToFilesystem(message, input); err != nil {
			return nil, err
		}
	}

	return message, nil
}

// List 列出指定邮箱下的邮件。
func (s *MessageService) List(mailboxID string) ([]domain.Message, error) {
	return s.repo.ListMessages(mailboxID)
}

// Get 获取单封邮件详情。
func (s *MessageService) Get(mailboxID, messageID string) (*domain.Message, error) {
	// 从数据库获取元数据
	message, err := s.repo.GetMessage(mailboxID, messageID)
	if err != nil {
		return nil, err
	}

	// 如果配置了文件系统存储，从文件加载内容
	if s.fsStore != nil {
		// 加载原始邮件内容
		if message.HasRaw {
			if rawBytes, err := s.fsStore.GetMessageRaw(mailboxID, messageID); err == nil {
				message.Raw = string(rawBytes)
			}
		}

		// 从元数据文件加载 Text 和 HTML（如果需要）
		if metadata, err := s.fsStore.GetMessageMetadata(mailboxID, messageID); err == nil {
			if message.HasText {
				message.Text = metadata.Text
			}
			if message.HasHTML {
				message.HTML = metadata.HTML
			}
		}

		// 加载附件内容
		for i, att := range message.Attachments {
			if fullAtt, err := s.fsStore.GetAttachment(mailboxID, messageID, att.ID); err == nil {
				message.Attachments[i] = fullAtt
			}
		}
	}

	return message, nil
}

// MarkRead 将邮件标记为已读。
func (s *MessageService) MarkRead(mailboxID, messageID string) error {
	return s.repo.MarkMessageRead(mailboxID, messageID)
}

// GetAttachment 获取邮件附件。
func (s *MessageService) GetAttachment(mailboxID, messageID, attachmentID string) (*domain.Attachment, error) {
	// 先验证邮件是否存在
	message, err := s.repo.GetMessage(mailboxID, messageID)
	if err != nil {
		return nil, err
	}

	// 如果配置了文件系统存储，从文件加载附件
	if s.fsStore != nil {
		return s.fsStore.GetAttachment(mailboxID, messageID, attachmentID)
	}

	// 否则从数据库查找附件（旧方式，向后兼容）
	for _, att := range message.Attachments {
		if att.ID == attachmentID {
			return att, nil
		}
	}

	return nil, storage.ErrAttachmentNotFound
}

// Delete 删除指定邮件。
func (s *MessageService) Delete(mailboxID, messageID string) error {
	return s.repo.DeleteMessage(mailboxID, messageID)
}

// ClearAll 清空邮箱中的所有邮件，返回删除数量。
func (s *MessageService) ClearAll(mailboxID string) (int, error) {
	return s.repo.DeleteAllMessages(mailboxID)
}

func (s *MessageService) persistToFilesystem(message *domain.Message, input CreateMessageInput) error {
	mailboxID := input.MailboxID
	messageID := message.ID

	if input.Raw != "" {
		if _, err := s.fsStore.SaveMessageRaw(mailboxID, messageID, []byte(input.Raw)); err != nil {
			return err
		}
	}

	attachments := make([]*domain.Attachment, 0, len(input.Attachments))
	for _, att := range input.Attachments {
		if att == nil {
			continue
		}

		if att.ID == "" {
			att.ID = uuid.NewString()
		}
		if att.Size == 0 {
			att.Size = int64(len(att.Content))
		}

		path, err := s.fsStore.SaveAttachment(mailboxID, messageID, att.ID, att)
		if err != nil {
			return err
		}

		// 存储时不保留内存中的附件内容
		attachments = append(attachments, &domain.Attachment{
			ID:          att.ID,
			MessageID:   messageID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			StoragePath: path,
		})
	}

	message.Attachments = attachments

	// 保存元数据（包含文本与附件信息）
	if _, err := s.fsStore.SaveMessageMetadata(mailboxID, messageID, message); err != nil {
		return err
	}

	return nil
}

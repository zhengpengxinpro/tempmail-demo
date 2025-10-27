package smtp

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"

	gosmtp "github.com/emersion/go-smtp"

	"tempmail/backend/internal/domain"
	"tempmail/backend/internal/service"
	"tempmail/backend/internal/websocket"
)

var (
	// ErrMailboxNotFound 表示收件人不存在。
	ErrMailboxNotFound = errors.New("mailbox not found")
)

// Backend 实现 go-smtp 的 Backend 接口。
//
// 【安全说明】
// 这是一个只接收邮件的 SMTP 服务器（Receiving-Only SMTP Server）。
// 特性：
// - ✅ 只接收发送到本系统邮箱/别名的邮件
// - ✅ 严格验证收件人地址必须存在于系统中
// - ✅ 动态加载系统域名配置（支持后台添加域名）
// - ❌ 不支持对外发送邮件（无邮件中继功能）
// - ❌ 不会成为垃圾邮件中继或开放中继
//
// 安全机制：
// 1. Rcpt() 方法严格验证收件人地址
// 2. 只有系统内的邮箱/别名才能接收邮件
// 3. 检查收件人域名是否在激活的系统域名列表中
// 4. 外部地址一律返回 550 错误拒绝
type Backend struct {
	mailboxes         *service.MailboxService
	messages          *service.MessageService
	aliases           *service.AliasService
	systemDomains     *service.SystemDomainService
	userDomainService *service.UserDomainService
	wsHub             *websocket.Hub
	fsStore           FilesystemStore // 文件系统存储接口
}

// FilesystemStore 文件系统存储接口
type FilesystemStore interface {
	SaveMessageRaw(mailboxID, messageID string, rawContent []byte) (string, error)
	SaveMessageMetadata(mailboxID, messageID string, message *domain.Message) (string, error)
	SaveAttachment(mailboxID, messageID, attachmentID string, attachment *domain.Attachment) (string, error)
	GetMessageRaw(mailboxID, messageID string) ([]byte, error)
	GetMessageMetadata(mailboxID, messageID string) (*domain.Message, error)
}

// NewBackend 创建 SMTP Backend。
func NewBackend(
	mailboxes *service.MailboxService,
	messages *service.MessageService,
	aliases *service.AliasService,
	systemDomains *service.SystemDomainService,
	userDomainService *service.UserDomainService,
	wsHub *websocket.Hub,
	fsStore FilesystemStore,
) *Backend {
	return &Backend{
		mailboxes:         mailboxes,
		messages:          messages,
		aliases:           aliases,
		systemDomains:     systemDomains,
		userDomainService: userDomainService,
		wsHub:             wsHub,
		fsStore:           fsStore,
	}
}

// NewSession 创建新的 SMTP 会话。
func (b *Backend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	return &session{
		backend: b,
	}, nil
}

type session struct {
	backend     *Backend
	fromAddress string
	recipients  []recipient
}

type recipient struct {
	address string
	id      string
}

// Mail 处理 MAIL 命令。
func (s *session) Mail(from string, opts *gosmtp.MailOptions) error {
	s.fromAddress = from
	return nil
}

// Rcpt 处理 RCPT 命令。
//
// 【安全关键】此方法是防止邮件中继的核心。
// 只接受发送到本系统邮箱或别名的邮件，拒绝所有外部地址。
// 这确保了服务器不会被用作垃圾邮件中继。
//
// 验证流程：
// 1. 提取收件人域名
// 2. 检查域名是否在激活的系统域名列表或用户域名列表中
// 3. 查找对应的邮箱或别名
// 4. 如果都不存在，返回 550 错误
func (s *session) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	addr := normalizeAddress(to)

	// 提取域名部分
	parts := strings.Split(addr, "@")
	if len(parts) != 2 {
		return &gosmtp.SMTPError{
			Code:         501,
			EnhancedCode: gosmtp.EnhancedCode{5, 1, 3},
			Message:      "invalid recipient address",
		}
	}
	recipientDomain := parts[1]

	// 验证域名是否被管理
	domainAllowed := false

	// 检查系统域名
	if s.backend.systemDomains != nil {
		activeDomains, err := s.backend.systemDomains.GetAllActiveDomains()
		if err == nil {
			for _, d := range activeDomains {
				if strings.EqualFold(d, recipientDomain) {
					domainAllowed = true
					break
				}
			}
		}
	}

	// 如果不在系统域名中，检查用户域名
	if !domainAllowed && s.backend.userDomainService != nil {
		userDomains, err := s.backend.userDomainService.ListUserDomains("")
		if err == nil {
			for _, ud := range userDomains {
				if ud.IsActive && ud.Status == domain.DomainStatusVerified {
					if strings.EqualFold(ud.Domain, recipientDomain) {
						domainAllowed = true
						break
					}
				}
			}
		}
	}

	// 域名不在管理列表中，拒绝接收
	if !domainAllowed {
		return &gosmtp.SMTPError{
			Code:         550,
			EnhancedCode: gosmtp.EnhancedCode{5, 7, 1},
			Message:      "relay access denied - domain not managed by this server",
		}
	}

	// 首先尝试查找主邮箱
	mb, err := s.backend.mailboxes.GetByAddress(addr)
	if err == nil {
		// 找到主邮箱
		s.recipients = append(s.recipients, recipient{
			address: addr,
			id:      mb.ID,
		})
		return nil
	}

	// 如果没有找到主邮箱，尝试查找别名
	if s.backend.aliases != nil {
		alias, err := s.backend.aliases.GetByAddress(addr)
		if err == nil && alias.IsActive {
			// 找到激活的别名，将邮件路由到主邮箱
			s.recipients = append(s.recipients, recipient{
				address: addr,            // 保留原始收件地址
				id:      alias.MailboxID, // 使用别名关联的主邮箱ID
			})
			return nil
		}
	}

	// 域名是管理的，但邮箱不存在
	// 返回 550 错误，拒绝接收发往不存在邮箱的邮件
	return &gosmtp.SMTPError{
		Code:         550,
		EnhancedCode: gosmtp.EnhancedCode{5, 1, 1},
		Message:      "recipient mailbox not found",
	}
}

// Data 处理邮件内容。
func (s *session) Data(r io.Reader) error {
	rawBytes, err := io.ReadAll(io.LimitReader(r, 10<<20)) // 10MB
	if err != nil {
		return err
	}

	// 使用新的 MIME 解析器
	parsed, err := ParseEmail(rawBytes)
	if err != nil {
		return fmt.Errorf("parse email: %w", err)
	}

	// 为每个收件人创建邮件
	for _, rcpt := range s.recipients {
		// 1️⃣ 创建邮件元数据（不包含 Raw、Text、HTML - 这些存文件）
		messageInput := service.CreateMessageInput{
			MailboxID: rcpt.id,
			From:      s.fromAddress,
			To:        rcpt.address,
			Subject:   parsed.Subject,
			Text:      parsed.Text,
			HTML:      parsed.HTML,
			Raw:       string(rawBytes),
			IsRead:    false,
		}

		for _, att := range parsed.Attachments {
			messageInput.Attachments = append(messageInput.Attachments, &domain.Attachment{
				ID:          att.ID,
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Size:        att.Size,
				Content:     att.Content,
			})
		}

		message, err := s.backend.messages.Create(messageInput)
		if err != nil {
			return err
		}

		// 4️⃣ WebSocket 通知（使用元数据）
		if s.backend.wsHub != nil {
			s.backend.wsHub.NotifyNewMail(rcpt.id, message)
		}
	}

	return nil
}

// AuthPlain 处理 PLAIN 认证（此处允许匿名）。
func (s *session) AuthPlain(username, password string) error {
	return nil
}

// Reset 重置状态。
func (s *session) Reset() {
	s.fromAddress = ""
	s.recipients = nil
}

// Logout 会话结束。
func (s *session) Logout() error {
	return nil
}

func normalizeAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.Trim(addr, "<>")
	return strings.ToLower(addr)
}

func decodeHeader(value string) string {
	if value == "" {
		return value
	}
	decoder := new(mime.WordDecoder)
	decoded, err := decoder.DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}

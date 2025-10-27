package websocket

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"tempmail/backend/internal/domain"
)

// MailboxStore 邮箱存储接口
type MailboxStore interface {
	GetMailbox(id string) (*domain.Mailbox, error)
	ListMailboxesByUserID(userID string) []domain.Mailbox
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Tier   string `json:"tier"`
	jwt.RegisteredClaims
}

// upgraderFactory 创建带有 Origin 验证的 WebSocket 升级器
func upgraderFactory(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// 如果允许所有来源
			for _, origin := range allowedOrigins {
				if origin == "*" {
					return true
				}
			}

			// 获取请求的 Origin
			requestOrigin := r.Header.Get("Origin")
			if requestOrigin == "" {
				// 如果没有 Origin，检查是否是同源请求
				return true
			}

			// 检查 Origin 是否在允许列表中
			for _, origin := range allowedOrigins {
				if requestOrigin == origin {
					return true
				}
			}

			return false
		},
	}
}

// MessageType 定义WebSocket消息类型
type MessageType string

const (
	MessageTypeNewMail       MessageType = "new_mail"
	MessageTypeMailboxUpdate MessageType = "mailbox_update"
	MessageTypePing          MessageType = "ping"
	MessageTypePong          MessageType = "pong"
	MessageTypeSubscribe     MessageType = "subscribe"
	MessageTypeUnsubscribe   MessageType = "unsubscribe"
	MessageTypeSubscribed    MessageType = "subscribed"
	MessageTypeError         MessageType = "error"
)

// Message 定义WebSocket消息结构
type Message struct {
	Type      MessageType     `json:"type"`
	MailboxID string          `json:"mailboxId,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// Client 代表一个WebSocket客户端连接
type Client struct {
	ID         string
	conn       *websocket.Conn
	send       chan []byte
	hub        *Hub
	mailboxIDs map[string]bool // 订阅的邮箱ID
	mu         sync.RWMutex
	log        *zap.Logger
	// 认证信息
	UserID      string   // 用户ID（JWT认证）
	MailboxID   string   // 邮箱ID（Mailbox Token认证）
	Token       string   // 原始token
	IsMailbox   bool     // 是否是邮箱token认证
	Permissions []string // 可访问的邮箱ID列表
}

// Hub 管理所有WebSocket连接
type Hub struct {
	clients        map[string]*Client            // clientID -> Client
	mailboxes      map[string]map[string]*Client // mailboxID -> clientID -> Client
	register       chan *Client
	unregister     chan *Client
	broadcast      chan *BroadcastMessage
	mu             sync.RWMutex
	log            *zap.Logger
	allowedOrigins []string // 允许的 Origin 列表
	// 认证相关
	jwtSecret    string       // JWT密钥
	mailboxStore MailboxStore // 邮箱存储接口
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	MailboxID string
	Message   *Message
}

// NewHub 创建WebSocket Hub
//
// 参数:
//   - allowedOrigins: 允许的 Origin 列表，用于 WebSocket 连接验证
//   - jwtSecret: JWT密钥，用于验证用户token
//   - mailboxStore: 邮箱存储接口，用于验证邮箱权限
//
// 返回值:
//   - *Hub: 创建的 Hub 实例
func NewHub(allowedOrigins []string, jwtSecret string, mailboxStore MailboxStore) *Hub {
	// 如果没有配置，默认允许所有
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return &Hub{
		clients:        make(map[string]*Client),
		mailboxes:      make(map[string]map[string]*Client),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan *BroadcastMessage, 256),
		log:            zap.NewNop(), // 临时使用空日志
		allowedOrigins: allowedOrigins,
		jwtSecret:      jwtSecret,
		mailboxStore:   mailboxStore,
	}
}

// Run 启动Hub
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.log.Info("websocket hub stopped")
			h.closeAllClients()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			h.log.Info("client registered", zap.String("id", client.ID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				// 从所有邮箱订阅中移除
				for mailboxID := range client.mailboxIDs {
					if clients, exists := h.mailboxes[mailboxID]; exists {
						delete(clients, client.ID)
						if len(clients) == 0 {
							delete(h.mailboxes, mailboxID)
						}
					}
				}
				delete(h.clients, client.ID)
				close(client.send)
				h.log.Info("client unregistered", zap.String("id", client.ID))
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.broadcastToMailbox(msg.MailboxID, msg.Message)

		case <-ticker.C:
			// 定期ping所有客户端
			h.pingAllClients()
		}
	}
}

// NewMailData 新邮件通知数据
type NewMailData struct {
	MessageID string `json:"messageId"`
	MailboxID string `json:"mailboxId"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Preview   string `json:"preview,omitempty"`
	HasHTML   bool   `json:"hasHtml"`
	HasText   bool   `json:"hasText"`
	CreatedAt string `json:"createdAt"`
}

// NotifyNewMail 通知新邮件
func (h *Hub) NotifyNewMail(mailboxID string, message *domain.Message) {
	// 构建前端期望的数据格式
	preview := ""
	if message.Text != "" && len(message.Text) > 100 {
		preview = message.Text[:100]
	} else if message.Text != "" {
		preview = message.Text
	}

	newMailData := NewMailData{
		MessageID: message.ID,
		MailboxID: mailboxID,
		From:      message.From,
		To:        message.To,
		Subject:   message.Subject,
		Preview:   preview,
		HasHTML:   message.HTML != "",
		HasText:   message.Text != "",
		CreatedAt: message.CreatedAt.Format(time.RFC3339),
	}

	data, err := json.Marshal(newMailData)
	if err != nil {
		h.log.Error("failed to marshal new mail data", zap.Error(err))
		return
	}

	msg := &Message{
		Type:      MessageTypeNewMail,
		MailboxID: mailboxID,
		Data:      data,
		Timestamp: time.Now(),
	}

	h.log.Info("broadcasting new mail notification",
		zap.String("mailboxID", mailboxID),
		zap.String("from", message.From),
		zap.String("subject", message.Subject))

	h.broadcast <- &BroadcastMessage{
		MailboxID: mailboxID,
		Message:   msg,
	}
}

// MailboxUpdateData 邮箱更新通知数据
type MailboxUpdateData struct {
	MailboxID    string `json:"mailboxId"`
	UnreadCount  int    `json:"unreadCount"`
	TotalCount   int    `json:"totalCount"`
	LastActivity string `json:"lastActivity"`
}

// NotifyMailboxUpdate 通知邮箱更新
func (h *Hub) NotifyMailboxUpdate(mailbox *domain.Mailbox) {
	// 构建前端期望的数据格式
	updateData := MailboxUpdateData{
		MailboxID:    mailbox.ID,
		UnreadCount:  mailbox.Unread,
		TotalCount:   mailbox.TotalCount,
		LastActivity: time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(updateData)
	if err != nil {
		h.log.Error("failed to marshal mailbox update data", zap.Error(err))
		return
	}

	msg := &Message{
		Type:      MessageTypeMailboxUpdate,
		MailboxID: mailbox.ID,
		Data:      data,
		Timestamp: time.Now(),
	}

	h.log.Info("broadcasting mailbox update",
		zap.String("mailboxID", mailbox.ID),
		zap.Int("unread", mailbox.Unread),
		zap.Int("total", mailbox.TotalCount))

	h.broadcast <- &BroadcastMessage{
		MailboxID: mailbox.ID,
		Message:   msg,
	}
}

// broadcastToMailbox 向订阅特定邮箱的客户端广播消息
func (h *Hub) broadcastToMailbox(mailboxID string, msg *Message) {
	h.mu.RLock()
	clients := h.mailboxes[mailboxID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		h.log.Error("failed to marshal message", zap.Error(err))
		return
	}

	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			// 客户端阻塞，跳过
			h.log.Warn("client channel blocked, skipping", zap.String("clientID", client.ID))
		}
	}
}

// pingAllClients 向所有客户端发送ping
func (h *Hub) pingAllClients() {
	msg := &Message{
		Type:      MessageTypePing,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
			// 跳过阻塞的客户端
		}
	}
}

// closeAllClients 关闭所有客户端连接
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		close(client.send)
	}
	h.clients = make(map[string]*Client)
	h.mailboxes = make(map[string]map[string]*Client)
}

// authenticateClient 认证客户端
func (h *Hub) authenticateClient(c *gin.Context) (*Client, error) {
	// 从URL参数或Header获取token
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}
	}

	if token == "" {
		return nil, errors.New("missing authentication token")
	}

	// 尝试JWT认证
	if userID, email, err := h.validateJWT(token); err == nil {
		// JWT认证成功，获取用户的所有邮箱
		mailboxes := h.mailboxStore.ListMailboxesByUserID(userID)

		permissions := make([]string, len(mailboxes))
		for i, mb := range mailboxes {
			permissions[i] = mb.ID
		}

		client := &Client{
			ID:          generateClientID(),
			UserID:      userID,
			Token:       token,
			IsMailbox:   false,
			Permissions: permissions,
			mailboxIDs:  make(map[string]bool),
			log:         h.log,
		}

		h.log.Info("JWT authentication successful",
			zap.String("userID", userID),
			zap.String("email", email),
			zap.Int("mailboxCount", len(permissions)))

		return client, nil
	}

    // 尝试Mailbox Token认证（需要提供 mailboxId 并且 token 与之匹配）
    if mailboxID, err := h.validateMailboxToken(token, c.Query("mailboxId")); err == nil {
        client := &Client{
            ID:          generateClientID(),
            MailboxID:   mailboxID,
            Token:       token,
            IsMailbox:   true,
            Permissions: []string{mailboxID},
            mailboxIDs:  make(map[string]bool),
            log:         h.log,
        }

        h.log.Info("Mailbox token authentication successful",
            zap.String("mailboxID", mailboxID))

        return client, nil
    }

	return nil, errors.New("invalid authentication token")
}

// validateJWT 验证JWT token
func (h *Hub) validateJWT(tokenString string) (userID, email string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.jwtSecret), nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims.UserID, claims.Email, nil
	}

	return "", "", errors.New("invalid token claims")
}

// validateMailboxToken 验证邮箱token
func (h *Hub) validateMailboxToken(token, mailboxID string) (string, error) {
    if mailboxID == "" {
        return "", errors.New("invalid mailbox token")
    }

    mailbox, err := h.mailboxStore.GetMailbox(mailboxID)
    if err != nil || mailbox == nil || mailbox.Token == "" {
        return "", errors.New("invalid mailbox token")
    }

    if subtle.ConstantTimeCompare([]byte(mailbox.Token), []byte(token)) != 1 {
        return "", errors.New("invalid mailbox token")
    }

    return mailbox.ID, nil
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(hub *Hub) gin.HandlerFunc {
	// 使用 Hub 配置的允许 Origin 创建 upgrader
	upgrader := upgraderFactory(hub.allowedOrigins)

	return func(c *gin.Context) {
		// 认证客户端
		client, err := hub.authenticateClient(c)
		if err != nil {
			hub.log.Warn("websocket authentication failed",
				zap.Error(err),
				zap.String("remote_addr", c.ClientIP()))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// 升级连接
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			hub.log.Error("failed to upgrade connection",
				zap.Error(err),
				zap.String("origin", c.Request.Header.Get("Origin")),
				zap.String("remote_addr", c.ClientIP()))
			return
		}

		// 设置连接和Hub
		client.conn = conn
		client.hub = hub
		client.send = make(chan []byte, 256)

		// 注册客户端
		hub.register <- client

		// 启动读写协程
		go client.writePump()
		go client.readPump()
	}
}

// readPump 处理客户端消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.log.Error("websocket error", zap.Error(err))
			}
			break
		}

		// 处理消息
		c.handleMessage(&msg)
	}
}

// writePump 发送消息给客户端
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypeSubscribe:
		c.subscribeMailbox(msg.MailboxID)
	case MessageTypeUnsubscribe:
		c.unsubscribeMailbox(msg.MailboxID)
	case MessageTypePong:
		// 客户端响应pong，更新活动时间
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	default:
		c.log.Warn("unknown message type", zap.String("type", string(msg.Type)))
	}
}

// subscribeMailbox 订阅邮箱
func (c *Client) subscribeMailbox(mailboxID string) {
	if mailboxID == "" {
		c.sendError("mailbox ID is required")
		return
	}

	// 验证权限
	hasPermission := false
	for _, permMailboxID := range c.Permissions {
		if permMailboxID == mailboxID {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		c.log.Warn("subscription denied: no permission",
			zap.String("clientID", c.ID),
			zap.String("mailboxID", mailboxID),
			zap.Bool("isMailbox", c.IsMailbox))
		c.sendError(fmt.Sprintf("no permission to access mailbox: %s", mailboxID))
		return
	}

	c.mu.Lock()
	c.mailboxIDs[mailboxID] = true
	c.mu.Unlock()

	c.hub.mu.Lock()
	if c.hub.mailboxes[mailboxID] == nil {
		c.hub.mailboxes[mailboxID] = make(map[string]*Client)
	}
	c.hub.mailboxes[mailboxID][c.ID] = c
	c.hub.mu.Unlock()

	c.log.Info("subscribed to mailbox",
		zap.String("clientID", c.ID),
		zap.String("mailboxID", mailboxID),
		zap.String("userID", c.UserID))

	// 发送订阅成功确认
	c.sendMessage(&Message{
		Type:      "subscribed",
		MailboxID: mailboxID,
		Timestamp: time.Now(),
	})
}

// sendError 发送错误消息给客户端
func (c *Client) sendError(errMsg string) {
	msg := &Message{
		Type:      MessageTypeError,
		Error:     errMsg,
		Timestamp: time.Now(),
	}
	c.sendMessage(msg)
}

// sendMessage 发送消息给客户端
func (c *Client) sendMessage(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		c.log.Error("failed to marshal message", zap.Error(err))
		return
	}

	select {
	case c.send <- data:
	default:
		c.log.Warn("client channel blocked", zap.String("clientID", c.ID))
	}
}

// unsubscribeMailbox 取消订阅邮箱
func (c *Client) unsubscribeMailbox(mailboxID string) {
	c.mu.Lock()
	delete(c.mailboxIDs, mailboxID)
	c.mu.Unlock()

	c.hub.mu.Lock()
	if clients, exists := c.hub.mailboxes[mailboxID]; exists {
		delete(clients, c.ID)
		if len(clients) == 0 {
			delete(c.hub.mailboxes, mailboxID)
		}
	}
	c.hub.mu.Unlock()

	c.log.Info("unsubscribed from mailbox",
		zap.String("clientID", c.ID),
		zap.String("mailboxID", mailboxID))
}

// generateClientID 生成客户端ID
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

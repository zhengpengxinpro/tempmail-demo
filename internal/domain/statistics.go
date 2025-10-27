package domain

import "time"

// SystemStatistics 系统统计信息
type SystemStatistics struct {
	TotalUsers        int              `json:"totalUsers"`
	ActiveUsers       int              `json:"activeUsers"`
	TotalMailboxes    int              `json:"totalMailboxes"`
	ActiveMailboxes   int              `json:"activeMailboxes"`
	TotalMessages     int              `json:"totalMessages"`
	MessagesToday     int              `json:"messagesToday"`
	UsersByTier       map[UserTier]int `json:"usersByTier"`
	UsersByRole       map[UserRole]int `json:"usersByRole"`
	MailboxesByDomain map[string]int   `json:"mailboxesByDomain"`
	RecentActivity    []ActivityLog    `json:"recentActivity"`
}

// ActivityLog 活动日志
type ActivityLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	UserEmail string    `json:"userEmail"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"createdAt"`
}

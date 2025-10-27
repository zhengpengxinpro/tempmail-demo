package domain

// Attachment 表示邮件附件。
type Attachment struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`            // 附件唯一标识
	MessageID   string `json:"messageId" gorm:"type:varchar(36);index;not null"` // 所属邮件ID
	Filename    string `json:"filename" gorm:"type:varchar(255)"`                // 文件名
	ContentType string `json:"contentType" gorm:"type:varchar(100)"`             // MIME类型
	Size        int64  `json:"size"`                                             // 大小（字节）
	StoragePath string `json:"storagePath,omitempty" gorm:"type:varchar(500)"`   // 文件存储路径（相对路径）
	Content     []byte `json:"-" gorm:"-"`                                       // 附件内容（不存数据库，从文件系统加载）
}

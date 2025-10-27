# 文件系统存储模块

## 📦 模块说明

该模块实现了临时邮件系统的文件系统存储功能，用于将邮件内容、附件等大数据从数据库迁移到文件系统，提高系统性能和可扩展性。

## 📁 文件结构

```
filesystem/
├── store.go           # 核心实现（401 行）
├── store_test.go      # 单元测试（687 行，32 个测试用例）
└── README.md          # 本文档
```

## 🚀 快速开始

### 创建存储实例

```go
import "tempmail/backend/internal/storage/filesystem"

// 创建文件系统存储
store, err := filesystem.NewStore("/path/to/storage")
if err != nil {
    log.Fatal(err)
}
```

### 保存邮件

```go
// 保存原始邮件
err := store.SaveMessageRaw(mailboxID, messageID, rawContent)

// 保存元数据
err := store.SaveMessageMetadata(mailboxID, messageID, message)

// 保存附件
for _, att := range message.Attachments {
    err := store.SaveAttachment(mailboxID, messageID, att.ID, att)
}
```

### 读取邮件

```go
// 读取原始邮件
rawContent, err := store.GetMessageRaw(mailboxID, messageID)

// 读取元数据
metadata, err := store.GetMessageMetadata(mailboxID, messageID)

// 读取附件
attachment, err := store.GetAttachment(mailboxID, messageID, attachmentID)
```

### 删除操作

```go
// 删除单个邮件
err := store.DeleteMessage(mailboxID, messageID)

// 删除整个邮箱
err := store.DeleteMailbox(mailboxID)

// 清理过期邮件（7 天前）
count, err := store.CleanupExpired(7)
```

### 统计信息

```go
// 获取存储统计
stats, err := store.GetStorageStats()
fmt.Printf("总大小: %d MB\n", stats["total_size_mb"])
fmt.Printf("邮件数: %d\n", stats["message_count"])
fmt.Printf("附件数: %d\n", stats["attachment_count"])
```

## 📂 存储结构

```
/path/to/storage/
└── mails/
    └── {mailbox_id}/
        └── {YYYY-MM-DD}/
            └── {message_id}/
                ├── raw.eml                      # 原始邮件
                ├── metadata.json                # 元数据
                └── attachments/                 # 附件目录
                    ├── {att_id}_filename.ext    # 附件文件
                    └── {att_id}_filename.ext.meta.json
```

## 🧪 运行测试

### 运行所有测试
```bash
go test -v ./internal/storage/filesystem/...
```

### 运行特定测试
```bash
go test -v ./internal/storage/filesystem/... -run TestSaveMessageRaw
```

### 生成覆盖率报告
```bash
go test ./internal/storage/filesystem/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### 测试结果
- ✅ **32 个测试用例** 全部通过
- ✅ **83.5% 代码覆盖率**
- ✅ **0 个失败** / 0 个错误

## 🔍 测试覆盖范围

### 功能测试
- ✅ Store 实例创建
- ✅ 邮件原始内容保存/读取
- ✅ 邮件元数据保存/读取
- ✅ 附件保存/读取
- ✅ 邮件删除
- ✅ 邮箱删除
- ✅ 过期邮件清理

### 边界测试
- ✅ 空内容处理
- ✅ 超长文本（10KB 主题）
- ✅ Unicode 字符（中文、日文、emoji）
- ✅ 路径遍历攻击防护
- ✅ 并发操作安全性

### 错误处理
- ✅ 文件不存在
- ✅ 目录不存在
- ✅ 无效参数

## 📊 性能特点

- **文件组织**: 按日期分层存储，便于清理和归档
- **安全性**: 防止路径遍历攻击，使用安全文件名
- **并发安全**: 支持多 goroutine 并发读写
- **错误处理**: 完善的错误处理和日志记录

## 🔗 相关文档

- [技术设计文档](../../FILESYSTEM_STORAGE.md)
- [实施指南](../../FILESYSTEM_STORAGE_GUIDE.md)
- [测试文档](../../FILESYSTEM_STORAGE_TESTS.md)
- [实施总结](../../FILESYSTEM_IMPLEMENTATION_SUMMARY.md)

## 📝 API 文档

### NewStore
```go
func NewStore(basePath string) (*Store, error)
```
创建文件系统存储实例，如果基础目录不存在则自动创建。

### SaveMessageRaw
```go
func (s *Store) SaveMessageRaw(mailboxID, messageID string, rawContent []byte) error
```
保存邮件原始内容到 `raw.eml` 文件。

### GetMessageRaw
```go
func (s *Store) GetMessageRaw(mailboxID, messageID string) ([]byte, error)
```
读取邮件原始内容。

### SaveMessageMetadata
```go
func (s *Store) SaveMessageMetadata(mailboxID, messageID string, message *domain.Message) error
```
保存邮件元数据到 `metadata.json` 文件。

### GetMessageMetadata
```go
func (s *Store) GetMessageMetadata(mailboxID, messageID string) (*domain.Message, error)
```
读取邮件元数据。

### SaveAttachment
```go
func (s *Store) SaveAttachment(mailboxID, messageID, attachmentID string, attachment *domain.Attachment) error
```
保存邮件附件。

### GetAttachment
```go
func (s *Store) GetAttachment(mailboxID, messageID, attachmentID string) (*domain.Attachment, error)
```
读取邮件附件。

### DeleteMessage
```go
func (s *Store) DeleteMessage(mailboxID, messageID string) error
```
删除邮件及其所有文件。

### DeleteMailbox
```go
func (s *Store) DeleteMailbox(mailboxID string) error
```
删除邮箱的所有邮件。

### CleanupExpired
```go
func (s *Store) CleanupExpired(retentionDays int) (int, error)
```
清理指定天数之前的邮件，返回删除数量。

### GetStorageStats
```go
func (s *Store) GetStorageStats() (map[string]interface{}, error)
```
获取存储统计信息（总大小、邮件数、附件数等）。

## ⚠️ 注意事项

1. **权限**: 确保进程有读写存储目录的权限
2. **磁盘空间**: 定期监控磁盘使用情况
3. **备份**: 建议定期备份存储目录
4. **清理**: 使用 `CleanupExpired()` 定期清理过期邮件
5. **并发**: 模块支持并发操作，但建议控制并发数

## 🐛 问题排查

### 文件不存在错误
```
message raw content not found
```
**原因**: 邮件文件已被删除或未创建  
**解决**: 检查邮件是否存在，重新保存邮件

### 权限错误
```
failed to create message directory: permission denied
```
**原因**: 进程没有写入权限  
**解决**: 检查目录权限，确保进程有写入权限

### 磁盘空间不足
```
failed to write raw message: no space left on device
```
**原因**: 磁盘空间已满  
**解决**: 清理磁盘空间或扩容

## 📄 许可证

本项目遵循 MIT 许可证。

## 👥 贡献者

- Claude (Anthropic) - 初始实现和测试

---

**版本**: v1.0  
**更新日期**: 2025-10-18





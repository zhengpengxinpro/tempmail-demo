package httptransport

import (
	"tempmail/backend/internal/service"
	"tempmail/backend/internal/storage/memory"
)

// 错误消息映射表（业务错误 -> 中文消息）
var errorMessages = map[error]string{
	// Mailbox 错误
	service.ErrDomainNotAllowed: "域名不在允许列表中",
	service.ErrPrefixInvalid:    "邮箱前缀格式无效",
	memory.ErrMailboxNotFound:   "邮箱不存在",

	// Message 错误
	memory.ErrMessageNotFound: "邮件不存在",

	// User Domain 错误
	service.ErrInvalidDomain:       "域名格式无效",
	service.ErrDomainAlreadyExists: "域名已存在",
	service.ErrDomainNotFound:      "域名不存在",
	service.ErrNotDomainOwner:      "您不是该域名的所有者",
	service.ErrDomainVerifyFailed:  "域名验证失败，请检查DNS记录",

	// Admin 错误
	service.ErrAdminUserNotFound:      "用户不存在",
	service.ErrCannotModifySelf:       "不能修改自己的账户",
	service.ErrCannotModifySuper:      "不能修改超级管理员账户",
	service.ErrInsufficientPermission: "权限不足",

	// API Key 错误
	service.ErrAPIKeyNotFound: "API Key不存在",
	service.ErrAPIKeyInvalid:  "API Key无效",
}

// GetErrorMessage 获取错误的中文消息
func GetErrorMessage(err error) string {
	if msg, ok := errorMessages[err]; ok {
		return msg
	}
	return err.Error()
}

// 通用错误消息
const (
	// 请求相关
	MsgInvalidRequest   = "请求参数格式错误"
	MsgInvalidJSON      = "JSON格式错误"
	MsgInvalidExpiresIn = "过期时间格式无效"
	MsgInvalidDuration  = "时长格式无效"
	MsgRequestBodyEmpty = "请求体不能为空"

	// 认证相关
	MsgAuthRequired       = "需要登录认证"
	MsgInvalidCredentials = "用户名或密码错误"
	MsgTokenExpired       = "登录已过期，请重新登录"
	MsgTokenInvalid       = "无效的访问令牌"
	MsgPermissionDenied   = "权限不足"

	// 邮箱相关
	MsgMailboxCreateFailed = "创建邮箱失败"
	MsgMailboxNotFound     = "邮箱不存在"
	MsgMailboxDeleteFailed = "删除邮箱失败"

	// 邮件相关
	MsgMessageCreateFailed   = "保存邮件失败"
	MsgMessageNotFound       = "邮件不存在"
	MsgMessageListFailed     = "获取邮件列表失败"
	MsgMessageMarkReadFailed = "标记已读失败"
	MsgMessageGetFailed      = "获取邮件详情失败"

	// 附件相关
	MsgAttachmentNotFound = "附件不存在"

	// 别名相关
	MsgAliasCreateFailed = "创建别名失败"
	MsgAliasNotFound     = "别名不存在"
	MsgAliasListFailed   = "获取别名列表失败"
	MsgAliasDeleteFailed = "删除别名失败"
	MsgAliasToggleFailed = "切换别名状态失败"

	// 用户域名相关
	MsgDomainAddFailed          = "添加域名失败"
	MsgDomainListFailed         = "获取域名列表失败"
	MsgDomainGetFailed          = "获取域名详情失败"
	MsgDomainVerifyFailed       = "验证域名失败"
	MsgDomainUpdateFailed       = "更新域名失败"
	MsgDomainDeleteFailed       = "删除域名失败"
	MsgDomainHasMailboxes       = "无法删除：该域名下存在活跃邮箱"
	MsgDomainInstructionsFailed = "获取配置说明失败"

	// 管理员相关
	MsgUserListFailed         = "获取用户列表失败"
	MsgUserNotFound           = "用户不存在"
	MsgUserGetFailed          = "获取用户信息失败"
	MsgUserUpdateFailed       = "更新用户信息失败"
	MsgUserDeleteFailed       = "删除用户失败"
	MsgDomainListFailedAdmin  = "获取域名列表失败"
	MsgDomainAddFailedAdmin   = "添加域名失败"
	MsgDomainAlreadyExists    = "域名已存在"
	MsgDomainRemoveFailed     = "删除域名失败"
	MsgDomainNotFoundAdmin    = "域名不存在"
	MsgCannotRemoveLastDomain = "不能删除最后一个域名"
	MsgStatisticsGetFailed    = "获取统计数据失败"
	MsgQuotaGetFailed         = "获取配额信息失败"
	MsgQuotaUpdateFailed      = "更新配额失败"

	// API Key相关
	MsgAPIKeyCreateFailed = "创建API Key失败"
	MsgAPIKeyListFailed   = "获取API Key列表失败"
	MsgAPIKeyNotFound     = "API Key不存在"
	MsgAPIKeyGetFailed    = "获取API Key详情失败"
	MsgAPIKeyDeleteFailed = "删除API Key失败"

	// 服务器错误
	MsgInternalError = "服务器内部错误，请稍后重试"
)

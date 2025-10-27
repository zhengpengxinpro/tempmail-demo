package httptransport

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`           // 业务状态码
	Msg  string      `json:"msg"`            // 中文提示信息
	Data interface{} `json:"data,omitempty"` // 数据载荷
}

// 业务状态码定义
const (
	// 成功状态码 2xx
	CodeSuccess   = 200 // 成功
	CodeCreated   = 201 // 创建成功
	CodeNoContent = 204 // 无内容（删除成功）

	// 客户端错误 4xx
	CodeBadRequest          = 400 // 请求参数错误
	CodeUnauthorized        = 401 // 未认证
	CodeForbidden           = 403 // 无权限
	CodeNotFound            = 404 // 资源不存在
	CodeConflict            = 409 // 资源冲突
	CodeUnprocessableEntity = 422 // 无法处理的实体

	// 服务器错误 5xx
	CodeInternalError = 500 // 服务器内部错误
)

// Success 成功响应（200）
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  "成功",
		Data: data,
	})
}

// SuccessWithMsg 成功响应（自定义消息）
func SuccessWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  msg,
		Data: data,
	})
}

// Created 创建成功响应（201）
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code: CodeCreated,
		Msg:  "创建成功",
		Data: data,
	})
}

// CreatedWithMsg 创建成功响应（自定义消息）
func CreatedWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code: CodeCreated,
		Msg:  msg,
		Data: data,
	})
}

// NoContent 无内容响应（204）- 通常用于删除成功
func NoContent(c *gin.Context) {
	c.JSON(http.StatusNoContent, Response{
		Code: CodeNoContent,
		Msg:  "操作成功",
		Data: nil,
	})
}

// BadRequest 请求参数错误（400）
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{
		Code: CodeBadRequest,
		Msg:  msg,
		Data: nil,
	})
}

// Unauthorized 未认证错误（401）
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{
		Code: CodeUnauthorized,
		Msg:  msg,
		Data: nil,
	})
}

// Forbidden 无权限错误（403）
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Response{
		Code: CodeForbidden,
		Msg:  msg,
		Data: nil,
	})
}

// NotFound 资源不存在错误（404）
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{
		Code: CodeNotFound,
		Msg:  msg,
		Data: nil,
	})
}

// Conflict 资源冲突错误（409）
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, Response{
		Code: CodeConflict,
		Msg:  msg,
		Data: nil,
	})
}

// UnprocessableEntity 无法处理的实体错误（422）
func UnprocessableEntity(c *gin.Context, msg string) {
	c.JSON(http.StatusUnprocessableEntity, Response{
		Code: CodeUnprocessableEntity,
		Msg:  msg,
		Data: nil,
	})
}

// InternalError 服务器内部错误（500）
func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{
		Code: CodeInternalError,
		Msg:  msg,
		Data: nil,
	})
}

// Error 通用错误响应（根据HTTP状态码自动选择）
func Error(c *gin.Context, httpCode int, msg string) {
	c.JSON(httpCode, Response{
		Code: httpCode,
		Msg:  msg,
		Data: nil,
	})
}

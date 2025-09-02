// Package response 提供HTTP响应数据的数据结构定义
// 本文件定义了统一的HTTP响应结构和便捷的响应方法，用于标准化API响应格式
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一HTTP响应结构
// 该结构体定义了所有API响应的标准格式，包含状态码、数据和消息三个字段
type Response struct {
	Code int    `json:"code"` // 响应状态码，用于标识请求处理结果
	Data any    `json:"data"` // 响应数据，包含业务数据或错误详情，支持任意类型
	Msg  string `json:"msg"`  // 响应消息，用于描述响应状态或提供用户提示
}

// Result 通用响应方法
// 该方法用于构建和返回标准的HTTP响应，支持自定义状态码、数据和消息
// 参数：
//   - c: Gin上下文，用于HTTP响应
//   - code: 响应状态码
//   - data: 响应数据
//   - msg: 响应消息
func Result(c *gin.Context, code int, data any, msg string) {
	c.JSON(http.StatusOK, Response{
		code,
		data,
		msg,
	})
}

// Ok 返回成功响应（无数据）
// 该方法返回标准的成功响应，使用预定义的成功状态码和消息
// 参数：
//   - c: Gin上下文，用于HTTP响应
func Ok(c *gin.Context) {
	Result(c, ResponseSuccess.code, map[string]any{}, ResponseSuccess.msg)
}

// OkWithMessage 返回成功响应（自定义消息）
// 该方法返回成功响应，支持自定义成功消息
// 参数：
//   - c: Gin上下文，用于HTTP响应
//   - message: 自定义成功消息
func OkWithMessage(c *gin.Context, message string) {
	Result(c, ResponseSuccess.code, map[string]any{}, message)
}

// OkWithData 返回成功响应（带数据）
// 该方法返回成功响应，包含业务数据
// 参数：
//   - c: Gin上下文，用于HTTP响应
//   - data: 业务数据
func OkWithData(c *gin.Context, data any) {
	Result(c, ResponseSuccess.code, data, ResponseSuccess.msg)
}

// OkWithDetail 返回成功响应（自定义消息和数据）
// 该方法返回成功响应，支持自定义消息和业务数据
// 参数：
//   - c: Gin上下文，用于HTTP响应
//   - message: 自定义成功消息
//   - data: 业务数据
func OkWithDetail(c *gin.Context, message string, data any) {
	Result(c, ResponseSuccess.code, data, message)
}

// Fail 返回失败响应（无数据）
// 该方法返回标准的失败响应，使用预定义的失败状态码和消息
// 参数：
//   - c: Gin上下文，用于HTTP响应
func Fail(c *gin.Context) {
	Result(c, ResponseFail.code, map[string]any{}, ResponseFail.msg)
}

// FailWithMessage 返回失败响应（自定义消息）
// 该方法返回失败响应，支持自定义失败消息
// 参数：
//   - c: Gin上下文，用于HTTP响应
//   - message: 自定义失败消息
func FailWithMessage(c *gin.Context, message string) {
	Result(c, ResponseFail.code, map[string]any{}, message)
}

// FailWithDetail 返回失败响应（自定义消息和数据）
// 该方法返回失败响应，支持自定义消息和错误详情数据
// 参数：
//   - c: Gin.Context，用于HTTP响应
//   - message: 自定义失败消息
//   - data: 错误详情数据
func FailWithDetail(c *gin.Context, message string, data any) {
	Result(c, ResponseFail.code, data, message)
}

// NoAuth 返回未授权响应
// 该方法返回HTTP 401状态码的未授权响应，用于认证失败场景
// 参数：
//   - c: Gin上下文，用于HTTP响应
//   - message: 未授权原因说明
func NoAuth(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		7,       // 使用特殊的未授权状态码
		nil,     // 无数据
		message, // 未授权原因
	})
}

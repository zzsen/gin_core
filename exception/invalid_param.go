package exception

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/zzsen/gin_core/model/response"
)

type InvalidParam struct {
	msg string
}

func (e InvalidParam) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return response.ResponseParamInvalid.GetMsg()
}

func NewInvalidParam(msg string) InvalidParam {
	return InvalidParam{msg: msg}
}

// NewInvalidParamFromValidator 从validator校验错误创建InvalidParam异常
// 将validator.ValidationErrors转换为InvalidParam异常
//
// 参数 validationErrors: validator校验错误集合
// 返回值: InvalidParam异常实例
func NewInvalidParamFromValidator(validationErrors validator.ValidationErrors) InvalidParam {
	return InvalidParam{msg: formatValidationErrors(validationErrors)}
}

func (e InvalidParam) OnException(*gin.Context) (msg string, code int) {
	return e.Error(), response.ResponseParamInvalid.GetCode()
}

// formatValidationErrors 格式化validator校验错误消息
// 将validator.ValidationErrors转换为可读的中文错误消息
//
// 参数 validationErrors: validator校验错误集合
// 返回值: 格式化后的错误消息字符串
//
// 处理逻辑：
// 1. 遍历所有校验错误
// 2. 根据错误类型生成对应的中文错误消息
// 3. 将所有错误消息用分号连接
func formatValidationErrors(validationErrors validator.ValidationErrors) string {
	messages := []string{"【参数校验不通过】"}
	for _, err := range validationErrors {
		// 获取字段名（优先使用命名空间以保留嵌套路径）
		// Namespace 返回完整路径如 "ApiResponseBatchRequest.Responses[0].ApiID"
		// 移除顶层结构体名称，保留有意义的字段路径
		namespace := err.Namespace()
		field := err.Field()
		// 如果是嵌套结构，使用去除顶层结构体后的路径
		if idx := strings.Index(namespace, "."); idx != -1 {
			field = namespace[idx+1:]
		}
		// 获取校验标签
		tag := err.Tag()
		// 获取字段值
		value := err.Value()

		// 根据校验标签生成对应的错误消息
		var msg string
		switch tag {
		case "required":
			msg = fmt.Sprintf("%s不能为空", field)
		case "min":
			msg = fmt.Sprintf("%s的值不能小于%s", field, err.Param())
		case "max":
			msg = fmt.Sprintf("%s的值不能大于%s", field, err.Param())
		case "len":
			msg = fmt.Sprintf("%s的长度必须为%s", field, err.Param())
		case "email":
			msg = fmt.Sprintf("%s必须是有效的邮箱地址", field)
		case "url":
			msg = fmt.Sprintf("%s必须是有效的URL地址", field)
		case "numeric":
			msg = fmt.Sprintf("%s必须是数字", field)
		case "alpha":
			msg = fmt.Sprintf("%s只能包含字母", field)
		case "alphanum":
			msg = fmt.Sprintf("%s只能包含字母和数字", field)
		case "gte":
			msg = fmt.Sprintf("%s的值必须大于或等于%s", field, err.Param())
		case "lte":
			msg = fmt.Sprintf("%s的值必须小于或等于%s", field, err.Param())
		case "gt":
			msg = fmt.Sprintf("%s的值必须大于%s", field, err.Param())
		case "lt":
			msg = fmt.Sprintf("%s的值必须小于%s", field, err.Param())
		case "oneof":
			msg = fmt.Sprintf("%s的值必须是以下之一: %s", field, err.Param())
		default:
			// 默认错误消息
			msg = fmt.Sprintf("%s校验失败(标签: %s, 值: %v)", field, tag, value)
		}
		messages = append(messages, msg)
	}

	// 将所有错误消息用分号连接
	return strings.Join(messages, "; ")
}

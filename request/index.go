// Package request 提供 HTTP 请求处理工具，包括参数校验等功能。
package request

import (
	"github.com/go-playground/validator/v10"
	"github.com/zzsen/gin_core/exception"
)

// Validate 对结构体执行参数校验，校验规则基于 validator v10 的 struct tag。
// 校验失败时直接 panic InvalidParam 异常，由框架 recover 中间件统一处理。
//
// 参数：
//   - s: 待校验的结构体实例（需在字段上定义 validate tag）
//
// 注意：此函数会 panic，请确保在有 recover 中间件保护的 handler 链中调用。
func Validate(s any) {
	err := validator.New().Struct(s)
	if err != nil {
		panic(exception.NewInvalidParam(err.Error()))
	}
}

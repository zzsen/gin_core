// Package exception 异常体系单元测试
//
// ==================== 测试说明 ====================
// 本文件包含框架异常体系的单元测试。
//
// 测试覆盖内容：
// 1. CommonError 创建、Error()、OnException()
// 2. AuthFailed Error()、OnException()
// 3. InvalidParam 创建、Error()（含默认消息）、OnException()
// 4. NewInvalidParamFromValidator 格式化校验错误
// 5. RpcError 创建、Error()、OnException()
// 6. InitError / InitErrorWithConfig 创建、Error()、Unwrap()
// 7. Handler 接口实现验证
//
// 运行测试：go test -v ./exception/... -run "Test"
// ==================================================
package exception

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/response"
)

// TestCommonError_New 测试 CommonError 创建
func TestCommonError_New(t *testing.T) {
	e := NewCommonError("测试错误")
	assert.Equal(t, "测试错误", e.Error())
}

// TestCommonError_OnException 测试 CommonError 异常处理
func TestCommonError_OnException(t *testing.T) {
	e := NewCommonError("业务异常")
	msg, code := e.OnException(nil)
	assert.Equal(t, "业务异常", msg)
	assert.Equal(t, response.ResponseExceptionCommon.GetCode(), code)
}

// TestCommonError_ImplementsHandler 验证 CommonError 实现 Handler 接口
func TestCommonError_ImplementsHandler(t *testing.T) {
	var _ Handler = CommonError{}
}

// TestAuthFailed_Error 测试 AuthFailed 错误消息
func TestAuthFailed_Error(t *testing.T) {
	e := AuthFailed{}
	assert.Equal(t, response.ResponseAuthFailed.GetMsg(), e.Error())
}

// TestAuthFailed_OnException 测试 AuthFailed 异常处理
func TestAuthFailed_OnException(t *testing.T) {
	e := AuthFailed{}
	msg, code := e.OnException(nil)
	assert.Equal(t, response.ResponseAuthFailed.GetMsg(), msg)
	assert.Equal(t, response.ResponseAuthFailed.GetCode(), code)
}

// TestInvalidParam_WithMsg 测试自定义消息的 InvalidParam
func TestInvalidParam_WithMsg(t *testing.T) {
	e := NewInvalidParam("用户名不能为空")
	assert.Equal(t, "用户名不能为空", e.Error())
}

// TestInvalidParam_DefaultMsg 测试默认消息的 InvalidParam
func TestInvalidParam_DefaultMsg(t *testing.T) {
	e := InvalidParam{}
	assert.Equal(t, response.ResponseParamInvalid.GetMsg(), e.Error())
}

// TestInvalidParam_OnException 测试 InvalidParam 异常处理
func TestInvalidParam_OnException(t *testing.T) {
	e := NewInvalidParam("参数错误")
	msg, code := e.OnException(nil)
	assert.Equal(t, "参数错误", msg)
	assert.Equal(t, response.ResponseParamInvalid.GetCode(), code)
}

// TestRpcError_New 测试 RpcError 创建
func TestRpcError_New(t *testing.T) {
	e := NewRpcError("服务调用超时")
	assert.Equal(t, "服务调用超时", e.Error())
}

// TestRpcError_OnException 测试 RpcError 异常处理
func TestRpcError_OnException(t *testing.T) {
	e := NewRpcError("RPC 失败")
	msg, code := e.OnException(nil)
	assert.Equal(t, "RPC 失败", msg)
	assert.Equal(t, response.ResponseExceptionRpc.GetCode(), code)
}

// TestInitError_Basic 测试 InitError 基本创建
func TestInitError_Basic(t *testing.T) {
	inner := errors.New("connection refused")
	e := NewInitError("db", "初始化连接", inner)

	assert.Contains(t, e.Error(), "[db]")
	assert.Contains(t, e.Error(), "初始化连接失败")
	assert.Contains(t, e.Error(), "connection refused")
}

// TestInitError_WithConfig 测试带配置的 InitError
func TestInitError_WithConfig(t *testing.T) {
	inner := errors.New("timeout")
	e := NewInitErrorWithConfig("redis", "创建客户端", "redis-main", inner)

	assert.Contains(t, e.Error(), "[redis]")
	assert.Contains(t, e.Error(), "[redis-main]")
	assert.Contains(t, e.Error(), "timeout")
}

// TestInitError_Unwrap 测试 InitError Unwrap
func TestInitError_Unwrap(t *testing.T) {
	inner := errors.New("root cause")
	e := NewInitError("es", "init", inner)

	require.True(t, errors.Is(e, inner))
	assert.Equal(t, inner, errors.Unwrap(e))
}

// ==================== formatValidationErrors ====================

type validationTarget struct {
	Name     string `validate:"required"`
	Email    string `validate:"required,email"`
	Age      int    `validate:"gte=0,lte=150"`
	Score    int    `validate:"min=0,max=100"`
	Len5     string `validate:"len=5"`
	URL      string `validate:"url"`
	Num      string `validate:"numeric"`
	Alpha    string `validate:"alpha"`
	AlphaNum string `validate:"alphanum"`
	Role     string `validate:"oneof=admin user guest"`
	Gt10     int    `validate:"gt=10"`
	Lt5      int    `validate:"lt=5"`
}

// TestNewInvalidParamFromValidator 测试从 validator 错误创建 InvalidParam
func TestNewInvalidParamFromValidator(t *testing.T) {
	v := validator.New()
	err := v.Struct(validationTarget{})
	require.Error(t, err)

	valErrs, ok := err.(validator.ValidationErrors)
	require.True(t, ok)

	e := NewInvalidParamFromValidator(valErrs)
	msg := e.Error()
	assert.Contains(t, msg, "【参数校验不通过】")
	assert.Contains(t, msg, "不能为空")
}

// TestFormatValidationErrors_AllTags 测试 formatValidationErrors 覆盖所有 switch 分支
func TestFormatValidationErrors_AllTags(t *testing.T) {
	v := validator.New()

	target := validationTarget{
		Name:     "",         // required
		Email:    "bad",      // email
		Age:      -1,         // gte
		Score:    200,        // max
		Len5:     "ab",       // len
		URL:      "noturl",   // url
		Num:      "abc",      // numeric
		Alpha:    "123",      // alpha
		AlphaNum: "a b",      // alphanum
		Role:     "superadm", // oneof
		Gt10:     5,          // gt
		Lt5:      10,         // lt
	}
	err := v.Struct(target)
	require.Error(t, err)

	valErrs := err.(validator.ValidationErrors)
	e := NewInvalidParamFromValidator(valErrs)
	msg := e.Error()

	assert.Contains(t, msg, "不能为空")
	assert.Contains(t, msg, "邮箱地址")
	assert.Contains(t, msg, "大于或等于")
	assert.Contains(t, msg, "不能大于")
	assert.Contains(t, msg, "长度必须为")
	assert.Contains(t, msg, "URL")
	assert.Contains(t, msg, "数字")
	assert.Contains(t, msg, "只能包含字母和数字")
	assert.Contains(t, msg, "以下之一")
	assert.Contains(t, msg, "必须大于")
	assert.Contains(t, msg, "必须小于")
}

// TestFormatValidationErrors_LteFallback 测试 lte 标签和 default 分支
func TestFormatValidationErrors_LteFallback(t *testing.T) {
	type lteTarget struct {
		Val int `validate:"lte=10"`
	}
	v := validator.New()
	err := v.Struct(lteTarget{Val: 20})
	require.Error(t, err)

	valErrs := err.(validator.ValidationErrors)
	e := NewInvalidParamFromValidator(valErrs)
	assert.Contains(t, e.Error(), "小于或等于")
}

// TestFormatValidationErrors_Min 测试 min 标签
func TestFormatValidationErrors_Min(t *testing.T) {
	type minTarget struct {
		Val int `validate:"min=5"`
	}
	v := validator.New()
	err := v.Struct(minTarget{Val: 2})
	require.Error(t, err)

	valErrs := err.(validator.ValidationErrors)
	e := NewInvalidParamFromValidator(valErrs)
	assert.Contains(t, e.Error(), "不能小于")
}

// TestFormatValidationErrors_Alpha 测试 alpha 标签（单独验证）
func TestFormatValidationErrors_Alpha(t *testing.T) {
	type alphaTarget struct {
		Val string `validate:"alpha"`
	}
	v := validator.New()
	err := v.Struct(alphaTarget{Val: "123"})
	require.Error(t, err)

	valErrs := err.(validator.ValidationErrors)
	e := NewInvalidParamFromValidator(valErrs)
	assert.Contains(t, e.Error(), "只能包含字母")
	assert.NotContains(t, e.Error(), "字母和数字")
}

// TestFormatValidationErrors_Default 测试 default 分支（未知标签）
func TestFormatValidationErrors_Default(t *testing.T) {
	type customTarget struct {
		Val string `validate:"contains=abc"`
	}
	v := validator.New()
	err := v.Struct(customTarget{Val: "xyz"})
	require.Error(t, err)

	valErrs := err.(validator.ValidationErrors)
	e := NewInvalidParamFromValidator(valErrs)
	assert.Contains(t, e.Error(), "校验失败")
	assert.Contains(t, e.Error(), "标签")
}

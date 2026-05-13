// Package request 请求校验工具测试
//
// ==================== 测试说明 ====================
// 本文件包含请求参数校验工具的单元测试。
//
// 测试覆盖内容：
// 1. Validate 校验通过不 panic
// 2. Validate 校验失败时 panic InvalidParam
// 3. 多字段校验失败
//
// 运行测试：go test -v ./request/... -run "Test"
// ==================================================
package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zzsen/gin_core/exception"
)

type testReq struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lte=150"`
}

// TestValidate_Pass 测试校验通过
func TestValidate_Pass(t *testing.T) {
	req := testReq{Name: "test", Email: "a@b.com", Age: 25}
	assert.NotPanics(t, func() { Validate(req) })
}

// TestValidate_Fail 测试校验失败时 panic
func TestValidate_Fail(t *testing.T) {
	req := testReq{Name: "", Email: "invalid", Age: -1}
	assert.Panics(t, func() { Validate(req) })
}

// TestValidate_PanicType 测试 panic 的类型为 InvalidParam
func TestValidate_PanicType(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		_, ok := r.(exception.InvalidParam)
		assert.True(t, ok, "panic 应为 InvalidParam 类型")
	}()
	Validate(testReq{})
}

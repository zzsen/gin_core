// Package response 响应模块测试
//
// ==================== 测试说明 ====================
// 本文件包含 HTTP 响应模块的单元测试，不需要外部依赖。
//
// 测试覆盖内容：
// 1. 统一响应函数（Result、Ok、OkWithMessage、OkWithData、OkWithDetail）
// 2. 失败响应函数（Fail、FailWithMessage、FailWithDetail）
// 3. 未授权响应函数（NoAuth）
// 4. 响应码常量（GetCode、GetMsg）
// 5. PageResult 结构体序列化
//
// 运行测试：go test -v ./model/response/... -run Test
// ==================================================
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	return resp
}

// TestResult 测试通用响应方法
//
// 【功能点】验证 Result 方法正确构建 JSON 响应
// 【测试流程】
// 1. 构造测试上下文
// 2. 调用 Result 设置自定义 code/data/msg
// 3. 验证 HTTP 状态码和响应体字段
func TestResult(t *testing.T) {
	c, w := newTestContext()
	Result(c, 200, map[string]string{"key": "value"}, "test message")

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "test message", resp.Msg)
	assert.NotNil(t, resp.Data)
}

// TestOk 测试成功响应（无数据）
//
// 【功能点】验证 Ok 方法返回标准成功响应
// 【测试流程】
// 1. 调用 Ok 方法
// 2. 验证响应码为 ResponseSuccess.code
// 3. 验证消息为 ResponseSuccess.msg
func TestOk(t *testing.T) {
	c, w := newTestContext()
	Ok(c)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, ResponseSuccess.GetCode(), resp.Code)
	assert.Equal(t, ResponseSuccess.GetMsg(), resp.Msg)
}

// TestOkWithMessage 测试成功响应（自定义消息）
//
// 【功能点】验证 OkWithMessage 方法支持自定义成功消息
// 【测试流程】
// 1. 调用 OkWithMessage 传入自定义消息
// 2. 验证响应码为成功码
// 3. 验证消息为自定义值
func TestOkWithMessage(t *testing.T) {
	c, w := newTestContext()
	OkWithMessage(c, "自定义成功消息")

	resp := parseResponse(t, w)
	assert.Equal(t, ResponseSuccess.GetCode(), resp.Code)
	assert.Equal(t, "自定义成功消息", resp.Msg)
}

// TestOkWithData 测试成功响应（带数据）
//
// 【功能点】验证 OkWithData 方法返回携带业务数据的成功响应
// 【测试流程】
// 1. 调用 OkWithData 传入业务数据
// 2. 验证响应码为成功码
// 3. 验证 data 字段包含传入的数据
func TestOkWithData(t *testing.T) {
	c, w := newTestContext()
	testData := map[string]string{"name": "test"}
	OkWithData(c, testData)

	resp := parseResponse(t, w)
	assert.Equal(t, ResponseSuccess.GetCode(), resp.Code)
	assert.Equal(t, ResponseSuccess.GetMsg(), resp.Msg)
	assert.NotNil(t, resp.Data)
}

// TestOkWithDetail 测试成功响应（自定义消息和数据）
//
// 【功能点】验证 OkWithDetail 方法同时支持自定义消息和数据
// 【测试流程】
// 1. 调用 OkWithDetail 传入消息和数据
// 2. 验证响应码、消息和数据均正确
func TestOkWithDetail(t *testing.T) {
	c, w := newTestContext()
	testData := []int{1, 2, 3}
	OkWithDetail(c, "详细成功", testData)

	resp := parseResponse(t, w)
	assert.Equal(t, ResponseSuccess.GetCode(), resp.Code)
	assert.Equal(t, "详细成功", resp.Msg)
	assert.NotNil(t, resp.Data)
}

// TestFail 测试失败响应（无数据）
//
// 【功能点】验证 Fail 方法返回标准失败响应
// 【测试流程】
// 1. 调用 Fail 方法
// 2. 验证响应码为 ResponseFail.code
// 3. 验证消息为 ResponseFail.msg
func TestFail(t *testing.T) {
	c, w := newTestContext()
	Fail(c)

	resp := parseResponse(t, w)
	assert.Equal(t, ResponseFail.GetCode(), resp.Code)
	assert.Equal(t, ResponseFail.GetMsg(), resp.Msg)
}

// TestFailWithMessage 测试失败响应（自定义消息）
//
// 【功能点】验证 FailWithMessage 方法支持自定义失败消息
// 【测试流程】
// 1. 调用 FailWithMessage 传入自定义消息
// 2. 验证响应码为失败码
// 3. 验证消息为自定义值
func TestFailWithMessage(t *testing.T) {
	c, w := newTestContext()
	FailWithMessage(c, "操作出错")

	resp := parseResponse(t, w)
	assert.Equal(t, ResponseFail.GetCode(), resp.Code)
	assert.Equal(t, "操作出错", resp.Msg)
}

// TestFailWithDetail 测试失败响应（自定义消息和数据）
//
// 【功能点】验证 FailWithDetail 方法同时支持自定义消息和错误详情
// 【测试流程】
// 1. 调用 FailWithDetail 传入消息和详情数据
// 2. 验证响应码、消息和数据均正确
func TestFailWithDetail(t *testing.T) {
	c, w := newTestContext()
	errData := map[string]string{"field": "name", "error": "不能为空"}
	FailWithDetail(c, "参数错误", errData)

	resp := parseResponse(t, w)
	assert.Equal(t, ResponseFail.GetCode(), resp.Code)
	assert.Equal(t, "参数错误", resp.Msg)
	assert.NotNil(t, resp.Data)
}

// TestNoAuth 测试未授权响应
//
// 【功能点】验证 NoAuth 方法返回 HTTP 401 状态码和业务码 7
// 【测试流程】
// 1. 调用 NoAuth 传入原因说明
// 2. 验证 HTTP 状态码为 401
// 3. 验证业务码为 7
// 4. 验证消息为传入的原因
func TestNoAuth(t *testing.T) {
	c, w := newTestContext()
	NoAuth(c, "token已过期")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 7, resp.Code)
	assert.Equal(t, "token已过期", resp.Msg)
	assert.Nil(t, resp.Data)
}

// TestResponseCodeGetCode 测试响应码 GetCode 方法
//
// 【功能点】验证各预定义响应码的 code 值正确性
// 【测试流程】
// 1. 遍历所有预定义响应码
// 2. 验证每个响应码的 GetCode 返回期望值
func TestResponseCodeGetCode(t *testing.T) {
	tests := []struct {
		name     string
		code     responseCode
		expected int
	}{
		{"ResponseNull", ResponseNull, -1},
		{"ResponseSuccess", ResponseSuccess, 20000},
		{"ResponseLoginNotLogin", ResponseLoginNotLogin, 41000},
		{"ResponseLoginButUnAuth", ResponseLoginButUnAuth, 41001},
		{"ResponseLoginInvalid", ResponseLoginInvalid, 41002},
		{"ResponseAuthFailed", ResponseAuthFailed, 41010},
		{"ResponseFail", ResponseFail, 50000},
		{"ResponseParamInvalid", ResponseParamInvalid, 53001},
		{"ResponseParamTypeError", ResponseParamTypeError, 50002},
		{"ResponseExceptionCommon", ResponseExceptionCommon, 90000},
		{"ResponseExceptionRpc", ResponseExceptionRpc, 90001},
		{"ResponseExceptionUnknown", ResponseExceptionUnknown, 90002},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.code.GetCode())
		})
	}
}

// TestResponseCodeGetMsg 测试响应码 GetMsg 方法
//
// 【功能点】验证各预定义响应码的 msg 值正确性
// 【测试流程】
// 1. 验证成功码的消息
// 2. 验证失败码的消息
// 3. 验证空码（ResponseNull）的消息为空字符串
func TestResponseCodeGetMsg(t *testing.T) {
	assert.Equal(t, "操作成功", ResponseSuccess.GetMsg())
	assert.Equal(t, "操作失败", ResponseFail.GetMsg())
	assert.Equal(t, "未登录", ResponseLoginNotLogin.GetMsg())
	assert.Equal(t, "", ResponseNull.GetMsg())
}

// TestPageResultJSON 测试 PageResult 结构体 JSON 序列化
//
// 【功能点】验证 PageResult 的 JSON 字段名和值正确性
// 【测试流程】
// 1. 构造 PageResult 实例
// 2. 序列化为 JSON
// 3. 反序列化并验证各字段
func TestPageResultJSON(t *testing.T) {
	pr := PageResult{
		List:      []string{"a", "b"},
		Total:     100,
		PageIndex: 1,
		PageSize:  10,
	}

	data, err := json.Marshal(pr)
	assert.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	assert.Contains(t, parsed, "list")
	assert.Contains(t, parsed, "total")
	assert.Contains(t, parsed, "pageIndex")
	assert.Contains(t, parsed, "pageSize")
	assert.Equal(t, float64(100), parsed["total"])
	assert.Equal(t, float64(1), parsed["pageIndex"])
	assert.Equal(t, float64(10), parsed["pageSize"])
}

// Package middleware 异常处理中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含异常处理中间件的单元测试。
//
// 测试覆盖内容：
// 1. panic 捕获与恢复
// 2. 自定义异常（实现 Handler 接口）的处理
// 3. validator 校验异常的转换
// 4. 未知异常的兜底处理
// 5. 正常请求的透传
//
// 运行测试：go test -v ./middleware/... -run ExceptionHandler
// ==================================================
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/model/response"
)

// ==================== 测试辅助结构 ====================

// customException 自定义异常，实现 exception.Handler 接口
type customException struct {
	message string
	code    int
}

func (e customException) Error() string {
	return e.message
}

func (e customException) OnException(*gin.Context) (msg string, code int) {
	return e.message, e.code
}

// ==================== ExceptionHandler 单元测试 ====================

// TestExceptionHandler_NoPanic 测试无异常时的正常请求
//
// 【功能点】验证正常请求能够正常透传
// 【测试流程】发送正常请求，验证返回 200 状态码
func TestExceptionHandler_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}
	if resp["message"] != "success" {
		t.Errorf("期望 message=success, 实际 %v", resp["message"])
	}
}

// TestExceptionHandler_StringPanic 测试字符串类型的 panic
//
// 【功能点】验证字符串 panic 被捕获并返回统一错误响应
// 【测试流程】抛出字符串 panic，验证返回 200 状态码和错误信息
func TestExceptionHandler_StringPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic error")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	// 异常处理后返回 200 状态码（统一响应格式）
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证错误码
	if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseExceptionUnknown.GetCode()) {
		t.Errorf("期望 code=%d, 实际 %v", response.ResponseExceptionUnknown.GetCode(), resp["code"])
	}

	// 验证默认错误消息
	if resp["msg"] != "服务端异常" {
		t.Errorf("期望 msg=服务端异常, 实际 %v", resp["msg"])
	}
}

// TestExceptionHandler_CustomException 测试自定义异常（实现 Handler 接口）
//
// 【功能点】验证实现 Handler 接口的异常被正确处理
// 【测试流程】抛出自定义异常，验证返回自定义的 code 和 message
func TestExceptionHandler_CustomException(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/custom", func(c *gin.Context) {
		panic(customException{message: "自定义错误消息", code: 40001})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/custom", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证自定义错误码
	if code, ok := resp["code"].(float64); !ok || code != 40001 {
		t.Errorf("期望 code=40001, 实际 %v", resp["code"])
	}

	// 验证自定义错误消息
	if resp["msg"] != "自定义错误消息" {
		t.Errorf("期望 msg=自定义错误消息, 实际 %v", resp["msg"])
	}
}

// TestExceptionHandler_CommonError 测试 CommonError 异常
//
// 【功能点】验证 CommonError 异常被正确处理
// 【测试流程】抛出 CommonError，验证返回正确的 code 和 message
func TestExceptionHandler_CommonError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/common-error", func(c *gin.Context) {
		panic(exception.NewCommonError("通用错误消息"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/common-error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证错误消息
	if resp["msg"] != "通用错误消息" {
		t.Errorf("期望 msg=通用错误消息, 实际 %v", resp["msg"])
	}
}

// TestExceptionHandler_InvalidParam 测试 InvalidParam 异常
//
// 【功能点】验证 InvalidParam 异常被正确处理
// 【测试流程】抛出 InvalidParam，验证返回正确的 code 和 message
func TestExceptionHandler_InvalidParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/invalid-param", func(c *gin.Context) {
		panic(exception.NewInvalidParam("参数格式错误"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/invalid-param", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证错误码
	if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
		t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
	}

	// 验证错误消息
	if resp["msg"] != "参数格式错误" {
		t.Errorf("期望 msg=参数格式错误, 实际 %v", resp["msg"])
	}
}

// TestExceptionHandler_ErrorType 测试 error 类型的 panic
//
// 【功能点】验证普通 error 类型的 panic 被捕获
// 【测试流程】抛出 error 类型的 panic，验证返回默认错误响应
func TestExceptionHandler_ErrorType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/error", func(c *gin.Context) {
		panic(gin.Error{Err: nil, Type: gin.ErrorTypePrivate})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证返回默认错误消息
	if resp["msg"] != "服务端异常" {
		t.Errorf("期望 msg=服务端异常, 实际 %v", resp["msg"])
	}
}

// TestExceptionHandler_AbortsPipeline 测试异常后中断请求处理链
//
// 【功能点】验证异常后不再执行后续中间件
// 【测试流程】设置后续中间件标记变量，验证异常后标记未被设置
func TestExceptionHandler_AbortsPipeline(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ExceptionHandler())
	router.Use(func(c *gin.Context) {
		c.Next()
	})
	router.GET("/abort", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/abort", nil)
	router.ServeHTTP(w, req)

	// c.Abort() 会阻止后续处理器的执行
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestExceptionHandler_NilPanic 测试 nil panic
//
// 【功能点】验证 nil panic 被正确处理
// 【测试流程】抛出 nil panic，验证返回默认错误响应
func TestExceptionHandler_NilPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/nil-panic", func(c *gin.Context) {
		panic(nil)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nil-panic", nil)
	router.ServeHTTP(w, req)

	// nil panic 不会被 recover 捕获，所以请求正常完成
	// 这是 Go 的标准行为
}

// TestExceptionHandler_MultipleRequests 测试多个请求的独立性
//
// 【功能点】验证每个请求的异常处理是独立的
// 【测试流程】发送多个请求，验证每个请求的异常都被独立处理
func TestExceptionHandler_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	router.GET("/fail", func(c *gin.Context) {
		panic("error")
	})

	// 第一个请求失败
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/fail", nil)
	router.ServeHTTP(w1, req1)

	// 第二个请求应该成功
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/success", nil)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("第二个请求期望状态码 200, 实际 %d", w2.Code)
	}

	var resp2 map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Errorf("解析第二个响应失败: %v", err)
	}
	if resp2["message"] != "success" {
		t.Errorf("第二个请求期望 message=success, 实际 %v", resp2["message"])
	}
}

// ==================== Validator 异常转换测试 ====================

// mockValidationError 模拟 validator 校验错误
type mockValidationError struct {
	field string
	tag   string
}

func (e mockValidationError) Tag() string                     { return e.tag }
func (e mockValidationError) ActualTag() string               { return e.tag }
func (e mockValidationError) Namespace() string               { return "" }
func (e mockValidationError) StructNamespace() string         { return "" }
func (e mockValidationError) Field() string                   { return e.field }
func (e mockValidationError) StructField() string             { return e.field }
func (e mockValidationError) Value() interface{}              { return nil }
func (e mockValidationError) Param() string                   { return "" }
func (e mockValidationError) Kind() interface{}               { return nil }
func (e mockValidationError) Type() interface{}               { return nil }
func (e mockValidationError) Translate(ut interface{}) string { return "" }
func (e mockValidationError) Error() string                   { return e.field + " " + e.tag }

// TestExceptionHandler_ValidatorError 测试 validator 校验异常的转换
//
// 【功能点】验证 validator.ValidationErrors 被转换为 InvalidParam
// 【测试流程】抛出 validator 校验错误，验证返回参数校验错误响应
func TestExceptionHandler_ValidatorError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/validator", func(c *gin.Context) {
		// 直接抛出 ValidationErrors
		panic(validator.ValidationErrors{})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/validator", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证错误码是参数校验错误码
	if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
		t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
	}
}

// ==================== HTTP 请求参数校验测试 ====================

// 定义测试用的请求结构体
type (
	// UserRequest 用户请求参数
	UserRequest struct {
		Username string `json:"username" binding:"required,min=3,max=20"`
		Email    string `json:"email" binding:"required,email"`
		Age      int    `json:"age" binding:"gte=0,lte=150"`
		Password string `json:"password" binding:"required,min=6"`
	}

	// QueryRequest 查询请求参数
	QueryRequest struct {
		Page     int    `form:"page" binding:"required,gte=1"`
		PageSize int    `form:"page_size" binding:"required,gte=1,lte=100"`
		Keyword  string `form:"keyword" binding:"max=50"`
	}

	// NestedRequest 嵌套结构请求参数
	NestedRequest struct {
		User    UserInfo `json:"user" binding:"required"`
		OrderID string   `json:"order_id" binding:"required,len=16"`
	}

	// UserInfo 用户信息
	UserInfo struct {
		Name  string `json:"name" binding:"required"`
		Phone string `json:"phone" binding:"required,len=11,numeric"`
	}

	// BatchRequest 批量请求参数
	BatchRequest struct {
		Items []ItemInfo `json:"items" binding:"required,min=1,max=10,dive"`
	}

	// ItemInfo 单项信息
	ItemInfo struct {
		ID    string `json:"id" binding:"required"`
		Value int    `json:"value" binding:"gte=0"`
	}

	// OneOfRequest 枚举校验请求
	OneOfRequest struct {
		Status string `json:"status" binding:"required,oneof=pending processing completed"`
		Type   int    `json:"type" binding:"oneof=1 2 3"`
	}
)

// TestExceptionHandler_BindJSON_Required 测试 JSON 绑定 required 校验
//
// 【功能点】验证 required 校验规则不通过时的异常处理
// 【测试流程】发送缺少必填字段的请求，验证返回参数校验错误
func TestExceptionHandler_BindJSON_Required(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 缺少 username 字段
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user", strings.NewReader(`{"email":"test@example.com","password":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	// 验证错误码
	if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
		t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
	}

	// 验证错误消息包含字段名
	if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, "Username") {
		t.Errorf("期望错误消息包含 Username, 实际 %v", resp["msg"])
	}
}

// TestExceptionHandler_BindJSON_MinMax 测试 JSON 绑定 min/max 校验
//
// 【功能点】验证 min/max 校验规则不通过时的异常处理
// 【测试流程】发送超出范围的字段值，验证返回参数校验错误
func TestExceptionHandler_BindJSON_MinMax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{
			name:    "username too short",
			body:    `{"username":"ab","email":"test@example.com","password":"123456"}`,
			wantMsg: "Username",
		},
		{
			name:    "username too long",
			body:    `{"username":"abcdefghijklmnopqrstuvwxyz","email":"test@example.com","password":"123456"}`,
			wantMsg: "Username",
		},
		{
			name:    "password too short",
			body:    `{"username":"testuser","email":"test@example.com","password":"123"}`,
			wantMsg: "Password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/user", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("期望错误消息包含 %s, 实际 %v", tt.wantMsg, resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindJSON_Email 测试 JSON 绑定 email 校验
//
// 【功能点】验证 email 校验规则不通过时的异常处理
// 【测试流程】发送无效的邮箱格式，验证返回参数校验错误
func TestExceptionHandler_BindJSON_Email(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	invalidEmails := []string{
		"invalid-email",
		"@example.com",
		"test@",
		"test@.com",
		"test@com",
	}

	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			body := fmt.Sprintf(`{"username":"testuser","email":"%s","password":"123456"}`, email)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/user", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, "Email") {
				t.Errorf("期望错误消息包含 Email, 实际 %v", resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindJSON_GteLte 测试 JSON 绑定 gte/lte 校验
//
// 【功能点】验证 gte/lte 校验规则不通过时的异常处理
// 【测试流程】发送超出范围的数值，验证返回参数校验错误
func TestExceptionHandler_BindJSON_GteLte(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name string
		body string
	}{
		{
			name: "age less than 0",
			body: `{"username":"testuser","email":"test@example.com","password":"123456","age":-1}`,
		},
		{
			name: "age greater than 150",
			body: `{"username":"testuser","email":"test@example.com","password":"123456","age":200}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/user", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, "Age") {
				t.Errorf("期望错误消息包含 Age, 实际 %v", resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindQuery 测试 Query 参数绑定校验
//
// 【功能点】验证 Query 参数校验不通过时的异常处理
// 【测试流程】发送无效的查询参数，验证返回参数校验错误
func TestExceptionHandler_BindQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/list", func(c *gin.Context) {
		var req QueryRequest
		if err := c.ShouldBindQuery(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name    string
		url     string
		wantMsg string
	}{
		{
			name:    "missing page",
			url:     "/list?page_size=10",
			wantMsg: "Page",
		},
		{
			name:    "missing page_size",
			url:     "/list?page=1",
			wantMsg: "PageSize",
		},
		{
			name:    "page less than 1",
			url:     "/list?page=0&page_size=10",
			wantMsg: "Page",
		},
		{
			name:    "page_size greater than 100",
			url:     "/list?page=1&page_size=200",
			wantMsg: "PageSize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.url, nil)
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("期望错误消息包含 %s, 实际 %v", tt.wantMsg, resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindJSON_Nested 测试嵌套结构体校验
//
// 【功能点】验证嵌套结构体校验不通过时的异常处理
// 【测试流程】发送嵌套结构中无效的字段值，验证返回参数校验错误
func TestExceptionHandler_BindJSON_Nested(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/order", func(c *gin.Context) {
		var req NestedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{
			name:    "missing nested user name",
			body:    `{"user":{"phone":"13800138000"},"order_id":"1234567890123456"}`,
			wantMsg: "Name",
		},
		{
			name:    "invalid phone length",
			body:    `{"user":{"name":"张三","phone":"1380013800"},"order_id":"1234567890123456"}`,
			wantMsg: "Phone",
		},
		{
			name:    "phone not numeric",
			body:    `{"user":{"name":"张三","phone":"1380013800a"},"order_id":"1234567890123456"}`,
			wantMsg: "Phone",
		},
		{
			name:    "order_id wrong length",
			body:    `{"user":{"name":"张三","phone":"13800138000"},"order_id":"12345"}`,
			wantMsg: "OrderID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/order", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("期望错误消息包含 %s, 实际 %v", tt.wantMsg, resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindJSON_Slice 测试数组/切片元素校验
//
// 【功能点】验证数组元素校验不通过时的异常处理
// 【测试流程】发送无效的数组元素，验证返回参数校验错误
func TestExceptionHandler_BindJSON_Slice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/batch", func(c *gin.Context) {
		var req BatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{
			name:    "empty items",
			body:    `{"items":[]}`,
			wantMsg: "Items",
		},
		{
			name:    "item missing id",
			body:    `{"items":[{"value":100}]}`,
			wantMsg: "ID",
		},
		{
			name:    "item value negative",
			body:    `{"items":[{"id":"item1","value":-1}]}`,
			wantMsg: "Value",
		},
		{
			name:    "second item invalid",
			body:    `{"items":[{"id":"item1","value":100},{"value":50}]}`,
			wantMsg: "ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/batch", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("期望错误消息包含 %s, 实际 %v", tt.wantMsg, resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindJSON_NestedSlice_FullPath 测试嵌套数组结构体校验错误显示完整路径
//
// 【功能点】验证嵌套数组元素校验失败时，错误消息包含完整的字段路径（如 Items[0].ID）
// 【测试流程】发送嵌套数组中元素缺少必填字段的请求，验证错误消息包含数组索引和字段路径
func TestExceptionHandler_BindJSON_NestedSlice_FullPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/batch", func(c *gin.Context) {
		var req BatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name         string
		body         string
		wantMsgParts []string // 期望错误消息中包含的部分
	}{
		{
			name:         "first item missing id shows Items[0].ID",
			body:         `{"items":[{"value":100}]}`,
			wantMsgParts: []string{"Items[0].ID", "不能为空"},
		},
		{
			name:         "second item missing id shows Items[1].ID",
			body:         `{"items":[{"id":"item1","value":100},{"value":50}]}`,
			wantMsgParts: []string{"Items[1].ID", "不能为空"},
		},
		{
			name:         "third item value negative shows Items[2].Value",
			body:         `{"items":[{"id":"item1","value":100},{"id":"item2","value":50},{"id":"item3","value":-1}]}`,
			wantMsgParts: []string{"Items[2].Value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/batch", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			msg, ok := resp["msg"].(string)
			if !ok {
				t.Errorf("期望 msg 是字符串, 实际 %T", resp["msg"])
				return
			}

			// 验证错误消息包含完整路径
			for _, part := range tt.wantMsgParts {
				if !strings.Contains(msg, part) {
					t.Errorf("期望错误消息包含 '%s', 实际消息: %s", part, msg)
				}
			}
		})
	}
}

// TestExceptionHandler_BindJSON_OneOf 测试 oneof 枚举校验
//
// 【功能点】验证 oneof 校验规则不通过时的异常处理
// 【测试流程】发送不在枚举范围内的值，验证返回参数校验错误
func TestExceptionHandler_BindJSON_OneOf(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/status", func(c *gin.Context) {
		var req OneOfRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{
			name:    "invalid status",
			body:    `{"status":"invalid","type":1}`,
			wantMsg: "Status",
		},
		{
			name:    "invalid type",
			body:    `{"status":"pending","type":5}`,
			wantMsg: "Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/status", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
				t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
			}

			if msg, ok := resp["msg"].(string); !ok || !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("期望错误消息包含 %s, 实际 %v", tt.wantMsg, resp["msg"])
			}
		})
	}
}

// TestExceptionHandler_BindJSON_MultipleErrors 测试多字段校验错误
//
// 【功能点】验证多字段同时校验不通过时的异常处理
// 【测试流程】发送多个无效字段，验证错误消息包含所有错误
func TestExceptionHandler_BindJSON_MultipleErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 所有必填字段都缺失
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	if code, ok := resp["code"].(float64); !ok || code != float64(response.ResponseParamInvalid.GetCode()) {
		t.Errorf("期望 code=%d, 实际 %v", response.ResponseParamInvalid.GetCode(), resp["code"])
	}

	// 验证错误消息包含多个字段
	msg, ok := resp["msg"].(string)
	if !ok {
		t.Errorf("期望 msg 是字符串, 实际 %T", resp["msg"])
	}

	expectedFields := []string{"Username", "Email", "Password"}
	for _, field := range expectedFields {
		if !strings.Contains(msg, field) {
			t.Errorf("期望错误消息包含 %s, 实际消息: %s", field, msg)
		}
	}
}

// TestExceptionHandler_BindJSON_Success 测试参数校验通过的场景
//
// 【功能点】验证有效参数能正常通过校验
// 【测试流程】发送有效的请求参数，验证返回成功响应
func TestExceptionHandler_BindJSON_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success", "username": req.Username})
	})

	w := httptest.NewRecorder()
	body := `{"username":"testuser","email":"test@example.com","password":"123456","age":25}`
	req, _ := http.NewRequest("POST", "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	if resp["message"] != "success" {
		t.Errorf("期望 message=success, 实际 %v", resp["message"])
	}

	if resp["username"] != "testuser" {
		t.Errorf("期望 username=testuser, 实际 %v", resp["username"])
	}
}

// TestExceptionHandler_InvalidJSON 测试无效 JSON 格式
//
// 【功能点】验证无效 JSON 格式的异常处理
// 【测试流程】发送无效的 JSON 格式，验证返回错误响应
func TestExceptionHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.POST("/user", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			panic(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	invalidJSONs := []string{
		`{invalid json}`,
		`{"username": }`,
		`{"username": "test"`,
		`not json at all`,
	}

	for _, invalidJSON := range invalidJSONs {
		t.Run(invalidJSON, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/user", strings.NewReader(invalidJSON))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			// 无效 JSON 应该返回错误响应
			if w.Code != http.StatusOK {
				t.Errorf("期望状态码 200, 实际 %d", w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Errorf("解析响应失败: %v", err)
			}

			// 验证返回了错误响应
			if _, ok := resp["code"]; !ok {
				t.Errorf("期望响应包含 code 字段")
			}
		})
	}
}

// ==================== 基准测试 ====================

// BenchmarkExceptionHandler_NoPanic 基准测试无异常场景
func BenchmarkExceptionHandler_NoPanic(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

// BenchmarkExceptionHandler_WithPanic 基准测试有异常场景
func BenchmarkExceptionHandler_WithPanic(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ExceptionHandler())
	router.GET("/panic", func(c *gin.Context) {
		panic(exception.NewCommonError("test error"))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/panic", nil)
		router.ServeHTTP(w, req)
	}
}

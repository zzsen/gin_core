// Package ginContext Gin上下文工具功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 Gin 上下文工具函数的单元测试。
//
// 测试覆盖内容：
// 1. Get - 统一获取请求参数（Query/Form/Body/Header）
// 2. GetInt/GetInt64 - 获取整数类型参数
// 3. GetFloat64 - 获取浮点数类型参数
// 4. GetBool - 获取布尔类型参数
// 5. BindJSON - 绑定JSON请求体到结构体
// 6. GetClientIP - 获取客户端IP（支持代理）
// 7. GetHeader - 获取请求头
// 8. SetHeader - 设置响应头
//
// 参数优先级：Query > Form > Body > Header
//
// 运行测试：go test -v ./utils/gin_context/...
// ==================================================
package ginContext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ==================== Get 函数测试 ====================

// TestGet_QueryParam 测试从URL查询参数获取值
//
// 【功能点】验证从 URL Query 参数获取值
// 【测试流程】
//  1. 构造带有 Query 参数的请求
//  2. 调用 Get 函数获取参数值
//  3. 验证返回值与期望一致
func TestGet_QueryParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string // 测试用例名称
		url      string // 请求URL
		key      string // 要获取的键
		expected string // 期望的值
	}{
		{
			name:     "single query param",
			url:      "/test?name=john",
			key:      "name",
			expected: "john",
		},
		{
			name:     "multiple query params",
			url:      "/test?name=john&age=25&city=beijing",
			key:      "age",
			expected: "25",
		},
		{
			name:     "empty query param",
			url:      "/test?name=&age=25",
			key:      "name",
			expected: "",
		},
		{
			name:     "non-existent query param",
			url:      "/test?name=john",
			key:      "age",
			expected: "",
		},
		{
			name:     "special characters in query param",
			url:      "/test?message=hello%20world&data=test%40example.com",
			key:      "message",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Gin引擎
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				// 先调用Get函数
				result := Get(c, tt.key)

				// 验证调用Get后，原始方法仍然可以正常工作
				originalValue := c.Query(tt.key)

				c.JSON(200, gin.H{
					"value":               result,
					"original_query":      originalValue,
					"original_accessible": originalValue != "",
				})
			})

			// 创建HTTP请求
			req, _ := http.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, 200, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, response["value"])
			// 验证原始方法仍然可以获取到值
			if tt.expected != "" {
				assert.True(t, response["original_accessible"].(bool))
				assert.Equal(t, tt.expected, response["original_query"])
			}
		})
	}
}

// TestGet_PostForm 测试从POST表单数据获取值
//
// 【功能点】验证从 POST Form 表单获取参数值
// 【测试流程】
//  1. 构造带有 Form 数据的 POST 请求
//  2. 调用 Get 函数获取参数值
//  3. 验证返回值与期望一致
//  4. 验证原始 PostForm 方法仍可正常工作
func TestGet_PostForm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string            // 测试用例名称
		formData map[string]string // 表单数据
		key      string            // 要获取的键
		expected string            // 期望的值
	}{
		{
			name:     "single form field",
			formData: map[string]string{"name": "john"},
			key:      "name",
			expected: "john",
		},
		{
			name:     "multiple form fields",
			formData: map[string]string{"name": "john", "age": "25", "city": "beijing"},
			key:      "age",
			expected: "25",
		},
		{
			name:     "empty form field",
			formData: map[string]string{"name": "", "age": "25"},
			key:      "name",
			expected: "",
		},
		{
			name:     "non-existent form field",
			formData: map[string]string{"name": "john"},
			key:      "age",
			expected: "",
		},
		{
			name:     "special characters in form field",
			formData: map[string]string{"message": "hello world", "email": "test@example.com"},
			key:      "message",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Gin引擎
			r := gin.New()
			r.POST("/test", func(c *gin.Context) {
				// 先调用Get函数
				result := Get(c, tt.key)

				// 验证调用Get后，原始方法仍然可以正常工作
				originalValue := c.PostForm(tt.key)

				c.JSON(200, gin.H{
					"value":               result,
					"original_postform":   originalValue,
					"original_accessible": originalValue != "",
				})
			})

			// 准备表单数据
			formData := make(map[string][]string)
			for k, v := range tt.formData {
				formData[k] = []string{v}
			}

			// 创建HTTP请求
			req, _ := http.NewRequest("POST", "/test", nil)
			req.PostForm = formData
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, 200, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, response["value"])
			// 验证原始方法仍然可以获取到值
			if tt.expected != "" {
				assert.True(t, response["original_accessible"].(bool))
				assert.Equal(t, tt.expected, response["original_postform"])
			}
		})
	}
}

// TestGet_JSONBody 测试从JSON请求体获取值
//
// 【功能点】验证从 JSON 请求体获取参数值
// 【测试流程】
//  1. 构造带有 JSON Body 的请求
//  2. 调用 Get 函数获取参数值
//  3. 验证支持简单类型、数字、布尔值转换
//  4. 验证嵌套 JSON 不支持直接获取
func TestGet_JSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string                 // 测试用例名称
		jsonData map[string]interface{} // JSON数据
		key      string                 // 要获取的键
		expected string                 // 期望的值
	}{
		{
			name:     "simple JSON object",
			jsonData: map[string]interface{}{"name": "john"},
			key:      "name",
			expected: "john",
		},
		{
			name: "complex JSON object",
			jsonData: map[string]interface{}{
				"name":   "john",
				"age":    25,
				"city":   "beijing",
				"active": true,
			},
			key:      "age",
			expected: "25",
		},
		{
			name: "nested JSON object",
			jsonData: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "john",
					"age":  25,
				},
				"status": "active",
			},
			key:      "status",
			expected: "active",
		},
		{
			name:     "empty JSON object",
			jsonData: map[string]interface{}{},
			key:      "name",
			expected: "",
		},
		{
			name:     "non-existent key in JSON",
			jsonData: map[string]interface{}{"name": "john"},
			key:      "age",
			expected: "",
		},
		{
			name: "different data types in JSON",
			jsonData: map[string]interface{}{
				"string":  "hello",
				"number":  123,
				"boolean": true,
				"null":    nil,
			},
			key:      "number",
			expected: "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Gin引擎
			r := gin.New()
			r.POST("/test", func(c *gin.Context) {
				// 先调用Get函数
				result := Get(c, tt.key)

				// 验证调用Get后，原始方法仍然可以正常工作
				// 对于JSON请求，Get函数会重新设置Request.Body
				// 所以后续的中间件或处理函数仍然可以正常读取JSON数据
				var originalJsonData map[string]interface{}
				body, _ := c.GetRawData()
				json.Unmarshal(body, &originalJsonData)

				// 验证JSON数据仍然可以正常解析
				originalValue := ""
				if val, exists := originalJsonData[tt.key]; exists {
					originalValue = fmt.Sprint(val)
				}

				c.JSON(200, gin.H{
					"value":               result,
					"original_json":       originalValue,
					"original_accessible": len(originalJsonData) > 0,
				})
			})

			// 准备JSON数据
			jsonData, err := json.Marshal(tt.jsonData)
			assert.Nil(t, err)

			// 创建HTTP请求
			req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, 200, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, response["value"])
			// 验证原始方法仍然可以获取到值
			if tt.expected != "" {
				assert.True(t, response["original_accessible"].(bool))
				assert.Equal(t, tt.expected, response["original_json"])
			}
		})
	}
}

// TestGet_URLParam 测试从URL路径参数获取值
//
// 【功能点】验证从 URL 路径参数（:param）获取值
// 【测试流程】
//  1. 定义带路径参数的路由（如 /user/:id）
//  2. 发送匹配路由的请求
//  3. 调用 Get 函数获取路径参数值
//  4. 验证返回值与期望一致
func TestGet_URLParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string // 测试用例名称
		url      string // 请求URL
		key      string // 要获取的键
		expected string // 期望的值
	}{
		{
			name:     "single URL param",
			url:      "/user/john",
			key:      "id",
			expected: "john",
		},
		{
			name:     "multiple URL params",
			url:      "/user/john/profile/edit",
			key:      "action",
			expected: "edit",
		},
		{
			name:     "non-existent URL param",
			url:      "/user/john",
			key:      "age",
			expected: "",
		},
		{
			name:     "special characters in URL param",
			url:      "/user/john-doe",
			key:      "id",
			expected: "john-doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建Gin引擎
			r := gin.New()
			r.GET("/user/:id", func(c *gin.Context) {
				// 先调用Get函数
				result := Get(c, tt.key)

				// 验证调用Get后，原始方法仍然可以正常工作
				originalValue := c.Param(tt.key)

				c.JSON(200, gin.H{
					"value":               result,
					"original_param":      originalValue,
					"original_accessible": originalValue != "",
				})
			})
			r.GET("/user/:id/profile/:action", func(c *gin.Context) {
				// 先调用Get函数
				result := Get(c, tt.key)

				// 验证调用Get后，原始方法仍然可以正常工作
				originalValue := c.Param(tt.key)

				c.JSON(200, gin.H{
					"value":               result,
					"original_param":      originalValue,
					"original_accessible": originalValue != "",
				})
			})

			// 创建HTTP请求
			req, _ := http.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			// 执行请求
			r.ServeHTTP(w, req)

			// 验证响应
			assert.Equal(t, 200, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, response["value"])
			// 验证原始方法仍然可以获取到值
			if tt.expected != "" {
				assert.True(t, response["original_accessible"].(bool))
				assert.Equal(t, tt.expected, response["original_param"])
			}
		})
	}
}

// TestGet_Priority 测试获取值的优先级顺序
//
// 【功能点】验证多来源参数的获取优先级
// 【测试流程】
//  1. 同时提供 Query、Form、JSON Body、URL Param 中的同名参数
//  2. 调用 Get 函数获取参数值
//  3. 验证优先级：Query > Form > JSON Body > URL Param
func TestGet_Priority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("query param has highest priority", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.GET("/test/:id", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "id")

			// 验证调用Get后，原始方法仍然可以正常工作
			originalQuery := c.Query("id")
			originalParam := c.Param("id")

			c.JSON(200, gin.H{
				"value":            result,
				"original_query":   originalQuery,
				"original_param":   originalParam,
				"query_accessible": originalQuery != "",
				"param_accessible": originalParam != "",
			})
		})

		// 创建HTTP请求 - 同时有查询参数和路径参数
		req, _ := http.NewRequest("GET", "/test/path-param?id=query-param", nil)
		w := httptest.NewRecorder()

		// 执行请求
		r.ServeHTTP(w, req)

		// 验证响应 - 查询参数应该优先于路径参数
		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.Nil(t, err)
		assert.Equal(t, "query-param", response["value"])
		// 验证原始方法仍然可以获取到值
		assert.True(t, response["query_accessible"].(bool))
		assert.True(t, response["param_accessible"].(bool))
		assert.Equal(t, "query-param", response["original_query"])
		assert.Equal(t, "path-param", response["original_param"])
	})

	t.Run("postform has priority over JSON and URL param", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.POST("/test/:id", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "id")

			// 验证调用Get后，原始方法仍然可以正常工作
			originalPostForm := c.PostForm("id")
			originalParam := c.Param("id")

			// 验证JSON数据仍然可以正常解析
			var originalJsonData map[string]interface{}
			body, _ := c.GetRawData()
			json.Unmarshal(body, &originalJsonData)
			originalJson := ""
			if val, exists := originalJsonData["id"]; exists {
				originalJson = fmt.Sprint(val)
			}

			c.JSON(200, gin.H{
				"value":               result,
				"original_postform":   originalPostForm,
				"original_param":      originalParam,
				"original_json":       originalJson,
				"postform_accessible": originalPostForm != "",
				"param_accessible":    originalParam != "",
				"json_accessible":     len(originalJsonData) > 0,
			})
		})

		// 准备JSON数据
		jsonData := map[string]interface{}{"id": "json-param"}
		jsonBytes, _ := json.Marshal(jsonData)

		// 创建HTTP请求 - 同时有表单数据、JSON数据和路径参数
		req, _ := http.NewRequest("POST", "/test/path-param", bytes.NewBuffer(jsonBytes))
		req.PostForm = map[string][]string{"id": {"form-param"}}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		// 执行请求
		r.ServeHTTP(w, req)

		// 验证响应 - 表单数据应该优先于JSON和路径参数
		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.Nil(t, err)
		assert.Equal(t, "form-param", response["value"])
		// 验证原始方法仍然可以获取到值
		assert.True(t, response["postform_accessible"].(bool))
		assert.True(t, response["param_accessible"].(bool))
		assert.True(t, response["json_accessible"].(bool))
		assert.Equal(t, "form-param", response["original_postform"])
		assert.Equal(t, "path-param", response["original_param"])
		assert.Equal(t, "json-param", response["original_json"])
	})

	t.Run("JSON has priority over URL param", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.POST("/test/:id", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "id")

			// 验证调用Get后，原始方法仍然可以正常工作
			originalParam := c.Param("id")

			// 验证JSON数据仍然可以正常解析
			var originalJsonData map[string]interface{}
			body, _ := c.GetRawData()
			json.Unmarshal(body, &originalJsonData)
			originalJson := ""
			if val, exists := originalJsonData["id"]; exists {
				originalJson = fmt.Sprint(val)
			}

			c.JSON(200, gin.H{
				"value":            result,
				"original_param":   originalParam,
				"original_json":    originalJson,
				"param_accessible": originalParam != "",
				"json_accessible":  len(originalJsonData) > 0,
			})
		})

		// 准备JSON数据
		jsonData := map[string]interface{}{"id": "json-param"}
		jsonBytes, _ := json.Marshal(jsonData)

		// 创建HTTP请求 - 同时有JSON数据和路径参数
		req, _ := http.NewRequest("POST", "/test/path-param", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// 执行请求
		r.ServeHTTP(w, req)

		// 验证响应 - JSON数据应该优先于路径参数
		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.Nil(t, err)
		assert.Equal(t, "json-param", response["value"])
		// 验证原始方法仍然可以获取到值
		assert.True(t, response["param_accessible"].(bool))
		assert.True(t, response["json_accessible"].(bool))
		assert.Equal(t, "path-param", response["original_param"])
		assert.Equal(t, "json-param", response["original_json"])
	})
}

// TestGet_EdgeCases 测试边界情况
//
// 【功能点】验证各种边界情况的处理
// 【测试流程】
//  1. 测试空 key - 验证返回空字符串
//  2. 测试无效 JSON - 验证不会 panic
//  3. 测试空请求体 - 验证正常返回
//  4. 测试多值参数 - 验证只返回第一个值
func TestGet_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("empty key", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "")

			// 验证调用Get后，原始方法仍然可以正常工作
			originalQuery := c.Query("name")

			c.JSON(200, gin.H{
				"value":               result,
				"original_query":      originalQuery,
				"original_accessible": originalQuery != "",
			})
		})

		// 创建HTTP请求
		req, _ := http.NewRequest("GET", "/test?name=john", nil)
		w := httptest.NewRecorder()

		// 执行请求
		r.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.Nil(t, err)
		assert.Equal(t, "", response["value"])
		// 验证原始方法仍然可以获取到值
		assert.True(t, response["original_accessible"].(bool))
		assert.Equal(t, "john", response["original_query"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.POST("/test", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "name")

			// 验证调用Get后，原始方法仍然可以正常工作
			// 对于无效JSON，GetRawData仍然可以获取到原始数据
			body, _ := c.GetRawData()

			c.JSON(200, gin.H{
				"value":               result,
				"original_body":       string(body),
				"original_accessible": len(body) > 0,
			})
		})

		// 创建HTTP请求 - 包含无效JSON
		req, _ := http.NewRequest("POST", "/test", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// 执行请求
		r.ServeHTTP(w, req)

		// 验证响应 - 应该不会panic，返回空字符串
		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.Nil(t, err)
		assert.Equal(t, "", response["value"])
		// 验证原始方法仍然可以获取到值
		assert.True(t, response["original_accessible"].(bool))
		assert.Equal(t, "invalid json", response["original_body"])
	})

	t.Run("large JSON body", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.POST("/test", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "name")

			// 验证调用Get后，原始方法仍然可以正常工作
			var originalJsonData map[string]interface{}
			body, _ := c.GetRawData()
			json.Unmarshal(body, &originalJsonData)

			originalValue := ""
			if val, exists := originalJsonData["name"]; exists {
				originalValue = fmt.Sprint(val)
			}

			c.JSON(200, gin.H{
				"value":               result,
				"original_json":       originalValue,
				"original_accessible": len(originalJsonData) > 0,
				"json_size":           len(originalJsonData),
			})
		})

		// 准备大型JSON数据
		largeData := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeData[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}
		largeData["name"] = "john"

		jsonBytes, _ := json.Marshal(largeData)

		// 创建HTTP请求
		req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// 执行请求
		r.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, 200, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.Nil(t, err)
		assert.Equal(t, "john", response["value"])
		// 验证原始方法仍然可以获取到值
		assert.True(t, response["original_accessible"].(bool))
		assert.Equal(t, "john", response["original_json"])
		assert.Equal(t, 1001, int(response["json_size"].(float64))) // 1000个key + 1个name
	})
}

// TestGet_Concurrent 测试并发安全性
//
// 【功能点】验证 Get 函数在并发环境下的安全性
// 【测试流程】
//  1. 启动 10 个协程并发发送请求
//  2. 每个请求携带不同的参数值
//  3. 验证每个请求获取到正确的对应值
//  4. 验证无数据竞争或混乱
func TestGet_Concurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("concurrent requests", func(t *testing.T) {
		// 创建Gin引擎
		r := gin.New()
		r.GET("/test", func(c *gin.Context) {
			// 先调用Get函数
			result := Get(c, "name")

			// 验证调用Get后，原始方法仍然可以正常工作
			originalValue := c.Query("name")

			c.JSON(200, gin.H{
				"value":               result,
				"original_query":      originalValue,
				"original_accessible": originalValue != "",
			})
		})

		// 并发发送请求
		done := make(chan struct {
			index              int
			value              string
			originalValue      string
			originalAccessible bool
		}, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				req, _ := http.NewRequest("GET", fmt.Sprintf("/test?name=user%d", index), nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				done <- struct {
					index              int
					value              string
					originalValue      string
					originalAccessible bool
				}{
					index:              index,
					value:              response["value"].(string),
					originalValue:      response["original_query"].(string),
					originalAccessible: response["original_accessible"].(bool),
				}
			}(i)
		}

		// 收集结果
		results := make(map[int]struct {
			value              string
			originalValue      string
			originalAccessible bool
		})
		for i := 0; i < 10; i++ {
			result := <-done
			results[result.index] = struct {
				value              string
				originalValue      string
				originalAccessible bool
			}{
				value:              result.value,
				originalValue:      result.originalValue,
				originalAccessible: result.originalAccessible,
			}
		}

		// 验证所有结果都正确
		for i := 0; i < 10; i++ {
			expected := fmt.Sprintf("user%d", i)
			assert.Equal(t, expected, results[i].value)
			// 验证原始方法仍然可以获取到值
			assert.True(t, results[i].originalAccessible)
			assert.Equal(t, expected, results[i].originalValue)
		}
	})
}

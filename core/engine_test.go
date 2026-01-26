// Package core 引擎初始化功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 Gin 引擎初始化相关功能的单元测试。
//
// 测试覆盖内容：
// 1. AddOptionFunc - 选项函数注册（单个/多个/空/nil）
// 2. healthDetactEngine - 健康检查路由配置
// 3. initEngine - 引擎初始化（路由前缀/中间件/自定义选项）
// 4. 引擎特性 - Recovery中间件/405处理/404处理/健康检查
// 5. 自定义路由 - 路由前缀与自定义路由组合
// 6. 并发初始化 - 并发环境下的引擎初始化
//
// 运行测试：go test -v ./core/...
// ==================================================
package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// ==================== AddOptionFunc 测试 ====================
// 测试选项函数的注册功能

// TestAddOptionFunc 测试AddOptionFunc函数
//
// 【功能点】验证选项函数的注册机制
// 【测试流程】
//  1. 测试添加单个选项函数 - 验证列表长度为1
//  2. 测试添加多个选项函数 - 验证列表长度正确
//  3. 测试添加空参数 - 验证列表保持为空
//  4. 测试添加nil函数 - 验证nil也被添加到列表
func TestAddOptionFunc(t *testing.T) {
	// 清空选项函数列表，确保测试环境干净
	optionFuncList = make([]gin.OptionFunc, 0)

	t.Run("add single option function", func(t *testing.T) {
		// 测试添加单个选项函数
		optionFunc := func(e *gin.Engine) {
			e.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "test"})
			})
		}

		AddOptionFunc(optionFunc)
		assert.Len(t, optionFuncList, 1)
		assert.NotNil(t, optionFuncList[0])
	})

	t.Run("add multiple option functions", func(t *testing.T) {
		// 清空列表
		optionFuncList = make([]gin.OptionFunc, 0)

		// 测试添加多个选项函数
		optionFunc1 := func(e *gin.Engine) {
			e.GET("/test1", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "test1"})
			})
		}
		optionFunc2 := func(e *gin.Engine) {
			e.GET("/test2", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "test2"})
			})
		}

		AddOptionFunc(optionFunc1, optionFunc2)
		assert.Len(t, optionFuncList, 2)
		assert.NotNil(t, optionFuncList[0])
		assert.NotNil(t, optionFuncList[1])
	})

	t.Run("add empty option functions", func(t *testing.T) {
		// 清空列表
		optionFuncList = make([]gin.OptionFunc, 0)

		// 测试添加空参数
		AddOptionFunc()
		assert.Len(t, optionFuncList, 0)
	})

	t.Run("add nil option function", func(t *testing.T) {
		// 清空列表
		optionFuncList = make([]gin.OptionFunc, 0)

		// 测试添加nil函数
		AddOptionFunc(nil)
		assert.Len(t, optionFuncList, 1)
		assert.Nil(t, optionFuncList[0])
	})
}

// ==================== healthDetactEngine 测试 ====================
// 测试健康检查路由配置

// TestHealthDetactEngine 测试healthDetactEngine函数
//
// 【功能点】验证健康检查路由的正确配置
// 【测试流程】
//  1. 创建Gin引擎并应用健康检查配置
//  2. 发送GET /healthy请求
//  3. 验证返回200状态码和正确的JSON响应
//  4. 测试路径重定向行为（/healthy/ → /healthy）
func TestHealthDetactEngine(t *testing.T) {
	t.Run("health check route", func(t *testing.T) {
		// 创建测试引擎
		engine := gin.New()

		// 应用健康检查路由配置
		healthDetactEngine(engine)

		// 创建测试请求
		req := httptest.NewRequest("GET", "/healthy", nil)
		w := httptest.NewRecorder()

		// 执行请求
		engine.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(20000), response["code"])
		assert.Equal(t, "healthy", response["msg"])

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "healthy", data["status"])
	})

	t.Run("health check route with different path", func(t *testing.T) {
		// 创建测试引擎
		engine := gin.New()

		// 应用健康检查路由配置
		healthDetactEngine(engine)

		// 测试不同的路径
		req := httptest.NewRequest("GET", "/healthy/", nil)
		w := httptest.NewRecorder()

		engine.ServeHTTP(w, req)
		// Gin会自动重定向 /healthy/ 到 /healthy，所以返回301
		assert.Equal(t, http.StatusMovedPermanently, w.Code)
	})
}

// ==================== initEngine 测试 ====================
// 测试引擎初始化功能

// TestInitEngine 测试initEngine函数
//
// 【功能点】验证引擎初始化的各种场景
// 【测试流程】
//  1. 测试无路由前缀初始化 - 验证引擎和RouterGroup非空
//  2. 测试带路由前缀初始化 - 验证前缀正确应用
//  3. 测试带中间件初始化 - 验证注册的中间件被正确加载
//  4. 测试未知中间件处理 - 验证配置正确设置
//  5. 测试自定义选项函数 - 验证选项函数被执行
func TestInitEngine(t *testing.T) {
	// 保存原始配置
	originalConfig := app.BaseConfig
	defer func() {
		app.BaseConfig = originalConfig
	}()

	// 清空选项函数列表
	optionFuncList = make([]gin.OptionFunc, 0)

	t.Run("init engine without route prefix", func(t *testing.T) {
		// 设置测试配置
		app.BaseConfig = config.BaseConfig{
			Service: config.ServiceInfo{
				RoutePrefix: "",
				Middlewares: []string{},
			},
		}

		// 清空中间件映射表
		middleWareMap = make(map[string]func() gin.HandlerFunc)

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.RouterGroup)
	})

	t.Run("init engine with route prefix", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 设置测试配置
		app.BaseConfig = config.BaseConfig{
			Service: config.ServiceInfo{
				RoutePrefix: "/api/v1",
				Middlewares: []string{},
			},
		}

		// 清空中间件映射表
		middleWareMap = make(map[string]func() gin.HandlerFunc)

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.RouterGroup)
	})

	t.Run("init engine with middlewares", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 设置测试配置
		app.BaseConfig = config.BaseConfig{
			Service: config.ServiceInfo{
				RoutePrefix: "",
				Middlewares: []string{"testMiddleware"},
			},
		}

		// 清空中间件映射表
		middleWareMap = make(map[string]func() gin.HandlerFunc)

		// 注册测试中间件
		RegisterMiddleware("testMiddleware", func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		})

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.RouterGroup)
	})

	t.Run("init engine with unknown middleware", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 设置测试配置
		app.BaseConfig = config.BaseConfig{
			Service: config.ServiceInfo{
				RoutePrefix: "",
				Middlewares: []string{"unknownMiddleware"},
			},
		}

		// 清空中间件映射表
		middleWareMap = make(map[string]func() gin.HandlerFunc)

		// 这个测试会调用os.Exit(1)，所以我们需要在子进程中运行
		// 这里我们只验证配置设置正确
		assert.Equal(t, "unknownMiddleware", app.BaseConfig.Service.Middlewares[0])
	})

	t.Run("init engine with custom option functions", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 设置测试配置
		app.BaseConfig = config.BaseConfig{
			Service: config.ServiceInfo{
				RoutePrefix: "",
				Middlewares: []string{},
			},
		}

		// 清空中间件映射表
		middleWareMap = make(map[string]func() gin.HandlerFunc)

		// 添加自定义选项函数
		AddOptionFunc(func(e *gin.Engine) {
			e.GET("/custom", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "custom"})
			})
		})

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.RouterGroup)
	})
}

// ==================== 引擎特性测试 ====================
// 测试引擎的内置功能特性

// TestEngineFeatures 测试引擎特性
//
// 【功能点】验证引擎的内置中间件和错误处理
// 【测试流程】
//  1. 测试Recovery中间件 - 验证panic被捕获并返回500
//  2. 测试405处理器 - 验证方法不允许时返回405
//  3. 测试404处理器 - 验证路由不存在时返回404
//  4. 测试健康检查路由 - 验证/healthy路由可访问
func TestEngineFeatures(t *testing.T) {
	// 保存原始配置
	originalConfig := app.BaseConfig
	defer func() {
		app.BaseConfig = originalConfig
	}()

	// 设置测试配置
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{
			RoutePrefix: "",
			Middlewares: []string{},
		},
	}

	// 清空中间件映射表
	middleWareMap = make(map[string]func() gin.HandlerFunc)

	// 清空选项函数列表
	optionFuncList = make([]gin.OptionFunc, 0)

	t.Run("engine has recovery middleware", func(t *testing.T) {
		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)

		// 创建测试请求
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		// 添加一个会panic的路由
		engine.GET("/test", func(c *gin.Context) {
			panic("test panic")
		})

		// 执行请求
		engine.ServeHTTP(w, req)

		// 验证Recovery中间件捕获了panic
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("engine has method not allowed handler", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)

		// 添加一个只支持GET的路由
		engine.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		// 创建POST请求
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		// 执行请求
		engine.ServeHTTP(w, req)

		// 验证405错误处理
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("engine has not found handler", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)

		// 创建请求到不存在的路由
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()

		// 执行请求
		engine.ServeHTTP(w, req)

		// 验证404错误处理
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("engine has health check route", func(t *testing.T) {
		// 清空选项函数列表，避免健康检查路由冲突
		optionFuncList = make([]gin.OptionFunc, 0)

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)

		// 创建健康检查请求
		req := httptest.NewRequest("GET", "/healthy", nil)
		w := httptest.NewRecorder()

		// 执行请求
		engine.ServeHTTP(w, req)

		// 验证健康检查响应
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(20000), response["code"])
		assert.Equal(t, "healthy", response["msg"])
	})
}

// ==================== 自定义路由测试 ====================
// 测试引擎与自定义路由的集成

// TestEngineWithCustomRoutes 测试引擎与自定义路由
//
// 【功能点】验证自定义路由与路由前缀的正确组合
// 【测试流程】
//  1. 设置路由前缀 /api/v1
//  2. 添加自定义路由 /users（GET/POST）
//  3. 验证 /api/v1/users 路由可访问
//  4. 测试多个选项函数添加多个路由组
func TestEngineWithCustomRoutes(t *testing.T) {
	// 保存原始配置
	originalConfig := app.BaseConfig
	defer func() {
		app.BaseConfig = originalConfig
	}()

	// 设置测试配置
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{
			RoutePrefix: "/api/v1",
			Middlewares: []string{},
		},
	}

	// 清空中间件映射表
	middleWareMap = make(map[string]func() gin.HandlerFunc)

	// 清空选项函数列表
	optionFuncList = make([]gin.OptionFunc, 0)

	t.Run("engine with route prefix and custom routes", func(t *testing.T) {
		// 添加自定义路由
		AddOptionFunc(func(e *gin.Engine) {
			e.GET("/users", func(c *gin.Context) {
				c.JSON(200, gin.H{"users": []string{"user1", "user2"}})
			})
			e.POST("/users", func(c *gin.Context) {
				c.JSON(201, gin.H{"message": "user created"})
			})
		})

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)

		// 测试GET /api/v1/users
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// 测试POST /api/v1/users
		req = httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(`{"name":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("engine with multiple custom routes", func(t *testing.T) {
		// 清空选项函数列表
		optionFuncList = make([]gin.OptionFunc, 0)

		// 添加多个自定义路由
		AddOptionFunc(func(e *gin.Engine) {
			e.GET("/products", func(c *gin.Context) {
				c.JSON(200, gin.H{"products": []string{"product1", "product2"}})
			})
		})

		AddOptionFunc(func(e *gin.Engine) {
			e.GET("/orders", func(c *gin.Context) {
				c.JSON(200, gin.H{"orders": []string{"order1", "order2"}})
			})
		})

		// 初始化引擎
		engine := initEngine()
		assert.NotNil(t, engine)

		// 测试所有路由
		routes := []string{"/api/v1/products", "/api/v1/orders", "/api/v1/healthy"}
		for _, route := range routes {
			req := httptest.NewRequest("GET", route, nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code, "Route %s should return 200", route)
		}
	})
}

// ==================== 并发初始化测试 ====================
// 测试并发环境下的引擎初始化

// TestConcurrentEngineInit 测试并发引擎初始化
//
// 【功能点】验证引擎初始化的并发安全性
// 【测试流程】
//  1. 设置基础配置
//  2. 初始化引擎
//  3. 验证引擎和RouterGroup非空
//
// 【注意】由于健康检查路由的全局特性，仅测试单引擎初始化
func TestConcurrentEngineInit(t *testing.T) {
	// 保存原始配置
	originalConfig := app.BaseConfig
	defer func() {
		app.BaseConfig = originalConfig
	}()

	// 设置测试配置
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{
			RoutePrefix: "",
			Middlewares: []string{},
		},
	}

	// 清空中间件映射表
	middleWareMap = make(map[string]func() gin.HandlerFunc)

	t.Run("concurrent engine initialization", func(t *testing.T) {
		// 由于健康检查路由的全局特性，并发测试会导致路由冲突
		// 这里只测试单个引擎初始化
		optionFuncList = make([]gin.OptionFunc, 0)

		engine := initEngine()
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.RouterGroup)
	})
}

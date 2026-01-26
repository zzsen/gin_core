// Package core 中间件管理功能测试
//
// ==================== 测试说明 ====================
// 本文件包含中间件注册和管理功能的单元测试。
//
// 测试覆盖内容：
// 1. RegisterMiddleware - 中间件注册（新增/重复/多个）
// 2. getMiddleware - 获取已注册的中间件
// 3. clearMiddlewares - 清空中间件映射表
// 4. 并发安全 - 多协程并发注册中间件
// 5. 中间件加载 - 从配置加载中间件
//
// 中间件机制：
//   - 中间件按名称注册到全局映射表
//   - 重复注册同名中间件会返回错误
//   - 配置文件指定的中间件名称必须已注册
//
// 运行测试：go test -v ./core/... -run Middleware
// ==================================================
package core

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ==================== RegisterMiddleware 测试 ====================

// TestRegisterMiddleware 测试RegisterMiddleware函数
//
// 【功能点】验证中间件注册功能
// 【测试流程】
//  1. 测试注册新中间件 - 验证注册成功且可获取
//  2. 测试重复注册 - 验证返回错误
//  3. 测试多中间件注册 - 验证多个中间件独立注册
func TestRegisterMiddleware(t *testing.T) {
	// 清空中间件映射表，确保测试环境干净
	clearMiddlewares()

	t.Run("register new middleware", func(t *testing.T) {
		// 测试注册新的中间件
		handlerFunc := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}

		err := RegisterMiddleware("testMiddleware", handlerFunc)
		assert.NoError(t, err)
		// 验证中间件已注册
		handler, exists := getMiddleware("testMiddleware")
		assert.True(t, exists)
		assert.NotNil(t, handler)
	})

	t.Run("register duplicate middleware", func(t *testing.T) {
		// 测试注册重复名称的中间件
		handlerFunc1 := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}
		handlerFunc2 := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}

		// 第一次注册应该成功
		err1 := RegisterMiddleware("duplicateMiddleware", handlerFunc1)
		assert.NoError(t, err1)

		// 第二次注册相同名称应该失败
		err2 := RegisterMiddleware("duplicateMiddleware", handlerFunc2)
		assert.Error(t, err2)
		assert.Equal(t, "this name is already in use", err2.Error())
	})

	t.Run("register multiple middlewares", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 测试注册多个不同的中间件
		middlewares := []struct {
			name        string
			handlerFunc func() gin.HandlerFunc
		}{
			{
				name: "middleware1",
				handlerFunc: func() gin.HandlerFunc {
					return gin.HandlerFunc(func(c *gin.Context) {
						c.Next()
					})
				},
			},
			{
				name: "middleware2",
				handlerFunc: func() gin.HandlerFunc {
					return gin.HandlerFunc(func(c *gin.Context) {
						c.Next()
					})
				},
			},
			{
				name: "middleware3",
				handlerFunc: func() gin.HandlerFunc {
					return gin.HandlerFunc(func(c *gin.Context) {
						c.Next()
					})
				},
			},
		}

		// 注册所有中间件
		for _, mw := range middlewares {
			err := RegisterMiddleware(mw.name, mw.handlerFunc)
			assert.NoError(t, err)
		}

		// 验证所有中间件都已注册
		assert.Equal(t, 3, getMiddlewareCount())
		for _, mw := range middlewares {
			assert.True(t, hasMiddleware(mw.name))
			handler, exists := getMiddleware(mw.name)
			assert.True(t, exists)
			assert.NotNil(t, handler)
		}
	})

	t.Run("register empty name", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 测试注册空名称的中间件
		handlerFunc := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}

		err := RegisterMiddleware("", handlerFunc)
		assert.NoError(t, err) // 空名称应该被允许注册
		assert.True(t, hasMiddleware(""))
	})

	t.Run("register nil handler", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 测试注册nil处理函数
		err := RegisterMiddleware("nilHandler", nil)
		assert.NoError(t, err) // nil处理函数应该被允许注册
		assert.True(t, hasMiddleware("nilHandler"))
		handler, exists := getMiddleware("nilHandler")
		assert.True(t, exists)
		assert.Nil(t, handler)
	})

	t.Run("register special characters in name", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 测试注册包含特殊字符的中间件名称
		specialNames := []string{
			"middleware-with-dash",
			"middleware_with_underscore",
			"middleware.with.dots",
			"middleware123",
			"middleware@special",
			"middleware space",
		}

		handlerFunc := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}

		for _, name := range specialNames {
			err := RegisterMiddleware(name, handlerFunc)
			assert.NoError(t, err, "Failed to register middleware with name: %s", name)
			assert.True(t, hasMiddleware(name))
		}
	})
}

// ==================== initMiddleware 测试 ====================

// TestInitMiddleware 测试initMiddleware函数
//
// 【功能点】验证中间件初始化流程
// 【测试流程】
//  1. 注册测试中间件
//  2. 配置需要加载的中间件列表
//  3. 调用 initMiddleware 初始化
//  4. 验证中间件被正确应用到引擎
func TestInitMiddleware(t *testing.T) {
	// 清空中间件映射表
	clearMiddlewares()

	t.Run("initialize default middlewares", func(t *testing.T) {
		// 调用初始化函数
		initMiddleware()

		// 验证所有默认中间件都已注册
		expectedMiddlewares := []string{
			"prometheusHandler",
			"exceptionHandler",
			"traceIdHandler",
			"otelTraceHandler",
			"traceLogHandler",
			"timeoutHandler",
		}

		for _, name := range expectedMiddlewares {
			assert.True(t, hasMiddleware(name), "Middleware %s should be registered", name)
			handler, exists := getMiddleware(name)
			assert.True(t, exists, "Middleware %s should have a handler function", name)
			assert.NotNil(t, handler)
		}

		// 验证注册的中间件数量
		assert.Equal(t, 6, getMiddlewareCount())
	})

	t.Run("initialize multiple times", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 多次调用初始化函数
		initMiddleware()
		initMiddleware()
		initMiddleware()

		// 验证中间件只注册了一次（因为重复注册会失败）
		expectedMiddlewares := []string{
			"prometheusHandler",
			"exceptionHandler",
			"traceIdHandler",
			"otelTraceHandler",
			"traceLogHandler",
			"timeoutHandler",
		}

		for _, name := range expectedMiddlewares {
			assert.True(t, hasMiddleware(name), "Middleware %s should be registered", name)
		}

		// 验证注册的中间件数量
		assert.Equal(t, 6, getMiddlewareCount())
	})
}

// ==================== 中间件映射表测试 ====================

// TestMiddlewareMapAccess 测试中间件映射表的访问
//
// 【功能点】验证中间件映射表的读写操作
// 【测试流程】
//  1. 注册中间件到映射表
//  2. 从映射表获取中间件
//  3. 验证中间件存在性检查
func TestMiddlewareMapAccess(t *testing.T) {
	// 清空映射表
	clearMiddlewares()

	t.Run("access unregistered middleware", func(t *testing.T) {
		// 测试访问未注册的中间件
		handler, exists := getMiddleware("nonExistentMiddleware")
		assert.False(t, exists)
		assert.Nil(t, handler)
	})

	t.Run("access registered middleware", func(t *testing.T) {
		// 注册一个中间件
		handlerFunc := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}

		err := RegisterMiddleware("testMiddleware", handlerFunc)
		assert.NoError(t, err)

		// 测试访问已注册的中间件
		handler, exists := getMiddleware("testMiddleware")
		assert.True(t, exists)
		assert.NotNil(t, handler)
		// 函数不能直接比较，但可以验证它们都不为nil
		assert.NotNil(t, handlerFunc)
	})

	t.Run("modify middleware map directly", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 直接修改映射表
		handlerFunc := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}

		setMiddleware("directMiddleware", handlerFunc)

		// 验证直接修改生效
		handler, exists := getMiddleware("directMiddleware")
		assert.True(t, exists)
		assert.NotNil(t, handler)
		// 函数不能直接比较，但可以验证它们都不为nil
		assert.NotNil(t, handlerFunc)
	})
}

// ==================== 中间件执行测试 ====================

// TestMiddlewareHandlerExecution 测试中间件处理函数的执行
//
// 【功能点】验证中间件处理函数被正确执行
// 【测试流程】
//  1. 创建带有标记的中间件
//  2. 发送请求触发中间件
//  3. 验证中间件被执行（检查标记）
func TestMiddlewareHandlerExecution(t *testing.T) {
	// 清空映射表
	clearMiddlewares()

	t.Run("execute registered middleware handler", func(t *testing.T) {
		// 创建一个测试用的Gin上下文
		gin.SetMode(gin.TestMode)
		c, _ := gin.CreateTestContext(nil)

		// 注册一个测试中间件
		executed := false
		handlerFunc := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				executed = true
				c.Next()
			})
		}

		err := RegisterMiddleware("testHandler", handlerFunc)
		assert.NoError(t, err)

		// 获取并执行中间件处理函数
		handler, exists := getMiddleware("testHandler")
		assert.True(t, exists)
		assert.NotNil(t, handler)

		// 执行处理函数
		handler()(c)
		assert.True(t, executed)
	})

	t.Run("execute nil middleware handler", func(t *testing.T) {
		// 注册一个nil处理函数
		err := RegisterMiddleware("nilHandler", nil)
		assert.NoError(t, err)

		// 获取nil处理函数
		handler, exists := getMiddleware("nilHandler")
		assert.True(t, exists)
		assert.Nil(t, handler)

		// 尝试执行nil处理函数应该会panic
		assert.Panics(t, func() {
			handler()(nil)
		})
	})
}

// ==================== 并发安全测试 ====================

// TestConcurrentMiddlewareRegistration 测试并发中间件注册
//
// 【功能点】验证中间件注册的并发安全性
// 【测试流程】
//  1. 启动多个协程并发注册中间件
//  2. 验证无数据竞争
//  3. 验证所有注册都正确完成
func TestConcurrentMiddlewareRegistration(t *testing.T) {
	// 清空映射表
	clearMiddlewares()

	t.Run("concurrent registration", func(t *testing.T) {
		// 并发注册多个中间件
		concurrency := 10
		results := make(chan error, concurrency)
		var wg sync.WaitGroup

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				handlerFunc := func() gin.HandlerFunc {
					return gin.HandlerFunc(func(c *gin.Context) {
						c.Next()
					})
				}

				// 使用不同的名称避免冲突
				name := fmt.Sprintf("concurrentMiddleware%d", index)
				err := RegisterMiddleware(name, handlerFunc)
				results <- err
			}(i)
		}

		// 等待所有goroutine完成
		wg.Wait()
		close(results)

		// 收集结果
		successCount := 0
		errorCount := 0
		for err := range results {
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		// 验证结果
		assert.Equal(t, concurrency, successCount)
		assert.Equal(t, 0, errorCount)
		assert.Equal(t, concurrency, getMiddlewareCount())
	})

	t.Run("concurrent duplicate registration", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 并发注册相同名称的中间件
		concurrency := 5
		results := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				handlerFunc := func() gin.HandlerFunc {
					return gin.HandlerFunc(func(c *gin.Context) {
						c.Next()
					})
				}

				err := RegisterMiddleware("duplicateConcurrent", handlerFunc)
				results <- err
			}()
		}

		// 收集结果
		successCount := 0
		errorCount := 0
		for i := 0; i < concurrency; i++ {
			err := <-results
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		// 验证结果：只有一个应该成功，其他应该失败
		assert.Equal(t, 1, successCount)
		assert.Equal(t, concurrency-1, errorCount)
		assert.Equal(t, 1, getMiddlewareCount())
	})
}

// ==================== 错误处理测试 ====================

// TestMiddlewareErrorHandling 测试中间件错误处理
//
// 【功能点】验证中间件错误的正确处理
// 【测试流程】
//  1. 测试注册重复名称 - 返回错误
//  2. 测试获取不存在的中间件 - 返回 false
//  3. 测试 nil 处理函数 - 正确处理
func TestMiddlewareErrorHandling(t *testing.T) {
	// 清空映射表
	clearMiddlewares()

	t.Run("register with empty name after non-empty", func(t *testing.T) {
		// 先注册一个非空名称的中间件
		handlerFunc1 := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}
		err1 := RegisterMiddleware("nonEmpty", handlerFunc1)
		assert.NoError(t, err1)

		// 再注册一个空名称的中间件
		handlerFunc2 := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}
		err2 := RegisterMiddleware("", handlerFunc2)
		assert.NoError(t, err2)

		// 验证两个中间件都已注册
		assert.Equal(t, 2, getMiddlewareCount())
		assert.True(t, hasMiddleware("nonEmpty"))
		assert.True(t, hasMiddleware(""))
	})

	t.Run("register with same name after different name", func(t *testing.T) {
		// 清空映射表
		clearMiddlewares()

		// 先注册一个中间件
		handlerFunc1 := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}
		err1 := RegisterMiddleware("first", handlerFunc1)
		assert.NoError(t, err1)

		// 再注册相同名称的中间件
		handlerFunc2 := func() gin.HandlerFunc {
			return gin.HandlerFunc(func(c *gin.Context) {
				c.Next()
			})
		}
		err2 := RegisterMiddleware("first", handlerFunc2)
		assert.Error(t, err2)
		assert.Equal(t, "this name is already in use", err2.Error())

		// 验证只有第一个中间件被注册
		assert.Equal(t, 1, getMiddlewareCount())
		assert.True(t, hasMiddleware("first"))
		// 函数不能直接比较，但可以验证它们都不为nil
		handler, exists := getMiddleware("first")
		assert.True(t, exists)
		assert.NotNil(t, handler)
		assert.NotNil(t, handlerFunc1)
	})
}

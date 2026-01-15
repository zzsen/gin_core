package core

import (
	"errors"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/middleware"
)

// middleWareMap 中间件注册映射表
// 存储中间件名称与对应处理函数的映射关系
// key: 中间件名称（字符串），用于在配置文件中引用
// value: 中间件工厂函数，返回具体的Gin中间件处理函数
// 这种设计模式允许通过配置文件动态启用/禁用中间件，提高了系统的灵活性
var middleWareMap = make(map[string]func() gin.HandlerFunc)

// middlewareMutex 保护middleWareMap并发访问的互斥锁
var middlewareMutex sync.RWMutex

// getMiddleware 安全地获取中间件处理函数 (仅用于测试)
func getMiddleware(name string) (func() gin.HandlerFunc, bool) {
	middlewareMutex.RLock()
	defer middlewareMutex.RUnlock()
	handler, exists := middleWareMap[name]
	return handler, exists
}

// setMiddleware 安全地设置中间件处理函数（仅用于测试）
func setMiddleware(name string, handler func() gin.HandlerFunc) {
	middlewareMutex.Lock()
	defer middlewareMutex.Unlock()
	middleWareMap[name] = handler
}

// clearMiddlewares 清空中间件映射表（仅用于测试）
func clearMiddlewares() {
	middlewareMutex.Lock()
	defer middlewareMutex.Unlock()
	middleWareMap = make(map[string]func() gin.HandlerFunc)
}

// getMiddlewareCount 安全地获取中间件数量（仅用于测试）
func getMiddlewareCount() int {
	middlewareMutex.RLock()
	defer middlewareMutex.RUnlock()
	return len(middleWareMap)
}

// hasMiddleware 安全地检查中间件是否存在（仅用于测试）
func hasMiddleware(name string) bool {
	middlewareMutex.RLock()
	defer middlewareMutex.RUnlock()
	_, exists := middleWareMap[name]
	return exists
}

// RegisterMiddleware 注册中间件到映射表
// 将中间件处理函数注册到全局中间件映射表中，使其可以通过名称引用
// 这是中间件系统的核心注册函数，支持插件化的中间件管理
//
// 参数：
//   - name: 中间件名称，必须唯一，用于在配置文件中引用
//   - handlerFunc: 中间件工厂函数，返回gin.HandlerFunc类型的处理函数
//
// 返回值：
//   - error: 如果中间件名称已存在则返回错误，否则返回nil
//
// 使用示例：
//
//	err := RegisterMiddleware("cors", func() gin.HandlerFunc {
//	  return gin.HandlerFunc(func(c *gin.Context) {
//	    // CORS处理逻辑
//	    c.Next()
//	  })
//	})
func RegisterMiddleware(name string, handlerFunc func() gin.HandlerFunc) error {
	// 使用写锁保护并发访问
	middlewareMutex.Lock()
	defer middlewareMutex.Unlock()

	// 检查中间件名称是否已被使用，防止重复注册
	if _, ok := middleWareMap[name]; ok {
		return errors.New("this name is already in use")
	}
	// 将中间件注册到映射表
	middleWareMap[name] = handlerFunc
	return nil
}

// initMiddleware 初始化系统默认中间件
// 注册框架提供的核心中间件到中间件映射表中
// 这些中间件提供了基础的功能支持，包括异常处理、请求追踪、日志记录和超时控制
// 该函数在应用启动时被调用，确保所有系统中间件都可用
//
// 注册的中间件包括：
// 1. exceptionHandler - 异常处理中间件，统一处理应用异常
// 2. traceIdHandler - 请求追踪中间件，为每个请求分配唯一ID
// 3. traceLogHandler - 追踪日志中间件，记录请求的详细信息
// 4. timeoutHandler - 超时处理中间件，防止请求长时间阻塞
func initMiddleware() {
	// 注册 Prometheus 指标采集中间件
	// 统计 HTTP 请求总数、耗时分布和处理中请求数
	if err := RegisterMiddleware("prometheusHandler", middleware.PrometheusHandler); err != nil {
		logger.Error("%s", err.Error())
	}

	// 注册异常处理中间件
	// 提供统一的异常捕获和错误响应处理，确保应用在遇到异常时能够优雅降级
	if err := RegisterMiddleware("exceptionHandler", middleware.ExceptionHandler); err != nil {
		logger.Error("%s", err.Error())
	}

	// 注册请求追踪ID中间件
	// 为每个HTTP请求生成唯一的追踪ID，便于在分布式系统中追踪请求链路
	// 追踪ID会被添加到响应头和日志中，方便问题排查和性能分析
	if err := RegisterMiddleware("traceIdHandler", middleware.TraceIdHandler); err != nil {
		logger.Error("%s", err.Error())
	}

	// 注册追踪日志中间件
	// 记录请求的详细信息，包括请求路径、方法、响应状态、执行时间等
	// 提供完整的请求生命周期日志，支持性能监控和问题诊断
	if err := RegisterMiddleware("traceLogHandler", middleware.TraceLogHandler); err != nil {
		logger.Error("%s", err.Error())
	}

	// 注册超时处理中间件
	// 防止请求处理时间过长导致的资源耗尽和用户体验问题
	// 超时时间可通过配置文件中的 Service.ApiTimeout 参数设置
	// 当请求处理时间超过设定值时，会自动终止请求并返回超时错误
	if err := RegisterMiddleware("timeoutHandler", middleware.TimeoutHandler); err != nil {
		logger.Error("%s", err.Error())
	}
}

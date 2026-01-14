package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/logger"

	"github.com/gin-gonic/gin"
)

// Start 启动Web服务器
// 这是应用程序的主入口函数，负责完整的服务器启动流程：
// 1. 初始化验证器、配置、中间件和服务
// 2. 配置HTTP服务器参数
// 3. 启动性能分析服务器（非生产环境）
// 4. 启动主HTTP服务器
// 5. 实现优雅关闭机制
//
// 参数 exitfunctions: 可选的退出回调函数列表，在服务关闭时执行
// 这些函数可用于清理资源、保存状态等操作
//
// 服务器特性：
// - 支持优雅关闭（接收SIGINT/SIGTERM信号）
// - 自动配置读写超时
// - 集成pprof性能分析工具
// - 完整的资源清理机制
func Start(exitfunctions ...func()) {
	// 重写gin的Validator，使用自定义的验证器配置
	// 这允许我们自定义验证规则和错误信息格式
	overrideValidator()

	// 加载配置文件，包括默认配置和环境特定配置
	// 配置加载失败会导致程序退出
	loadConfig(app.Config)

	// 初始化系统中间件，注册所有可用的中间件到映射表
	// 包括异常处理、请求追踪、日志记录、超时控制等
	initMiddleware()

	// 初始化各种服务组件
	// 包括数据库、Redis、Elasticsearch、消息队列、定时任务等
	initService()

	// 创建用于优雅关闭的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动信号监听协程，处理优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		// 监听中断信号（Ctrl+C）和终止信号
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		// 收到信号后取消上下文，触发优雅关闭流程
		cancel()
	}()

	// 构建服务器监听地址
	serverAddr := fmt.Sprintf("%s:%d", app.BaseConfig.Service.Ip, app.BaseConfig.Service.Port)
	logger.Info("[server] Service start by %s:%d", app.BaseConfig.Service.Ip, app.BaseConfig.Service.Port)

	// 创建HTTP服务器实例，配置关键参数
	server := &http.Server{
		Addr:         serverAddr,                                                       // 监听地址
		Handler:      initEngine(),                                                     // Gin引擎作为请求处理器
		ReadTimeout:  time.Duration(app.BaseConfig.Service.ReadTimeout) * time.Second,  // 读取超时
		WriteTimeout: time.Duration(app.BaseConfig.Service.WriteTimeout) * time.Second, // 写入超时
	}

	// 启动优雅关闭处理协程
	go func() {
		// 等待关闭信号
		<-ctx.Done()
		fmt.Println("Shutdown HTTP Server ...")

		// 执行用户自定义的退出函数
		for _, exitfunction := range exitfunctions {
			exitfunction()
		}

		// 关闭各种服务连接
		_ = lifecycle.CloseServices(context.Background())

		// 设置5秒的关闭超时时间
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 优雅关闭HTTP服务器
		err := server.Shutdown(timeout)
		if err != nil {
			fmt.Printf("Failed to Shutdown HTTP Server: %v", err)
		}
	}()

	// 在非生产环境启动pprof性能分析服务器
	// pprof提供CPU分析、内存分析、协程分析等性能监控功能
	if app.Env != constant.ProdEnv {
		pprofPort := app.BaseConfig.Service.PprofPort
		// 默认使用6060端口，如果配置了特定端口则使用配置的端口
		pprofAddr := fmt.Sprintf("%s:%d", app.BaseConfig.Service.Ip, constant.DefaultPprofPort)
		if pprofPort != nil && *pprofPort != app.BaseConfig.Service.Port {
			pprofAddr = fmt.Sprintf("%s:%d", app.BaseConfig.Service.Ip, *pprofPort)
		}

		// 创建pprof服务器的路由器
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)

		// 在独立协程中启动pprof服务器
		go func() {
			logger.Info("[pprof server] Service start, access %s/debug/pprof to analysis", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, mux); err != nil {
				logger.Error("[pprof server]pprof服务启动异常：%v", err)
			}
		}()
	}

	// 启动主HTTP服务器（阻塞调用）
	// 服务器会持续运行直到收到关闭信号或发生错误
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("[server] 服务启动异常：%v", err)
	}
}

// NotFound 处理404错误（页面不存在）
// 当请求的路由不存在时，Gin框架会调用此函数
// 返回标准的HTTP 404响应，并记录详细的错误日志
//
// 参数 ctx: Gin上下文，包含请求和响应信息
//
// 响应内容：
// - HTTP状态码：404
// - 响应体：纯文本格式的"Not Found"
// - 日志：包含请求的详细信息，便于问题排查
func NotFound(ctx *gin.Context) {
	// 返回标准的404响应
	ctx.String(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	// 记录详细的错误日志，包含请求信息
	logger.Error("[server] Status: %d, Times(ms): %d, Ip: %s, Method: %s, Uri: %s, StatusText: %s",
		ctx.Writer.Status(), 0, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI, http.StatusText(http.StatusNotFound))
}

// MethodNotAllowed 处理405错误（方法不允许）
// 当请求的HTTP方法不被支持时，Gin框架会调用此函数
// 例如：路由只支持GET请求，但客户端发送了POST请求
//
// 参数 ctx: Gin上下文，包含请求和响应信息
//
// 响应内容：
// - HTTP状态码：405
// - 响应体：纯文本格式的"Method Not Allowed"
// - 日志：包含请求的详细信息，便于问题排查
func MethodNotAllowed(ctx *gin.Context) {
	// 返回标准的405响应
	ctx.String(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
	// 记录详细的错误日志，包含请求信息
	logger.Error("[server] Status: %d, Times(ms): %d, Ip: %s, Method: %s, Uri: %s, StatusText: %s",
		ctx.Writer.Status(), 0, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI, http.StatusText(http.StatusMethodNotAllowed))
}

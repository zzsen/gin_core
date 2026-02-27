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
	"github.com/zzsen/gin_core/metrics"

	"github.com/gin-gonic/gin"
)

// Start 启动 Web 服务器
// 这是应用程序的主入口函数，负责完整的服务器启动流程（钩子驱动）：
// 1. overrideValidator — 自定义验证器
// 2. loadConfig — 加载配置
// 3. ExecuteAppHooks(AppBeforeInit) — 应用初始化前钩子
// 4. initMiddleware — 初始化中间件
// 5. initService — 初始化服务组件
//   - 成功 → ExecuteAppHooks(AppAfterInit)
//   - 失败 → ExecuteAppHooks(AppOnInitFailed)，然后 panic
//
// 6. 创建 HTTP Server
// 7. server.ListenAndServe()
// 8. ExecuteAppHooks(AppOnReady)（在独立 goroutine 中，确认监听成功后触发）
//
// 关闭流程：
// 9.  收到 SIGINT/SIGTERM
// 10. ExecuteAppHooks(AppBeforeShutdown)
// 11. lifecycle.CloseServices()
// 12. server.Shutdown(shutdownTimeout)
// 13. ExecuteAppHooks(AppAfterShutdown)
//
// 服务器特性：
// - 支持优雅关闭（接收 SIGINT/SIGTERM 信号）
// - 优雅关闭超时可配置（ServiceInfo.ShutdownTimeout）
// - 应用级生命周期钩子驱动
// - 集成 pprof 性能分析工具
func Start() {
	// 1. 重写 gin 的 Validator
	overrideValidator()

	// 2. 加载配置文件
	loadConfig(app.Config)

	// 3. 执行应用初始化前钩子
	if err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppBeforeInit); err != nil {
		logger.Error("[server] AppBeforeInit 钩子执行失败: %v", err)
		_ = lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppOnInitFailed)
		panic(err)
	}

	// 4. 初始化系统中间件
	initMiddleware()

	// 5. 初始化各种服务组件
	initService()

	// 6. 执行应用初始化后钩子
	if err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppAfterInit); err != nil {
		logger.Error("[server] AppAfterInit 钩子执行失败: %v", err)
		_ = lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppOnInitFailed)
		panic(err)
	}

	// 创建用于优雅关闭的上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 启动 Prometheus 指标收集器
	if app.BaseConfig.Metrics.Enabled {
		metrics.StartCollector(ctx, 15*time.Second)
		logger.Info("[server] Prometheus 指标收集器已启动")
	}
	defer cancel()

	// 启动信号监听协程，处理优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	// 构建服务器监听地址
	serverAddr := fmt.Sprintf("%s:%d", app.BaseConfig.Service.Ip, app.BaseConfig.Service.Port)
	logger.Info("[server] Service start by %s:%d", app.BaseConfig.Service.Ip, app.BaseConfig.Service.Port)

	// 创建 HTTP 服务器实例
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      initEngine(),
		ReadTimeout:  time.Duration(app.BaseConfig.Service.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(app.BaseConfig.Service.WriteTimeout) * time.Second,
	}

	// 启动优雅关闭处理协程
	go func() {
		<-ctx.Done()
		fmt.Println("Shutdown HTTP Server ...")

		// 10. 执行应用关闭前钩子
		if err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppBeforeShutdown); err != nil {
			logger.Error("[server] AppBeforeShutdown 钩子执行失败: %v", err)
		}

		// 11. 关闭各种服务连接
		_ = lifecycle.CloseServices(context.Background())

		// 12. 使用可配置的关闭超时时间
		shutdownSeconds := app.BaseConfig.Service.GetShutdownTimeout()
		timeout, timeoutCancel := context.WithTimeout(context.Background(), time.Duration(shutdownSeconds)*time.Second)
		defer timeoutCancel()

		if err := server.Shutdown(timeout); err != nil {
			logger.Error("[server] HTTP Server 关闭失败: %v", err)
		}

		// 13. 执行应用关闭后钩子
		if err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppAfterShutdown); err != nil {
			logger.Error("[server] AppAfterShutdown 钩子执行失败: %v", err)
		}
	}()

	// 在非生产环境启动 pprof 性能分析服务器
	if app.Env != constant.ProdEnv {
		pprofPort := app.BaseConfig.Service.PprofPort
		pprofAddr := fmt.Sprintf("%s:%d", app.BaseConfig.Service.Ip, constant.DefaultPprofPort)
		if pprofPort != nil && *pprofPort != app.BaseConfig.Service.Port {
			pprofAddr = fmt.Sprintf("%s:%d", app.BaseConfig.Service.Ip, *pprofPort)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)

		go func() {
			logger.Info("[pprof server] Service start, access %s/debug/pprof to analysis", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, mux); err != nil {
				logger.Error("[pprof server] pprof 服务启动异常: %v", err)
			}
		}()
	}

	// 8. 在独立 goroutine 中触发 AppOnReady 钩子
	go func() {
		// 短暂等待确认 ListenAndServe 已启动
		time.Sleep(100 * time.Millisecond)
		if err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppOnReady); err != nil {
			logger.Error("[server] AppOnReady 钩子执行失败: %v", err)
		}
	}()

	// 7. 启动主 HTTP 服务器（阻塞调用）
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("[server] 服务启动异常: %v", err)
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

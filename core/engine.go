package core

import (
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"

	"github.com/gin-gonic/gin"
)

// optionFuncList 存储用户自定义的路由选项函数列表
// 这些函数会在引擎初始化时被依次调用，用于注册用户自定义的路由
// 支持动态添加多个路由配置函数，提供灵活的路由注册机制
var optionFuncList = make([]gin.OptionFunc, 0)

// AddOptionFunc 添加路由选项函数
// 允许用户注册自定义的路由配置函数，这些函数会在引擎初始化时被调用
// 支持传入多个函数，函数会按照添加顺序执行
// 参数 optionFunc: 一个或多个 gin.OptionFunc 类型的路由配置函数
//
// 使用示例：
//
//	AddOptionFunc(func(e *gin.Engine) {
//	  r := e.Group("/api")
//	  r.GET("/users", getUsersHandler)
//	})
func AddOptionFunc(optionFunc ...gin.OptionFunc) {
	optionFuncList = append(optionFuncList, optionFunc...)
}

// healthDetactEngine 健康检查路由配置函数
// 为应用添加健康检查端点，用于监控服务状态
// 提供标准的健康检查接口，便于负载均衡器、监控系统等外部服务检查应用状态
//
// 路由信息：
//   - GET /healthy       - 存活检查（liveness），始终返回健康状态
//   - GET /healthy/ready - 就绪检查（readiness），检查所有依赖服务状态
//   - GET /healthy/stats - 连接池统计信息
var healthDetactEngine = func(e *gin.Engine) {
	r := e.Group("healthy")

	// 存活检查 - 只要服务运行就返回健康
	r.GET("", func(c *gin.Context) {
		response.OkWithDetail(c, "healthy", gin.H{
			"status": "healthy",
		})
	})

	// 就绪检查 - 检查所有依赖服务
	r.GET("/ready", func(c *gin.Context) {
		health := app.CheckPoolHealth()
		allHealthy := app.IsAllHealthy()

		if allHealthy {
			response.OkWithDetail(c, "ready", gin.H{
				"status":   "ready",
				"services": health,
			})
		} else {
			c.JSON(503, gin.H{
				"code": 50300,
				"msg":  "not ready",
				"data": gin.H{
					"status":   "not ready",
					"services": health,
				},
			})
		}
	})

	// 连接池统计
	r.GET("/stats", func(c *gin.Context) {
		stats := app.GetPoolStats()
		response.OkWithDetail(c, "stats", stats)
	})
}

// metricsEngine Prometheus 指标端点配置函数
// 为应用添加 Prometheus 指标端点，用于指标采集
var metricsEngine = func(e *gin.Engine) {
	if !app.BaseConfig.Metrics.Enabled {
		return
	}

	path := app.BaseConfig.Metrics.Path
	if path == "" {
		path = "/metrics"
	}

	e.GET(path, gin.WrapH(promhttp.Handler()))
	logger.Info("[server] Prometheus 指标端点已启用: %s", path)
}

// initEngine 初始化Gin引擎
// 这是Web服务器引擎的核心初始化函数，负责：
// 1. 创建Gin引擎实例
// 2. 配置统一路由前缀
// 3. 注册Recovery中间件（异常恢复）
// 4. 注册用户配置的中间件
// 5. 配置404和405错误处理
// 6. 注册健康检查路由
// 7. 应用用户自定义的路由配置
//
// 返回值: 配置完成的 *gin.Engine 实例
func initEngine() *gin.Engine {
	// 创建新的Gin引擎实例（不包含默认中间件）
	engine := gin.New()

	// 配置统一路由前缀
	// 如果配置文件中设置了路由前缀，所有路由都会添加该前缀
	// 例如：设置前缀为 "/api/v1"，则所有路由都会变成 "/api/v1/xxx"
	if app.BaseConfig.Service.RoutePrefix != "" {
		engine.RouterGroup = *engine.RouterGroup.Group(app.BaseConfig.Service.RoutePrefix)
		logger.Info("[server] 统一路由前缀设置成功: %s",
			app.BaseConfig.Service.RoutePrefix)
	}

	// 添加Recovery中间件，用于捕获panic并恢复程序运行
	// 防止单个请求的panic导致整个服务崩溃
	engine.Use(gin.Recovery())

	// 注册用户配置的中间件
	// 从配置文件中读取需要启用的中间件列表，并按顺序注册
	useMiddlewares := app.BaseConfig.Service.Middlewares
	if len(useMiddlewares) > 0 {
		for _, useMiddleware := range useMiddlewares {
			// 检查中间件是否已注册到中间件映射表中
			if _, ok := middleWareMap[useMiddleware]; !ok {
				logger.Error("[server] can not find %s middleware, please register first", useMiddleware)
				os.Exit(1)
			}
			// 注册中间件到引擎
			engine.Use(middleWareMap[useMiddleware]())
		}
	}

	// 启用HTTP方法不允许的处理
	// 当请求的HTTP方法不被支持时，会调用MethodNotAllowed处理函数
	engine.HandleMethodNotAllowed = true
	// 设置405错误（方法不允许）的处理函数
	engine.NoMethod(MethodNotAllowed)
	// 设置404错误（路由不存在）的处理函数
	engine.NoRoute(NotFound)

	// 添加健康检查路由，用于检测服务是否正常运行
	AddOptionFunc(healthDetactEngine)
	// 添加 Prometheus 指标端点
	AddOptionFunc(metricsEngine)
	// 应用所有用户自定义的路由配置函数
	// 这些函数在应用启动时通过AddOptionFunc注册
	engine.With(optionFuncList...)

	return engine
}

package core

import (
	"os"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"

	"github.com/gin-gonic/gin"
)

var opts = make([]gin.OptionFunc, 0)

func AddOptionFunc(optionFunc gin.OptionFunc) {
	opts = append(opts, optionFunc)
}

func AddOptionFuncs(optionFuncs []gin.OptionFunc) {
	opts = append(opts, optionFuncs...)
}

// 健康检查
var healthDetactEngine = func(e *gin.Engine) {
	r := e.Group("healthy")
	r.GET("", func(c *gin.Context) {
		response.OkWithDetail(c, "healthy", gin.H{
			"status": "healthy",
		})
	})
}

func initEngine() *gin.Engine {
	engine := gin.New()
	if global.BaseConfig.Service.RoutePrefix != "" {
		engine.RouterGroup = *engine.RouterGroup.Group(formatRoute(global.BaseConfig.Service.RoutePrefix))
		logger.Info("[server] 统一路由前缀设置成功: %s",
			global.BaseConfig.Service.RoutePrefix)
	}
	engine.Use(gin.Recovery())

	// 注册中间件
	useMiddlewares := global.BaseConfig.Service.Middlewares
	if len(useMiddlewares) > 0 {
		for _, useMiddleware := range useMiddlewares {
			if _, ok := middleWareMap[useMiddleware]; !ok {
				logger.Error("[server] can not find %s middleware, please register first", useMiddleware)
				os.Exit(1)
			}
			engine.Use(middleWareMap[useMiddleware]())
		}
	}

	engine.HandleMethodNotAllowed = true
	engine.NoMethod(MethodNotAllowed)
	engine.NoRoute(NotFound)

	// 健康检查
	AddOptionFunc(healthDetactEngine)
	engine.With(opts...)

	return engine
}

package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"

	"github.com/gin-gonic/gin"
)

var middleWareMap = make(map[string]func() gin.HandlerFunc)

func RegisterMiddleware(name string, handlerFunc func() gin.HandlerFunc) error {
	if _, ok := middleWareMap[name]; ok {
		return errors.New("this name is already in use")
	}
	middleWareMap[name] = handlerFunc
	return nil
}

// new 新建对象
func new(opts ...gin.OptionFunc) *gin.Engine {
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
	engine.With(opts...)

	return engine
}

// Start 启动服务
func Start(opts []gin.OptionFunc, functions ...func()) {
	// 初始化服务
	initService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	serverAddr := fmt.Sprintf("%s:%d", global.BaseConfig.Service.Ip, global.BaseConfig.Service.Port)
	logger.Info("[server] Service start by %s:%d", global.BaseConfig.Service.Ip, global.BaseConfig.Service.Port)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      new(opts...),
		ReadTimeout:  time.Duration(global.BaseConfig.Service.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(global.BaseConfig.Service.WriteTimeout) * time.Second,
	}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutdown HTTP Server ...")
		for _, function := range functions {
			function()
		}
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()
		err := server.Shutdown(timeout)
		if err != nil {
			fmt.Printf("Failed to Shutdown HTTP Server: %v", err)
		}
	}()

	if err := server.ListenAndServe(); err != nil {
		logger.Error("[server] [server] 服务启动异常：%v", err)
	}
}

// NotFound 页面不存在
func NotFound(ctx *gin.Context) {
	ctx.String(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	logger.Error("[server] Status: %d, Times(ms): %d, Ip: %s, Method: %s, Uri: %s, StatusText: %s",
		ctx.Writer.Status(), 0, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI, http.StatusText(http.StatusNotFound))
}

// MethodNotAllowed 方法不允许
func MethodNotAllowed(ctx *gin.Context) {
	ctx.String(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
	logger.Error("[server] Status: %d, Times(ms): %d, Ip: %s, Method: %s, Uri: %s, StatusText: %s",
		ctx.Writer.Status(), 0, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI, http.StatusText(http.StatusMethodNotAllowed))
}

func formatRoute(data string) string {
	newData := fmt.Sprintf("/%s", strings.Trim(data, "/"))
	return newData
}

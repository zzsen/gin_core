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
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logging"

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

func initService() {
	if global.BaseConfig.System.UseRedis {
		initialize.InitRedis()
		initialize.InitRedisList()
	}
	if global.BaseConfig.System.UseMysql {
		initialize.InitDB()
	}
}

// new 新建对象
func new(opts ...gin.OptionFunc) *gin.Engine {
	// opts := []gin.OptionFunc{}
	opts = append(opts, func(e *gin.Engine) {
		e.Group(formatRoute(global.BaseConfig.Service.RoutePrefix))
	})

	engine := gin.New(opts...)
	engine.Use(gin.Recovery())

	// 注册中间件
	useMiddlewares := global.BaseConfig.Service.Middlewares
	if len(useMiddlewares) > 0 {
		for _, useMiddleware := range useMiddlewares {
			if _, ok := middleWareMap[useMiddleware]; !ok {
				logging.Error("can not find %s middleware,please register first", useMiddleware)
				os.Exit(1)
			}
			engine.Use(middleWareMap[useMiddleware]())
		}
	}

	engine.HandleMethodNotAllowed = true
	engine.NoMethod(MethodNotAllowed)
	engine.NoRoute(NotFound)

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
	logging.Info("Service start by %s:%d", global.BaseConfig.Service.Ip, global.BaseConfig.Service.Port)

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
		logging.Error("服务启动异常：%v", err)
	}
}

// NotFound 页面不存在
func NotFound(ctx *gin.Context) {
	ctx.String(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	logging.Error("Status: %d, Times(ms): %d, Ip: %s, Method: %s, Uri: %s, StatusText: %s",
		ctx.Writer.Status(), 0, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI, http.StatusText(http.StatusNotFound))
}

// MethodNotAllowed 方法不允许
func MethodNotAllowed(ctx *gin.Context) {
	ctx.String(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
	logging.Error("Status: %d, Times(ms): %d, Ip: %s, Method: %s, Uri: %s, StatusText: %s",
		ctx.Writer.Status(), 0, ctx.ClientIP(), ctx.Request.Method, ctx.Request.RequestURI, http.StatusText(http.StatusMethodNotAllowed))
}

func formatRoute(data string) string {
	newData := fmt.Sprintf("/%s", strings.Trim(data, "/"))
	return newData
}

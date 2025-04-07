package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"

	"github.com/gin-gonic/gin"
)

// Start 启动服务
func Start(exitfunctions ...func()) {
	// 加载配置
	loadConfig(global.Config)

	// 初始化中间件
	initMiddleware()

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
		Handler:      initEngine(),
		ReadTimeout:  time.Duration(global.BaseConfig.Service.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(global.BaseConfig.Service.WriteTimeout) * time.Second,
	}

	go func() {
		<-ctx.Done()
		fmt.Println("Shutdown HTTP Server ...")
		for _, exitfunction := range exitfunctions {
			exitfunction()
		}
		closeService()
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()
		err := server.Shutdown(timeout)

		if err != nil {
			fmt.Printf("Failed to Shutdown HTTP Server: %v", err)
		}
	}()

	if global.Env != constant.ProdEnv {
		pprofPort := global.BaseConfig.Service.PprofPort
		pprofAddr := fmt.Sprintf("%s:%d", global.BaseConfig.Service.Ip, constant.DefaultPprofPort)
		if pprofPort != nil && *pprofPort == global.BaseConfig.Service.Port {
			pprofAddr = fmt.Sprintf("%s:%d", global.BaseConfig.Service.Ip, *pprofPort)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		go func() {
			logger.Info("[pprof server] Service start, access %s/debug/pprof to analysis", pprofAddr)
			if err := http.ListenAndServe(pprofAddr, mux); err != nil {
				logger.Error("[pprof server]pprof服务启动异常：%v", err)
			}
		}()
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Error("[server] 服务启动异常：%v", err)
	}
}

func closeService() {
	// 关闭redis
	if global.BaseConfig.System.UseRedis && global.Redis != nil {
		global.Redis.Close()
	}
	// 关闭消息队列
	if global.BaseConfig.System.UseRabbitMQ && len(messageQueueConsumerList) > 0 {
		for _, rabbitMqProducer := range global.RabbitMQProducerList {
			rabbitMqProducer.Close()
			logger.Error("[server] [消息队列] 已关闭消息队列生产者：%s", rabbitMqProducer.GetInfo())
		}
	}
	// 关闭etcd
	if global.BaseConfig.System.UseEtcd && global.Etcd != nil {
		global.Etcd.Close()
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

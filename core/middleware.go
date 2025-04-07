package core

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/middleware"
)

var middleWareMap = make(map[string]func() gin.HandlerFunc)

func RegisterMiddleware(name string, handlerFunc func() gin.HandlerFunc) error {
	if _, ok := middleWareMap[name]; ok {
		return errors.New("this name is already in use")
	}
	middleWareMap[name] = handlerFunc
	return nil
}

func initMiddleware() {
	if err := RegisterMiddleware("exceptionHandler", middleware.ExceptionHandler); err != nil {
		logger.Error("%s", err.Error())
	}

	if err := RegisterMiddleware("traceLogHandler", middleware.TraceLogHandler); err != nil {
		logger.Error("%s", err.Error())
	}

	// timeout := time.Duration(global.BaseConfig.Service.ApiTimeout) * time.Second
	if err := RegisterMiddleware("timeoutHandler", middleware.TimeoutHandler); err != nil {
		logger.Error("%s", err.Error())
	}
}

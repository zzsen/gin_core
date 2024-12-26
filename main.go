package main

import (
	"fmt"

	"github.com/zzsen/gin_core/core"
	_ "github.com/zzsen/gin_core/middleware"
	"github.com/zzsen/gin_core/model/config"
	"github.com/zzsen/gin_core/model/response"

	"github.com/gin-gonic/gin"
)

type CustomConfig struct {
	config.BaseConfig `yaml:",inline"`
	Secret            string `yaml:"secret"`
}

func execFunc() {
	fmt.Println("server stop")
}

func getCustomRouter1() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("customRouter1")
		r.GET("test", func(c *gin.Context) {
			response.Ok(c)
		})
	}
}
func getCustomRouter2() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("customRouter2")
		r.GET("test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "success",
			})
		})
	}
}

func main() {
	opts := []gin.OptionFunc{}
	opts = append(opts, getCustomRouter1())
	opts = append(opts, getCustomRouter2())

	customConfig := &CustomConfig{}
	core.LoadConfig(customConfig)
	//启动服务
	core.Start(opts, execFunc)
}

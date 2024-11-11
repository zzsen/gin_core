package main

import (
	"fmt"

	"github.com/zzsen/github.com/zzsen/gin_core/core"
	_ "github.com/zzsen/github.com/zzsen/gin_core/middleware"
	"github.com/zzsen/github.com/zzsen/gin_core/model/config"

	"github.com/gin-gonic/gin"
)

type CustomConfig struct {
	config.BaseConfig `yaml:",inline"`
	Test              string `yaml:"test"`
	Id                string `yaml:"id"`
}

func print() {
	fmt.Println("***********************************")
}

func getCustomRouter1() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("111")
		r.GET("test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "success",
			})
		})
	}
}
func getCustomRouter2() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("222")
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
	core.Start(opts, print)
}

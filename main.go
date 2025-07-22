package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v9/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/logger"
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
			type TestData struct {
				Id   int    `json:"id"`
				Name string `json:"name"`
			}
			testDatas := []TestData{}
			size := 2
			resp, _ := app.ES.Search().Index("test_data*").Request(&search.Request{
				Query: &types.Query{
					Match: map[string]types.MatchQuery{
						"extension": types.MatchQuery{
							Query: ".html",
						},
					},
				},
				Size: &size,
			}).Do(context.Background())

			fmt.Println(resp.Hits.Total.Value)

			// 遍历命中结果
			for _, hit := range resp.Hits.Hits {
				// 声明结构体变量
				var testData TestData

				if err := json.Unmarshal(hit.Source_, &testData); err != nil {
					log.Printf("解析 %v 失败: %v", hit.Id_, err)
					continue
				}
				testDatas = append(testDatas, testData)
			}
			response.OkWithData(c, gin.H{
				"total": resp.Hits.Total.Value,
				"data":  testDatas,
			})
		})
	}
}
func getCustomRouter2() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("customRouter2")
		r.GET("test", func(c *gin.Context) {
			app.SendRabbitMqMsg("QueueName", "ExchangeName", "fanout", "RoutingKey", "message", "rabbitMQ1")

			c.JSON(200, gin.H{
				"message": "success",
			})
		})
	}
}

func mqFunc(message string) error {
	fmt.Println("message", message)
	return nil
}
func Print() {
	logger.Info("schedule run")
}

func main() {
	core.AddOptionFunc(getCustomRouter1())
	core.AddOptionFunc(getCustomRouter2())

	core.InitCustomConfig(&CustomConfig{})
	core.AddMessageQueueConsumer(config.MessageQueue{
		QueueName:    "QueueName",
		ExchangeName: "ExchangeName",
		ExchangeType: "fanout",
		RoutingKey:   "RoutingKey",
		Fun:          mqFunc,
	})
	core.AddSchedule(config.ScheduleInfo{
		Cron: "@every 10s",
		Cmd:  Print,
	})
	//启动服务
	core.Start(execFunc)
}

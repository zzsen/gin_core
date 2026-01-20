package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

			// 使用带超时的 context，避免请求长时间阻塞
			ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
			defer cancel()

			resp, err := app.ES.Search().Index("test_data*").Request(&search.Request{
				Query: &types.Query{
					Match: map[string]types.MatchQuery{
						"extension": types.MatchQuery{
							Query: ".html",
						},
					},
				},
				Size: &size,
			}).Do(ctx)

			// 正确处理 ES 查询错误
			if err != nil {
				logger.Error("ES查询失败: %v", err)
				response.FailWithMessage(c, "查询失败")
				return
			}

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
			err := app.SendRabbitMqMsg("QueueName", "ExchangeName", "fanout", "RoutingKey", "message", "rabbitMQ1")
			if err != nil {
				response.FailWithMessage(c, fmt.Sprintf("消息发送失败: %v", err))
				return
			}

			response.OkWithMessage(c, "消息发送成功")
		})
	}
}

func getCustomRouter3() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("customRouter3")
		r.GET("test", func(c *gin.Context) {
			logger.TraceWithFields(map[string]any{
				"traceId":      "1234567890",
				"requestId":    "1234567890",
				"statusCode":   200,
				"responseTime": 1000,
				"clientIp":     "127.0.0.1",
				"reqMethod":    "GET",
				"reqUri":       "/customRouter3/test",
				"body":         "{\"name\":\"test\"}",
				"errStr":       "",
			}, "password=1234567890")
			response.OkWithMessage(c, "消息发送成功")
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
	core.AddOptionFunc(getCustomRouter3())

	core.InitCustomConfig(&CustomConfig{})
	core.AddMessageQueueConsumer(&config.MessageQueue{
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

// Package core 提供 gin_core 框架的核心引擎功能。
//
// 本包是 gin_core 框架的入口层，封装了应用程序从启动到关闭的完整生命周期管理，
// 包括以下核心能力：
//
//   - 配置加载：支持多环境配置文件（YAML）自动加载与自定义配置注入
//   - 服务注册：基于依赖图的服务注册与自动初始化（MySQL、Redis、MQ 等）
//   - 中间件管理：内置常用中间件（CORS、限流、链路追踪、异常恢复等）并支持自定义扩展
//   - 生命周期钩子：提供 BeforeInit / AfterInit / Ready / BeforeShutdown / AfterShutdown 等钩子点
//   - HTTP 服务器：基于 Gin 的 HTTP 服务器管理，支持优雅关闭
//   - 命令行参数：支持通过命令行参数覆盖配置（环境、端口等）
//
// 基本用法：
//
//	core.Start(func(engine *gin.Engine) {
//	    // 注册路由
//	    engine.GET("/ping", func(c *gin.Context) {
//	        c.JSON(200, gin.H{"message": "pong"})
//	    })
//	})
package core

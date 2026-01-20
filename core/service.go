package core

import (
	"context"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/core/services"
	"github.com/zzsen/gin_core/model/config"
)

// --- 类型别名，保持向后兼容 ---

// Service 服务接口
type Service = lifecycle.Service

// HealthChecker 健康检查接口
type HealthChecker = lifecycle.HealthChecker

// Hook 初始化钩子
type Hook = lifecycle.Hook

// HookPhase 钩子执行阶段
type HookPhase = lifecycle.HookPhase

// ServiceState 服务状态
type ServiceState = lifecycle.ServiceState

// InitConfig 初始化配置
type InitConfig = lifecycle.InitConfig

// 钩子阶段常量
const (
	BeforeInit  = lifecycle.BeforeInit
	AfterInit   = lifecycle.AfterInit
	BeforeClose = lifecycle.BeforeClose
	AfterClose  = lifecycle.AfterClose
)

// 服务状态常量
const (
	StateUninitialized = lifecycle.StateUninitialized
	StateInitializing  = lifecycle.StateInitializing
	StateReady         = lifecycle.StateReady
	StateFailed        = lifecycle.StateFailed
	StateClosed        = lifecycle.StateClosed
)

// DefaultInitConfig 默认初始化配置
var DefaultInitConfig = lifecycle.DefaultInitConfig

// --- 全局函数，重导出 lifecycle 包的函数 ---

// RegisterService 注册服务到全局注册中心
func RegisterService(service Service) error {
	return lifecycle.RegisterService(service)
}

// RegisterServiceHook 注册钩子到全局注册中心
func RegisterServiceHook(serviceName string, hook Hook) {
	lifecycle.RegisterServiceHook(serviceName, hook)
}

// GetServiceState 获取服务状态
func GetServiceState(name string) ServiceState {
	return lifecycle.GetServiceState(name)
}

// SetInitConfig 设置全局初始化配置
func SetInitConfig(cfg InitConfig) {
	lifecycle.SetInitConfig(cfg)
}

// AddMessageQueueConsumer 添加消息队列消费者配置
func AddMessageQueueConsumer(messageQueue *config.MessageQueue) {
	lifecycle.AddMessageQueueConsumer(messageQueue)
}

// AddMessageQueueProducer 添加消息队列发送者配置
func AddMessageQueueProducer(messageQueue *config.MessageQueue) {
	lifecycle.AddMessageQueueProducer(messageQueue)
}

// AddSchedule 添加定时任务配置
func AddSchedule(schedule config.ScheduleInfo) {
	lifecycle.AddSchedule(schedule)
}

// --- 内部使用函数 ---

// registerBuiltinServices 注册内置服务
func registerBuiltinServices() {
	// 注册日志服务（最先初始化）
	_ = RegisterService(&services.LoggerService{})

	// 注册链路追踪服务（在日志之后、其他服务之前初始化）
	_ = RegisterService(&services.TracingService{})

	// 注册Redis服务
	_ = RegisterService(&services.RedisService{})

	// 注册MySQL服务
	_ = RegisterService(&services.MySQLService{})

	// 注册Elasticsearch服务
	_ = RegisterService(&services.ElasticsearchService{})

	// 注册RabbitMQ服务
	_ = RegisterService(services.NewRabbitMQService(
		lifecycle.GetMessageQueueConsumerList(),
		lifecycle.GetMessageQueueProducerList(),
	))

	// 注册Etcd服务
	_ = RegisterService(&services.EtcdService{})

	// 注册定时任务服务
	_ = RegisterService(services.NewScheduleService(lifecycle.GetScheduleList()))
}

// initService 初始化所有服务组件
func initService() {
	// 注册内置服务
	registerBuiltinServices()

	// 使用并行初始化器初始化所有服务
	ctx := context.Background()
	if err := lifecycle.InitAllServices(ctx, &app.BaseConfig); err != nil {
		panic(err)
	}
}

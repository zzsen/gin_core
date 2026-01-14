// Package services 提供框架内置服务的实现
// 所有服务都实现了 core.Service 接口，支持自动依赖解析和并行初始化
//
// 内置服务列表：
//   - LoggerService: 日志服务（优先级0，无依赖）
//   - RedisService: Redis缓存服务（优先级10，依赖logger）
//   - MySQLService: MySQL数据库服务（优先级10，依赖logger）
//   - ElasticsearchService: Elasticsearch搜索服务（优先级20，依赖logger）
//   - RabbitMQService: RabbitMQ消息队列服务（优先级30，依赖logger）
//   - EtcdService: Etcd配置中心服务（优先级20，依赖logger）
//   - ScheduleService: 定时任务服务（优先级100，依赖logger）
//
// 使用示例：
//
//	// 注册自定义服务
//	core.RegisterService(&MyCustomService{})
//
//	// 注册初始化钩子
//	core.RegisterServiceHook("mysql", core.Hook{
//	    Phase: core.AfterInit,
//	    Fn: func(ctx context.Context, name string) error {
//	        // 执行数据迁移
//	        return nil
//	    },
//	})
package services

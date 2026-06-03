// Package lifecycle 定义服务生命周期管理的核心接口与实现。
//
// 本包提供了一套完整的服务编排机制，用于管理框架内各组件（MySQL、Redis、
// Elasticsearch、RabbitMQ、Etcd、Tracing 等）的初始化和关闭流程，主要包括：
//
//   - Service 接口：定义服务的名称、优先级、依赖关系、初始化/关闭/健康检查等行为
//   - ServiceRegistry：服务注册中心，维护所有已注册服务的状态与生命周期钩子
//   - DependencyResolver：依赖解析器，基于拓扑排序计算服务的初始化顺序
//   - ParallelInitializer：并行初始化器，按依赖层级分组后同层并行、跨层串行地初始化服务
//   - Hook 机制：支持 BeforeInit / AfterInit / BeforeClose / AfterClose 四个阶段的钩子注入
//
// 服务初始化流程：
//
//	注册服务 → 依赖解析（拓扑排序）→ 分层分组 → 同层并行初始化 → 逆序并行关闭
package lifecycle

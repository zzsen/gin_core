// Package config 提供应用程序的配置结构定义
// 本文件定义了HTTP服务的配置结构，包含网络、会话、中间件和性能相关配置
package config

// ServiceInfo HTTP服务配置信息
// 该结构体包含了HTTP服务器运行所需的所有配置参数，支持中间件配置和性能调优
type ServiceInfo struct {
	Ip            string   `yaml:"ip"`            // 服务绑定的IP地址，支持0.0.0.0表示监听所有网络接口
	Port          int      `yaml:"port"`          // 服务监听的端口号，用于客户端连接
	RoutePrefix   string   `yaml:"routePrefix"`   // 路由前缀，所有API路由都会自动添加此前缀
	SessionExpire int      `yaml:"sessionExpire"` // 缓存的有效时长（秒），控制会话数据的过期时间
	SessionPrefix string   `yaml:"sessionPrefix"` // redis中缓存前缀，用于区分不同类型的会话数据
	Middlewares   []string `yaml:"middlewares"`   // 中间件列表，顺序对应中间件调用顺序，影响请求处理流程
	ApiTimeout    int      `yaml:"apiTimeout"`    // API超时时间（秒），超过此时间的请求会被自动终止
	ReadTimeout   int      `yaml:"readTimeout"`   // 读取超时时间（秒），控制HTTP请求体的读取超时
	WriteTimeout  int      `yaml:"writeTimeout"`  // 写入超时时间（秒），控制HTTP响应体的写入超时
	PprofPort     *int     `yaml:"pprofPort"`     // pprof服务端口，用于性能分析和调试，指针类型支持配置文件中不设置该字段
}

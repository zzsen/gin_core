package discovery

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

var (
	hooksOnce sync.Once
	activeReg *Registry
	regMu     sync.Mutex
)

// ShouldRegister 判断是否应自动注册本实例
//
// 条件：discovery 非空、enabled、serviceName 非空，且 IsRegister() 为 true。
func ShouldRegister(d *config.EtcdDiscoveryConfig) bool {
	if d == nil || !d.Enabled {
		return false
	}
	if d.ServiceName == "" {
		return false
	}
	return d.IsRegister()
}

// InstallHooks 注册 AppOnReady / AppBeforeShutdown 钩子（幂等）
//
// 【功能】在 EtcdService.Init 时挂载一次；实际注册/注销由生命周期触发
// 【流程】
//  1. sync.Once 保证只注册一次
//  2. AppOnReady → onReadyRegister
//  3. AppBeforeShutdown → onShutdownDeregister
func InstallHooks() {
	hooksOnce.Do(func() {
		// 步骤 2：Ready 后注册
		lifecycle.RegisterAppHook(lifecycle.AppHook{
			Name:     "etcd-discovery-register",
			Phase:    lifecycle.AppOnReady,
			Priority: 50,
			Fn:       onReadyRegister,
		})
		// 步骤 3：关机前注销
		lifecycle.RegisterAppHook(lifecycle.AppHook{
			Name:     "etcd-discovery-deregister",
			Phase:    lifecycle.AppBeforeShutdown,
			Priority: 50,
			Fn:       onShutdownDeregister,
		})
	})
}

// onReadyRegister AppOnReady：按配置注册本实例
//
// 【功能】条件满足时创建 Registry 并 Register；失败仅 warn，不阻断启动
// 【流程】
//  1. 读取配置，ShouldRegister 为 false 则跳过
//  2. app.Etcd 为空则 warn 并跳过
//  3. buildSelfInstance 组装本机 Instance
//  4. NewRegistry + Register；失败 warn
//  5. 保存 activeReg 供关机注销
func onReadyRegister(ctx context.Context) error {
	// 步骤 1：开关判断
	cfg := discoveryConfig()
	if !ShouldRegister(cfg) {
		return nil
	}

	// 步骤 2：依赖 Etcd 客户端
	if app.Etcd == nil {
		logger.Warn("[discovery] enabled but app.Etcd is nil, skip register")
		return nil
	}

	// 步骤 3：组装本机实例
	self, err := buildSelfInstance(cfg)
	if err != nil {
		logger.Warn("[discovery] build self instance failed: %v", err)
		return nil
	}

	// 步骤 4：注册到 Etcd
	reg := NewRegistry(app.Etcd, cfg, self)
	if err := reg.Register(ctx); err != nil {
		logger.Warn("[discovery] register failed: %v", err)
		return nil
	}

	// 步骤 5：保存活跃 Registry
	regMu.Lock()
	activeReg = reg
	regMu.Unlock()
	logger.Info("[discovery] registered service=%s instance=%s", self.ServiceName, self.InstanceID)
	return nil
}

// onShutdownDeregister AppBeforeShutdown：注销本实例
//
// 【功能】若 Ready 阶段成功注册过，则 Deregister；失败仅 warn
// 【流程】
//  1. 取出并清空 activeReg
//  2. 调用 Deregister
func onShutdownDeregister(ctx context.Context) error {
	// 步骤 1：取出活跃注册器
	regMu.Lock()
	reg := activeReg
	activeReg = nil
	regMu.Unlock()
	if reg == nil {
		return nil
	}

	// 步骤 2：注销
	if err := reg.Deregister(ctx); err != nil {
		logger.Warn("[discovery] deregister failed: %v", err)
	}
	return nil
}

// discoveryConfig 从全局配置读取 etcd.discovery（可能为 nil）
func discoveryConfig() *config.EtcdDiscoveryConfig {
	if app.BaseConfig.Etcd == nil {
		return nil
	}
	return app.BaseConfig.Etcd.Discovery
}

// buildSelfInstance 根据配置与 service 信息组装本机 Instance
//
// 【流程】
//  1. 解析 advertise IP/Port（配置优先，否则回落 service.ip/port）
//  2. 缺省 IP=127.0.0.1；Port 非法则报错
//  3. 解析 hostname 与 instanceID（空则 DefaultInstanceID）
//  4. 规范化 weight，组装 Instance
func buildSelfInstance(cfg *config.EtcdDiscoveryConfig) (Instance, error) {
	// 步骤 1：advertise 地址
	ip := cfg.AdvertiseIP
	port := cfg.AdvertisePort
	if ip == "" && app.BaseConfig.Service.Ip != "" {
		ip = app.BaseConfig.Service.Ip
	}
	if port == 0 {
		port = app.BaseConfig.Service.Port
	}

	// 步骤 2：缺省与校验
	if ip == "" {
		ip = "127.0.0.1"
	}
	if port <= 0 {
		return Instance{}, errInvalidAdvertise
	}

	// 步骤 3：hostname / instanceID
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	id := cfg.InstanceID
	if id == "" {
		id = DefaultInstanceID(host, port)
	}

	// 步骤 4：weight 与组装
	weight := cfg.Weight
	if weight <= 0 {
		weight = 1
	}
	return Instance{
		ServiceName: cfg.ServiceName,
		InstanceID:  id,
		IP:          ip,
		Port:        port,
		Weight:      weight,
		Meta:        cfg.Meta,
	}, nil
}

var errInvalidAdvertise = errors.New("discovery: invalid advertise port")

// Package services 配置中心服务测试
//
// ==================== 测试说明 ====================
// 验证 ConfigCenterService 的 ShouldInit 与依赖拓扑：启用时位于 etcd 之后、mysql 之前。
//
// 测试覆盖内容：
// 1. ShouldInit 开关
// 2. Resolve 层序（enabled）
// 3. disabled 时 mysql 不依赖 configcenter 节点
//
// 运行测试：go test ./core/services/ -count=1 -run ConfigCenter -v
// ==================================================

package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/configcenter"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/model/config"
)

// TestConfigCenterService_ShouldInit 开关语义
func TestConfigCenterService_ShouldInit(t *testing.T) {
	s := &ConfigCenterService{}
	assert.False(t, s.ShouldInit(&config.BaseConfig{System: config.SystemInfo{UseEtcd: true}}))
	assert.False(t, s.ShouldInit(&config.BaseConfig{
		System: config.SystemInfo{UseEtcd: true},
		Etcd:   &config.EtcdInfo{ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: false}},
	}))
	assert.True(t, s.ShouldInit(&config.BaseConfig{
		System: config.SystemInfo{UseEtcd: true},
		Etcd:   &config.EtcdInfo{ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: true}},
	}))
}

// TestConfigCenter_DependencyOrder_Enabled configcenter 在 mysql 前
//
// 【功能点】启用时拓扑 logger→etcd→configcenter→mysql
// 【测试流程】过滤 ShouldInit 后 Resolve，断言层序索引
func TestConfigCenter_DependencyOrder_Enabled(t *testing.T) {
	cfg := &config.BaseConfig{
		System: config.SystemInfo{UseEtcd: true, UseMysql: true},
		Etcd:   &config.EtcdInfo{ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: true}},
	}
	svcs := map[string]lifecycle.Service{}
	for _, s := range []lifecycle.Service{
		&LoggerService{},
		&EtcdService{},
		&ConfigCenterService{},
		&MySQLService{},
	} {
		if s.ShouldInit(cfg) {
			svcs[s.Name()] = s
		}
	}
	layers, err := lifecycle.NewDependencyResolver(svcs).Resolve()
	require.NoError(t, err)
	order := flatten(layers)
	assert.Less(t, indexOf(order, "etcd"), indexOf(order, "configcenter"))
	assert.Less(t, indexOf(order, "configcenter"), indexOf(order, "mysql"))
}

// TestConfigCenter_DependencyOrder_Disabled 未启用时无 configcenter 节点
func TestConfigCenter_DependencyOrder_Disabled(t *testing.T) {
	cfg := &config.BaseConfig{
		System: config.SystemInfo{UseEtcd: true, UseMysql: true},
		Etcd:   &config.EtcdInfo{ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: false}},
	}
	svcs := map[string]lifecycle.Service{}
	for _, s := range []lifecycle.Service{
		&LoggerService{},
		&EtcdService{},
		&ConfigCenterService{},
		&MySQLService{},
	} {
		if s.ShouldInit(cfg) {
			svcs[s.Name()] = s
		}
	}
	_, ok := svcs["configcenter"]
	assert.False(t, ok)
	layers, err := lifecycle.NewDependencyResolver(svcs).Resolve()
	require.NoError(t, err)
	order := flatten(layers)
	assert.Contains(t, order, "mysql")
	assert.NotContains(t, order, "configcenter")
}

func flatten(layers [][]string) []string {
	var out []string
	for _, layer := range layers {
		out = append(out, layer...)
	}
	return out
}

func indexOf(xs []string, name string) int {
	for i, x := range xs {
		if x == name {
			return i
		}
	}
	return -1
}

// TestConfigCenterService_Init_NilEtcd_Optional 可选模式跳过
func TestConfigCenterService_Init_NilEtcd_Optional(t *testing.T) {
	origEtcd := app.Etcd
	origCfg := app.BaseConfig
	t.Cleanup(func() {
		app.Etcd = origEtcd
		app.BaseConfig = origCfg
	})
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: true}},
	}
	s := &ConfigCenterService{}
	require.NoError(t, s.Init(context.Background()))
	require.NoError(t, s.Close(context.Background()))
}

// TestConfigCenterService_Init_NilEtcd_Required 必选模式失败
func TestConfigCenterService_Init_NilEtcd_Required(t *testing.T) {
	origEtcd := app.Etcd
	origCfg := app.BaseConfig
	t.Cleanup(func() {
		app.Etcd = origEtcd
		app.BaseConfig = origCfg
	})
	req := true
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: true, Required: &req}},
	}
	s := &ConfigCenterService{}
	err := s.Init(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, configcenter.ErrOverlayRequired)
}

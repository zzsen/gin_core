// Package services EtcdService 行为测试
//
// ==================== 测试说明 ====================
// 覆盖 required 降级/阻断、HealthCheck 边界（无需真实集群的路径）。
//
// 测试覆盖内容：
// 1. required=false 连接失败降级
// 2. required=true 连接失败返回 error
// 3. HealthCheck：客户端 nil、空 endpoints、非法 strategy 回退 any
//
// 运行测试：go test -v ./core/services/ -run TestEtcdService_
// ==================================================

package services

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

func startEmbeddedEtcdEndpointForService(t *testing.T) (endpoint string, shutdown func()) {
	t.Helper()
	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	cfg.ZapLoggerBuilder = embed.NewZapLoggerBuilder(zap.NewNop())

	peerLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	peerPort := peerLn.Addr().(*net.TCPAddr).Port
	require.NoError(t, peerLn.Close())

	clientLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	clientPort := clientLn.Addr().(*net.TCPAddr).Port
	require.NoError(t, clientLn.Close())

	parseU := func(raw string) url.URL {
		u, perr := url.Parse(raw)
		require.NoError(t, perr)
		return *u
	}
	pPeer := fmt.Sprintf("http://127.0.0.1:%d", peerPort)
	pClient := fmt.Sprintf("http://127.0.0.1:%d", clientPort)
	cfg.ListenPeerUrls = []url.URL{parseU(pPeer)}
	cfg.AdvertisePeerUrls = []url.URL{parseU(pPeer)}
	cfg.ListenClientUrls = []url.URL{parseU(pClient)}
	cfg.AdvertiseClientUrls = []url.URL{parseU(pClient)}
	cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)

	e, err := embed.StartEtcd(cfg)
	require.NoError(t, err)
	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(20 * time.Second):
		e.Close()
		t.Fatal("embedded etcd ready timeout")
	}
	ep := "127.0.0.1:" + strconv.Itoa(clientPort)
	return ep, func() { e.Close() }
}

// TestEtcdService_Init_RequiredFalse_连接失败降级 required 默认/false 时降级
//
// 【功能点】连接失败且 required=false 时 Init 成功且 app.Etcd 为 nil
// 【测试流程】短超时连不可达端口 → Init → 无 error、Etcd nil
func TestEtcdService_Init_RequiredFalse_连接失败降级(t *testing.T) {
	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		app.BaseConfig = origCfg
		if app.Etcd != nil && app.Etcd != origEtcd {
			_ = app.Etcd.Close()
		}
		app.Etcd = origEtcd
	}()

	timeout := 1
	req := false
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{"http://127.0.0.1:1"},
			Required:  &req,
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
		},
	}

	err := (&EtcdService{}).Init(context.Background())
	require.NoError(t, err)
	assert.Nil(t, app.Etcd)
}

// TestEtcdService_Init_RequiredTrue_连接失败返回Error required=true 阻断
//
// 【功能点】连接失败且 required=true 时 Init 返回 error
// 【测试流程】短超时连不可达端口 → Init → error
func TestEtcdService_Init_RequiredTrue_连接失败返回Error(t *testing.T) {
	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		app.BaseConfig = origCfg
		if app.Etcd != nil && app.Etcd != origEtcd {
			_ = app.Etcd.Close()
		}
		app.Etcd = origEtcd
	}()

	timeout := 1
	req := true
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{"http://127.0.0.1:1"},
			Required:  &req,
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
		},
	}

	err := (&EtcdService{}).Init(context.Background())
	require.Error(t, err)
	assert.Nil(t, app.Etcd)
}

// TestEtcdService_HealthCheck_空Endpoints 空 endpoints 报错
//
// 【功能点】客户端非 nil 但 endpoints 为空时 HealthCheck 失败且不 panic
func TestEtcdService_HealthCheck_空Endpoints(t *testing.T) {
	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		app.BaseConfig = origCfg
		app.Etcd = origEtcd
	}()

	app.Etcd = &clientv3.Client{}
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{Endpoints: []string{}},
	}
	err := (&EtcdService{}).HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoints")
}

// TestEtcdService_HealthCheck_非法Strategy_first回退Any first 非法时回退 any
//
// 【功能点】strategy=first 不报错，按默认 any 探活成功
// 【测试流程】embed etcd + strategy=first → HealthCheck nil
func TestEtcdService_HealthCheck_非法Strategy_first回退Any(t *testing.T) {
	ep, stop := startEmbeddedEtcdEndpointForService(t)
	defer stop()

	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		app.BaseConfig = origCfg
		app.Etcd = origEtcd
	}()

	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{ep},
			Health:    &config.EtcdHealthConfig{Strategy: "first"},
		},
	}
	require.NoError(t, initialize.InitEtcd())
	require.NotNil(t, app.Etcd)
	defer func() { _ = app.Etcd.Close(); app.Etcd = nil }()

	err := (&EtcdService{}).HealthCheck(context.Background())
	require.NoError(t, err)
}

// TestEtcdService_HealthCheck_客户端Nil 未初始化
func TestEtcdService_HealthCheck_客户端Nil(t *testing.T) {
	orig := app.Etcd
	defer func() { app.Etcd = orig }()
	app.Etcd = nil
	err := (&EtcdService{}).HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd未初始化")
}

// TestEtcdService_HealthCheck_StrategyAny_Embed strategy=any 成功
//
// 【功能点】embed etcd 上 HealthCheck(any) 返回 nil
// 【测试流程】InitEtcd → HealthCheck
func TestEtcdService_HealthCheck_StrategyAny_Embed(t *testing.T) {
	ep, stop := startEmbeddedEtcdEndpointForService(t)
	defer stop()

	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		if app.Etcd != nil && app.Etcd != origEtcd {
			_ = app.Etcd.Close()
		}
		app.BaseConfig = origCfg
		app.Etcd = origEtcd
	}()

	timeout := 5
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{ep},
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
			Health:    &config.EtcdHealthConfig{Strategy: "any"},
		},
	}
	require.NoError(t, initialize.InitEtcd())
	require.NoError(t, (&EtcdService{}).HealthCheck(context.Background()))
}

// TestEtcdService_HealthCheck_StrategyAll_Embed strategy=all 成功
//
// 【功能点】单节点 embed 上 HealthCheck(all) 返回 nil
func TestEtcdService_HealthCheck_StrategyAll_Embed(t *testing.T) {
	ep, stop := startEmbeddedEtcdEndpointForService(t)
	defer stop()

	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		if app.Etcd != nil && app.Etcd != origEtcd {
			_ = app.Etcd.Close()
		}
		app.BaseConfig = origCfg
		app.Etcd = origEtcd
	}()

	timeout := 5
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{ep},
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
			Health:    &config.EtcdHealthConfig{Strategy: "all"},
		},
	}
	require.NoError(t, initialize.InitEtcd())
	require.NoError(t, (&EtcdService{}).HealthCheck(context.Background()))
}

// TestEtcdService_HealthCheck_StrategyAll_部分失败 all 遇失败即 error
//
// 【功能点】endpoints 含不可达地址时 strategy=all 失败
func TestEtcdService_HealthCheck_StrategyAll_部分失败(t *testing.T) {
	ep, stop := startEmbeddedEtcdEndpointForService(t)
	defer stop()

	origCfg := app.BaseConfig
	origEtcd := app.Etcd
	defer func() {
		if app.Etcd != nil && app.Etcd != origEtcd {
			_ = app.Etcd.Close()
		}
		app.BaseConfig = origCfg
		app.Etcd = origEtcd
	}()

	timeout := 5
	app.Etcd = nil
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{ep},
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
			Health:    &config.EtcdHealthConfig{Strategy: "all"},
		},
	}
	require.NoError(t, initialize.InitEtcd())
	// 探测列表追加不可达地址（客户端仍指向可连集群）
	app.BaseConfig.Etcd.Endpoints = []string{ep, "127.0.0.1:1"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := (&EtcdService{}).HealthCheck(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all")
}

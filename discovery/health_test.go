// Package discovery 健康两阶段摘除测试
//
// ==================== 测试说明 ====================
// fake ReadyProbe + embed etcd 验证 weight=0 与 Deregister。
//
// 测试覆盖内容：
// 1. 连续失败 → degraded(weight=0)
// 2. 再连续失败 → unlinked（Key 消失）
// 3. unlink=false 时不启动循环（Start 返回的 loop 不跑业务：此处测 HealthUnlink 配置）
//
// 运行测试：go test -v ./discovery/ -run Health -timeout 120s
// ==================================================

package discovery

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestHealthUnlinker_DegradeThenUnlink 两阶段摘除
func TestHealthUnlinker_DegradeThenUnlink(t *testing.T) {
	ep, stop := startEmbeddedEtcd(t)
	defer stop()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer func() { _ = cli.Close() }()

	self := Instance{ServiceName: "svc", InstanceID: "h1", IP: "127.0.0.1", Port: 9, Weight: 2}
	cfg := &config.EtcdDiscoveryConfig{
		Enabled: true, Prefix: "services/", Env: "dev", ServiceName: "svc", TTLSeconds: 30, Weight: 2,
		Health: &config.EtcdDiscoveryHealthConfig{
			Unlink: true, IntervalSeconds: 60, FailThreshold: 2, SuccessThreshold: 2,
		},
	}
	reg := NewRegistry(cli, cfg, self)
	require.NoError(t, reg.Register(context.Background()))
	key := BuildKey(cfg.EffectivePrefix(), "dev", "svc", "h1")

	var fail atomic.Bool
	fail.Store(true)
	probe := func(ctx context.Context) error {
		if fail.Load() {
			return errors.New("not ready")
		}
		return nil
	}
	h := StartHealthUnlink(reg, cfg, probe)
	require.NotNil(t, h)
	defer h.Stop()

	// failThreshold=2 → degraded
	h.tick(context.Background())
	h.tick(context.Background())
	assert.Equal(t, healthStateDegraded, h.State())
	resp, err := cli.Get(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, resp.Kvs, 1)
	inst, err := UnmarshalInstance(resp.Kvs[0].Value)
	require.NoError(t, err)
	assert.Equal(t, 0, inst.Weight)

	// 再 2 次失败 → unlinked
	h.tick(context.Background())
	h.tick(context.Background())
	assert.Equal(t, healthStateUnlinked, h.State())
	resp2, err := cli.Get(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, 0, len(resp2.Kvs))
}

// TestHealthUnlinker_RecoverFromDegraded 降权后恢复
func TestHealthUnlinker_RecoverFromDegraded(t *testing.T) {
	ep, stop := startEmbeddedEtcd(t)
	defer stop()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer func() { _ = cli.Close() }()

	self := Instance{ServiceName: "svc", InstanceID: "h2", IP: "127.0.0.1", Port: 9, Weight: 3}
	cfg := &config.EtcdDiscoveryConfig{
		Enabled: true, Prefix: "services/", Env: "dev", ServiceName: "svc", TTLSeconds: 30, Weight: 3,
		Health: &config.EtcdDiscoveryHealthConfig{
			Unlink: true, IntervalSeconds: 60, FailThreshold: 1, SuccessThreshold: 2,
		},
	}
	reg := NewRegistry(cli, cfg, self)
	require.NoError(t, reg.Register(context.Background()))
	defer func() { _ = reg.Deregister(context.Background()) }()
	key := BuildKey(cfg.EffectivePrefix(), "dev", "svc", "h2")

	var fail atomic.Bool
	fail.Store(true)
	probe := func(ctx context.Context) error {
		if fail.Load() {
			return errors.New("not ready")
		}
		return nil
	}
	h := StartHealthUnlink(reg, cfg, probe)
	require.NotNil(t, h)
	defer h.Stop()

	h.tick(context.Background())
	assert.Equal(t, healthStateDegraded, h.State())

	fail.Store(false)
	h.tick(context.Background())
	h.tick(context.Background())
	assert.Equal(t, healthStateHealthy, h.State())
	resp, err := cli.Get(context.Background(), key)
	require.NoError(t, err)
	require.Len(t, resp.Kvs, 1)
	inst, err := UnmarshalInstance(resp.Kvs[0].Value)
	require.NoError(t, err)
	assert.Equal(t, 3, inst.Weight)
}

// TestEtcdDiscoveryConfig_HealthUnlinkDefault 默认不启用摘除
func TestEtcdDiscoveryConfig_HealthUnlinkDefault(t *testing.T) {
	assert.False(t, (&config.EtcdDiscoveryConfig{}).HealthUnlink())
}

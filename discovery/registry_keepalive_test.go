// Package discovery KeepAlive 重建测试
//
// ==================== 测试说明 ====================
// embed etcd：Revoke 租约后监督循环应重建 Key。
//
// 测试覆盖内容：
// 1. KeepAlive/租约被撤销后 Key 经重建再次可见
// 2. Deregister 后不再重建
//
// 运行测试：go test -v ./discovery/ -run KeepAlive -timeout 120s
// ==================================================

package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestRegistry_KeepAliveRebuild_AfterRevoke 租约被撤销后重建成功
//
// 【功能点】kaSupervisor 在 KeepAlive 结束后 Grant/Put/KeepAlive
// 【测试流程】
// 1. Register
// 2. 外部 Revoke 当前 lease
// 3. 等待重建后 Key 再次可见
func TestRegistry_KeepAliveRebuild_AfterRevoke(t *testing.T) {
	ep, stop := startEmbeddedEtcd(t)
	defer stop()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer func() { _ = cli.Close() }()

	self := Instance{ServiceName: "svc", InstanceID: "ka1", IP: "127.0.0.1", Port: 9, Weight: 1}
	rebuild := true
	cfg := &config.EtcdDiscoveryConfig{
		Enabled: true, Prefix: "services/", Env: "dev", ServiceName: "svc", TTLSeconds: 10,
		Keepalive: &config.EtcdDiscoveryKeepaliveConfig{Rebuild: &rebuild, MaxBackoffSeconds: 2},
	}
	reg := NewRegistry(cli, cfg, self)
	require.NoError(t, reg.Register(context.Background()))
	defer func() { _ = reg.Deregister(context.Background()) }()

	key := BuildKey(cfg.EffectivePrefix(), "dev", "svc", "ka1")
	reg.mu.Lock()
	leaseID := reg.leaseID
	reg.mu.Unlock()
	require.NotZero(t, leaseID)

	_, err = cli.Revoke(context.Background(), leaseID)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		resp, err := cli.Get(context.Background(), key)
		return err == nil && len(resp.Kvs) == 1
	}, 8*time.Second, 100*time.Millisecond)
}

// TestRegistry_KeepAliveRebuild_Disabled_NoRebuild rebuild=false 时不重建
func TestRegistry_KeepAliveRebuild_Disabled_NoRebuild(t *testing.T) {
	ep, stop := startEmbeddedEtcd(t)
	defer stop()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer func() { _ = cli.Close() }()

	self := Instance{ServiceName: "svc", InstanceID: "ka2", IP: "127.0.0.1", Port: 9, Weight: 1}
	rebuild := false
	cfg := &config.EtcdDiscoveryConfig{
		Enabled: true, Prefix: "services/", Env: "dev", ServiceName: "svc", TTLSeconds: 5,
		Keepalive: &config.EtcdDiscoveryKeepaliveConfig{Rebuild: &rebuild},
	}
	reg := NewRegistry(cli, cfg, self)
	require.NoError(t, reg.Register(context.Background()))

	key := BuildKey(cfg.EffectivePrefix(), "dev", "svc", "ka2")
	reg.mu.Lock()
	leaseID := reg.leaseID
	reg.mu.Unlock()
	_, err = cli.Revoke(context.Background(), leaseID)
	require.NoError(t, err)

	time.Sleep(1500 * time.Millisecond)
	resp, err := cli.Get(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, 0, len(resp.Kvs))
	_ = reg.Deregister(context.Background())
}

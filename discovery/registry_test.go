// Package discovery Registry 注册注销测试
//
// ==================== 测试说明 ====================
// 使用 embed etcd 验证 Register / Deregister。
//
// 测试覆盖内容：
// 1. Register 后 Key 可见
// 2. Deregister 后 Key 消失
//
// 运行测试：go test -v ./discovery/ -run TestRegistry_ -timeout 120s
// ==================================================

package discovery

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
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

func startEmbeddedEtcd(t *testing.T) (endpoint string, shutdown func()) {
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

// TestRegistry_RegisterThenGetKey 注册后可读、注销后删除
//
// 【功能点】Lease Put 与 Deregister
func TestRegistry_RegisterThenGetKey(t *testing.T) {
	ep, stop := startEmbeddedEtcd(t)
	defer stop()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer func() { _ = cli.Close() }()

	self := Instance{ServiceName: "svc", InstanceID: "i1", IP: "127.0.0.1", Port: 9, Weight: 1}
	cfg := &config.EtcdDiscoveryConfig{Enabled: true, Prefix: "services/", Env: "dev", ServiceName: "svc", TTLSeconds: 10}
	reg := NewRegistry(cli, cfg, self)
	require.NoError(t, reg.Register(context.Background()))
	key := BuildKey(cfg.EffectivePrefix(), "dev", "svc", "i1")
	resp, err := cli.Get(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Kvs))
	require.NoError(t, reg.Deregister(context.Background()))
	resp2, err := cli.Get(context.Background(), key)
	require.NoError(t, err)
	assert.Equal(t, 0, len(resp2.Kvs))
}

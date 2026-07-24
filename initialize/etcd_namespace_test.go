// Package initialize Etcd keyPrefix namespace 测试
//
// ==================== 测试说明 ====================
// 使用进程内 embed etcd 验证 InitEtcd 在 keyPrefix 非空时包装 KV。
//
// 测试覆盖内容：
// 1. keyPrefix 非空时 Put 落在带前缀的 key
// 2. keyPrefix 为空时不增加前缀
//
// 运行测试：go test -v ./initialize/ -run TestInitEtcd_KeyPrefix
// ==================================================

package initialize

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
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

func startEmbeddedEtcdEndpoint(t *testing.T) (endpoint string, shutdown func()) {
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

// TestInitEtcd_KeyPrefix包装KV keyPrefix 非空时 KV 带前缀
//
// 【功能点】InitEtcd 对 KV 做 namespace 包装
// 【测试流程】
// 1. 启动 embed etcd
// 2. InitEtcd(keyPrefix=/ns/)
// 3. 经 app.Etcd Put("foo")；裸客户端 Get("/ns/foo") 命中
func TestInitEtcd_KeyPrefix包装KV(t *testing.T) {
	ep, stopEmbed := startEmbeddedEtcdEndpoint(t)
	defer stopEmbed()

	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	timeout := 5
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{ep},
			KeyPrefix: "/ns/",
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
		},
	}
	require.NoError(t, InitEtcd())
	require.NotNil(t, app.Etcd)
	defer func() { _ = app.Etcd.Close(); app.Etcd = nil }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := app.Etcd.Put(ctx, "foo", "bar")
	require.NoError(t, err)

	raw, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{ep},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()

	resp, err := raw.Get(ctx, "/ns/foo")
	require.NoError(t, err)
	require.Len(t, resp.Kvs, 1)
	assert.Equal(t, "bar", string(resp.Kvs[0].Value))
}

// TestInitEtcd_空KeyPrefix不包装 空前缀时 key 原样写入
//
// 【功能点】keyPrefix 为空不包装
// 【测试流程】Put("plain") 后裸客户端 Get("plain") 命中
func TestInitEtcd_空KeyPrefix不包装(t *testing.T) {
	ep, stopEmbed := startEmbeddedEtcdEndpoint(t)
	defer stopEmbed()

	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	timeout := 5
	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Endpoints: []string{ep},
			KeyPrefix: "",
			Dial:      &config.EtcdDialConfig{Timeout: &timeout},
		},
	}
	require.NoError(t, InitEtcd())
	require.NotNil(t, app.Etcd)
	defer func() { _ = app.Etcd.Close(); app.Etcd = nil }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := app.Etcd.Put(ctx, "plain", "v1")
	require.NoError(t, err)

	raw, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{ep},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()

	resp, err := raw.Get(ctx, "plain")
	require.NoError(t, err)
	require.Len(t, resp.Kvs, 1)
	assert.Equal(t, "v1", string(resp.Kvs[0].Value))
}

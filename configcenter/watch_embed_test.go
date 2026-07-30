// Package configcenter StartWatch embed etcd 集成测试
//
// ==================== 测试说明 ====================
// 使用进程内 embed etcd 验证 Watch 热更可应用 rateLimit 白名单。
//
// 运行测试：go test ./configcenter/ -count=1 -run StartWatch_Embed -v
// ==================================================

package configcenter

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

func newEmbeddedEtcdClient(t *testing.T) (*clientv3.Client, func()) {
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
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 10 * time.Second})
	require.NoError(t, err)
	return cli, func() {
		_ = cli.Close()
		e.Close()
	}
}

// TestStartWatch_EmbedAppliesRateLimit Watch 后热更限流
func TestStartWatch_EmbedAppliesRateLimit(t *testing.T) {
	cli, cleanup := newEmbeddedEtcdClient(t)
	t.Cleanup(cleanup)

	orig := app.BaseConfig
	t.Cleanup(func() { app.BaseConfig = orig })
	app.BaseConfig = config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 100}}

	const key = "config/app.yml"
	_, err := cli.Put(context.Background(), key, "rateLimit:\n  defaultRate: 11\n  enabled: true\n")
	require.NoError(t, err)

	etcdCfg := &config.EtcdInfo{
		ConfigCenter: &config.EtcdConfigCenterConfig{
			Enabled:    true,
			Prefix:     key,
			DebounceMs: 50,
			Whitelist:  []string{"rateLimit.*"},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stop := StartWatch(ctx, cli, etcdCfg, "")
	t.Cleanup(stop)

	_, err = cli.Put(context.Background(), key, "rateLimit:\n  defaultRate: 22\n  enabled: true\n")
	require.NoError(t, err)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if app.BaseConfig.RateLimit.DefaultRate == 22 {
			assert.True(t, app.BaseConfig.RateLimit.Enabled)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("hot-reload timeout, rate=%d", app.BaseConfig.RateLimit.DefaultRate)
}

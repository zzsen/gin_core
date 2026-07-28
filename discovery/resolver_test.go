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

// TestResolver_WatchAndPick Register 后 Resolver 可见并可 Pick
func TestResolver_WatchAndPick(t *testing.T) {
	ep, stop := startEmbeddedEtcd(t)
	defer stop()
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{ep}, DialTimeout: 5 * time.Second})
	require.NoError(t, err)
	defer func() { _ = cli.Close() }()

	cfg := &config.EtcdDiscoveryConfig{Enabled: true, Prefix: "services/", Env: "dev", ServiceName: "svc", TTLSeconds: 10}
	self := Instance{ServiceName: "svc", InstanceID: "i1", IP: "127.0.0.1", Port: 9, Weight: 1}
	reg := NewRegistry(cli, cfg, self)
	require.NoError(t, reg.Register(context.Background()))
	defer func() { _ = reg.Deregister(context.Background()) }()

	res := NewResolver(cli, cfg.EffectivePrefix(), "dev")
	require.NoError(t, res.Start(context.Background()))
	defer res.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(res.GetInstances("svc")) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.NotEmpty(t, res.GetInstances("svc"))

	inst, err := res.Pick("svc", "round_robin")
	require.NoError(t, err)
	assert.Equal(t, "i1", inst.InstanceID)

	_, err = res.Pick("missing", "random")
	require.ErrorIs(t, err, ErrNoInstances)

	_, err = res.Pick("svc", "first")
	require.ErrorIs(t, err, ErrUnsupportedStrategy)
}

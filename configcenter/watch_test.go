// Package configcenter Watch 热更测试
//
// ==================== 测试说明 ====================
// 验证非法 YAML 走 reject 路径；reloadAndApply 白名单成功路径。
//
// 运行测试：go test ./configcenter/ -count=1 -run "Watch|reloadAndApply" -v
// ==================================================

package configcenter

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/metrics"
	"github.com/zzsen/gin_core/model/config"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestReloadAndApply_InvalidYAML_Rejects 非法 YAML 拒绝
func TestReloadAndApply_InvalidYAML_Rejects(t *testing.T) {
	orig := app.BaseConfig
	t.Cleanup(func() { app.BaseConfig = orig })
	app.BaseConfig = config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 9}}

	before := testutil.ToFloat64(metrics.ConfigReloadTotal.WithLabelValues("reject"))
	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{{Value: []byte(":::not-yaml")}}}, nil
	}}
	err := reloadAndApply(context.Background(), kv, "k", "", []string{"rateLimit.*"})
	require.Error(t, err)
	assert.Equal(t, 9, app.BaseConfig.RateLimit.DefaultRate)
	after := testutil.ToFloat64(metrics.ConfigReloadTotal.WithLabelValues("reject"))
	assert.GreaterOrEqual(t, after, before+1)
}

// TestReloadAndApply_KeyMissing_Rejects 空 KV reject
func TestReloadAndApply_KeyMissing_Rejects(t *testing.T) {
	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return &clientv3.GetResponse{}, nil
	}}
	err := reloadAndApply(context.Background(), kv, "missing", "", []string{"rateLimit.*"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

// TestReloadAndApply_RateLimitSuccess 白名单成功
func TestReloadAndApply_RateLimitSuccess(t *testing.T) {
	orig := app.BaseConfig
	t.Cleanup(func() { app.BaseConfig = orig })
	_ = logger.SetLevel("info")
	app.BaseConfig = config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 100}}

	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{{Value: []byte("rateLimit:\n  defaultRate: 3\n  enabled: true\n")}}}, nil
	}}
	require.NoError(t, reloadAndApply(context.Background(), kv, "k", "", []string{"rateLimit.*"}))
	assert.Equal(t, 3, app.BaseConfig.RateLimit.DefaultRate)
	assert.True(t, app.BaseConfig.RateLimit.Enabled)
}

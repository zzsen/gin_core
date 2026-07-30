// Package configcenter LoadOverlay 测试
//
// ==================== 测试说明 ====================
// 使用内存 KV 桩验证 LoadOverlay：合并、禁用短路、required 失败。
//
// 测试覆盖内容：
// 1. 成功 Get YAML 后合并进 app.BaseConfig
// 2. enabled=false 不调用 Get
// 3. required=true 且 Get 失败返回错误
// 4. required=false 且 Get 失败返回 nil 且不改配置
//
// 运行测试：go test ./configcenter/ -count=1 -run LoadOverlay -v
// ==================================================

package configcenter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type stubKV struct {
	getFn   func(ctx context.Context, key string) (*clientv3.GetResponse, error)
	gotKeys []string
}

func (s *stubKV) Get(ctx context.Context, key string, _ ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	s.gotKeys = append(s.gotKeys, key)
	if s.getFn != nil {
		return s.getFn(ctx, key)
	}
	return &clientv3.GetResponse{}, nil
}

// TestLoadOverlay_MergesIntoBaseConfig 成功合并
//
// 【功能点】Get YAML → 合并 rateLimit；保留 etcd endpoints
// 【测试流程】
// 1. 准备 BaseConfig 与 stub KV
// 2. LoadOverlay
// 3. 断言 DefaultRate 与 endpoints
func TestLoadOverlay_MergesIntoBaseConfig(t *testing.T) {
	orig := app.BaseConfig
	t.Cleanup(func() { app.BaseConfig = orig })

	app.BaseConfig = config.BaseConfig{
		RateLimit: config.RateLimitConfig{DefaultRate: 100},
		Etcd: &config.EtcdInfo{
			Endpoints: []string{"http://local:2379"},
			ConfigCenter: &config.EtcdConfigCenterConfig{
				Enabled: true,
				Prefix:  "config/app.yml",
			},
		},
	}

	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		assert.Equal(t, "config/app.yml", key)
		return &clientv3.GetResponse{
			Kvs: []*mvccpb.KeyValue{{Value: []byte("rateLimit:\n  defaultRate: 42\n")}},
		}, nil
	}}

	err := LoadOverlay(context.Background(), kv, app.BaseConfig.Etcd.ConfigCenter, "")
	require.NoError(t, err)
	assert.Equal(t, 42, app.BaseConfig.RateLimit.DefaultRate)
	assert.Equal(t, []string{"http://local:2379"}, app.BaseConfig.Etcd.Endpoints)
}

// TestLoadOverlay_DisabledNoGet enabled=false 不读 Etcd
func TestLoadOverlay_DisabledNoGet(t *testing.T) {
	kv := &stubKV{}
	cc := &config.EtcdConfigCenterConfig{Enabled: false}
	require.NoError(t, LoadOverlay(context.Background(), kv, cc, ""))
	assert.Empty(t, kv.gotKeys)
}

// TestLoadOverlay_RequiredGetError 阻断
func TestLoadOverlay_RequiredGetError(t *testing.T) {
	req := true
	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return nil, errors.New("boom")
	}}
	cc := &config.EtcdConfigCenterConfig{Enabled: true, Required: &req, Prefix: "k"}
	err := LoadOverlay(context.Background(), kv, cc, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOverlayRequired)
}

// TestLoadOverlay_OptionalGetError 跳过
func TestLoadOverlay_OptionalGetError(t *testing.T) {
	orig := app.BaseConfig
	t.Cleanup(func() { app.BaseConfig = orig })
	app.BaseConfig = config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 7}}

	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return nil, errors.New("boom")
	}}
	cc := &config.EtcdConfigCenterConfig{Enabled: true, Prefix: "k"}
	require.NoError(t, LoadOverlay(context.Background(), kv, cc, ""))
	assert.Equal(t, 7, app.BaseConfig.RateLimit.DefaultRate)
}

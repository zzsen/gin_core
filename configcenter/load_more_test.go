// Package configcenter LoadOverlay 补充边界测试
//
// ==================== 测试说明 ====================
// 覆盖 key 缺失、nil kv、nil cc。
//
// 运行测试：go test ./configcenter/ -count=1 -run LoadOverlay_ -v
// ==================================================

package configcenter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TestLoadOverlay_KeyMissing_Optional 可选模式下 key 缺失跳过
func TestLoadOverlay_KeyMissing_Optional(t *testing.T) {
	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return &clientv3.GetResponse{}, nil
	}}
	cc := &config.EtcdConfigCenterConfig{Enabled: true, Prefix: "missing"}
	require.NoError(t, LoadOverlay(context.Background(), kv, cc, ""))
}

// TestLoadOverlay_KeyMissing_Required 必选模式下 key 缺失失败
func TestLoadOverlay_KeyMissing_Required(t *testing.T) {
	req := true
	kv := &stubKV{getFn: func(ctx context.Context, key string) (*clientv3.GetResponse, error) {
		return &clientv3.GetResponse{}, nil
	}}
	cc := &config.EtcdConfigCenterConfig{Enabled: true, Required: &req, Prefix: "missing"}
	err := LoadOverlay(context.Background(), kv, cc, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOverlayRequired)
}

// TestLoadOverlay_NilKV_Required nil 客户端
func TestLoadOverlay_NilKV_Required(t *testing.T) {
	req := true
	cc := &config.EtcdConfigCenterConfig{Enabled: true, Required: &req}
	err := LoadOverlay(context.Background(), nil, cc, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOverlayRequired)
}

// TestLoadOverlay_NilConfig 空配置
func TestLoadOverlay_NilConfig(t *testing.T) {
	require.NoError(t, LoadOverlay(context.Background(), &stubKV{}, nil, ""))
}

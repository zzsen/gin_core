// Package configcenter StartWatch 边界测试
//
// ==================== 测试说明 ====================
// 验证未启用 / watch=false / cli=nil 时 StartWatch 返回可调用空 stop。
//
// 运行测试：go test ./configcenter/ -count=1 -run StartWatch -v
// ==================================================

package configcenter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// TestStartWatch_DisabledNoop 未启用返回空 stop
func TestStartWatch_DisabledNoop(t *testing.T) {
	stop := StartWatch(context.Background(), nil, &config.EtcdInfo{
		ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: false},
	}, "")
	require.NotNil(t, stop)
	stop()
}

// TestStartWatch_WatchFalseNoop watch=false
func TestStartWatch_WatchFalseNoop(t *testing.T) {
	w := false
	stop := StartWatch(context.Background(), nil, &config.EtcdInfo{
		ConfigCenter: &config.EtcdConfigCenterConfig{Enabled: true, Watch: &w},
	}, "")
	require.NotNil(t, stop)
	stop()
}

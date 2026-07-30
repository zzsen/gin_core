// Package configcenter 配置合并测试
//
// ==================== 测试说明 ====================
// 验证 BaseConfig 深度合并（etcd 覆盖 file）及 Etcd 连接 bootstrap 字段保护。
//
// 测试覆盖内容：
// 1. rateLimit.defaultRate 冲突时 overlay 胜出
// 2. overlay 中的 etcd.endpoints 不得改写 dst
//
// 运行测试：go test ./configcenter/ -count=1 -run MergeBaseConfig -v
// ==================================================

package configcenter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// TestMergeBaseConfig_EtcdWinsRateLimit overlay 覆盖 rateLimit.defaultRate
//
// 【功能点】etcd > file 合并语义
// 【测试流程】
// 1. dst DefaultRate=100，src DefaultRate=50
// 2. Merge 后 dst 为 50
func TestMergeBaseConfig_EtcdWinsRateLimit(t *testing.T) {
	dst := config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 100}}
	src := config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 50}}
	require.NoError(t, MergeBaseConfig(&dst, &src))
	assert.Equal(t, 50, dst.RateLimit.DefaultRate)
}

// TestMergeBaseConfig_StripsEtcdBootstrap 保护 endpoints
//
// 【功能点】合并后保留 dst 的 etcd 连接 bootstrap
// 【测试流程】
// 1. dst/src 均含不同 endpoints
// 2. Merge 后仍为 dst 原 endpoints
func TestMergeBaseConfig_StripsEtcdBootstrap(t *testing.T) {
	dst := config.BaseConfig{Etcd: &config.EtcdInfo{Endpoints: []string{"http://local:2379"}, Username: "u1"}}
	src := config.BaseConfig{Etcd: &config.EtcdInfo{Endpoints: []string{"http://evil:2379"}, Username: "evil"}}
	require.NoError(t, MergeBaseConfig(&dst, &src))
	require.NotNil(t, dst.Etcd)
	assert.Equal(t, []string{"http://local:2379"}, dst.Etcd.Endpoints)
	assert.Equal(t, "u1", dst.Etcd.Username)
}

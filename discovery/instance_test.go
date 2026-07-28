// Package discovery Instance / Key 工具测试
//
// ==================== 测试说明 ====================
// 覆盖服务发现实例序列化与 Key / instanceID 拼装。
//
// 测试覆盖内容：
// 1. BuildKey 路径拼装
// 2. DefaultInstanceID 默认规则
// 3. Instance JSON 往返
//
// 运行测试：go test -v ./discovery/ -run "TestBuildKey|TestInstance|TestDefaultInstanceID"
// ==================================================

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildKey_与DefaultInstanceID Key 与默认 instanceID
//
// 【功能点】prefix/env/service/id 拼装；hostname-port 默认 ID
func TestBuildKey_与DefaultInstanceID(t *testing.T) {
	assert.Equal(t, "services/dev/user/host-8080", BuildKey("services/", "dev", "user", "host-8080"))
	assert.Equal(t, "host-8080", DefaultInstanceID("host", 8080))
}

// TestInstance_JSON往返 序列化反序列化
//
// 【功能点】Instance 字段经 JSON 不丢关键信息
func TestInstance_JSON往返(t *testing.T) {
	in := Instance{ServiceName: "user", InstanceID: "h-1", IP: "10.0.0.1", Port: 8080, Weight: 1}
	b, err := in.MarshalValue()
	require.NoError(t, err)
	out, err := UnmarshalInstance(b)
	require.NoError(t, err)
	assert.Equal(t, in.IP, out.IP)
	assert.Equal(t, 8080, out.Port)
	assert.Equal(t, "user", out.ServiceName)
}

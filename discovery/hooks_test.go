package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zzsen/gin_core/model/config"
)

// TestShouldRegister_开关 注册开关语义
//
// 【功能点】enabled/serviceName/register 组合
func TestShouldRegister_开关(t *testing.T) {
	assert.False(t, ShouldRegister(nil))
	assert.False(t, ShouldRegister(&config.EtcdDiscoveryConfig{Enabled: false}))
	assert.False(t, ShouldRegister(&config.EtcdDiscoveryConfig{Enabled: true, ServiceName: ""}))
	reg := true
	assert.True(t, ShouldRegister(&config.EtcdDiscoveryConfig{Enabled: true, Register: &reg, ServiceName: "x"}))
	regFalse := false
	assert.False(t, ShouldRegister(&config.EtcdDiscoveryConfig{Enabled: true, Register: &regFalse, ServiceName: "x"}))
}

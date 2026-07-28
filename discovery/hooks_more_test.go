package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// TestBuildSelfInstance_Advertise与默认ID
//
// 【功能点】advertise 覆盖与 instanceID 默认生成
func TestBuildSelfInstance_Advertise与默认ID(t *testing.T) {
	orig := app.BaseConfig
	defer func() { app.BaseConfig = orig }()
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{Ip: "10.0.0.2", Port: 9000},
	}
	cfg := &config.EtcdDiscoveryConfig{
		Enabled:     true,
		ServiceName: "api",
		Weight:      0,
	}
	inst, err := buildSelfInstance(cfg)
	require.NoError(t, err)
	assert.Equal(t, "10.0.0.2", inst.IP)
	assert.Equal(t, 9000, inst.Port)
	assert.Equal(t, 1, inst.Weight)
	assert.Contains(t, inst.InstanceID, "-9000")

	cfg.AdvertiseIP = "1.2.3.4"
	cfg.AdvertisePort = 80
	cfg.InstanceID = "fixed-id"
	inst2, err := buildSelfInstance(cfg)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", inst2.IP)
	assert.Equal(t, 80, inst2.Port)
	assert.Equal(t, "fixed-id", inst2.InstanceID)
}

// TestInstallHooks_幂等
func TestInstallHooks_幂等(t *testing.T) {
	InstallHooks()
	InstallHooks()
}

// TestEffectivePrefix_空默认
func TestEffectivePrefix_空默认(t *testing.T) {
	assert.Equal(t, "services/", (&config.EtcdDiscoveryConfig{}).EffectivePrefix())
	assert.Equal(t, "services/", (*config.EtcdDiscoveryConfig)(nil).EffectivePrefix())
}

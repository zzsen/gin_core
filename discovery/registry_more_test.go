package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// TestRegistry_NilClient 空客户端报错
func TestRegistry_NilClient(t *testing.T) {
	reg := NewRegistry(nil, &config.EtcdDiscoveryConfig{ServiceName: "x", TTLSeconds: 10}, Instance{InstanceID: "i"})
	err := reg.Register(context.Background())
	require.Error(t, err)
}

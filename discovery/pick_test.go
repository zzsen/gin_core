package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPick_SkipZeroWeight 所有策略跳过 weight<=0
func TestPick_SkipZeroWeight(t *testing.T) {
	r := NewResolver(nil, "services/", "dev")
	r.mu.Lock()
	r.byService["svc"] = map[string]Instance{
		"a": {InstanceID: "a", Weight: 0, IP: "1.1.1.1", Port: 1},
		"b": {InstanceID: "b", Weight: 1, IP: "2.2.2.2", Port: 2},
	}
	r.mu.Unlock()

	for _, strategy := range []string{"round_robin", "random", "weighted_random"} {
		inst, err := r.Pick("svc", strategy)
		require.NoError(t, err, strategy)
		assert.Equal(t, "b", inst.InstanceID, strategy)
	}
}

// TestPick_AllZeroWeight_ErrNoInstances 全被过滤后报错
func TestPick_AllZeroWeight_ErrNoInstances(t *testing.T) {
	r := NewResolver(nil, "services/", "dev")
	r.mu.Lock()
	r.byService["svc"] = map[string]Instance{
		"a": {InstanceID: "a", Weight: 0},
	}
	r.mu.Unlock()
	_, err := r.Pick("svc", "round_robin")
	require.ErrorIs(t, err, ErrNoInstances)
}

// TestPick_RoundRobin_EqualAmongPositive round_robin 在正权重间等权轮询（忽略 weight 大小）
func TestPick_RoundRobin_EqualAmongPositive(t *testing.T) {
	r := NewResolver(nil, "services/", "dev")
	r.mu.Lock()
	r.byService["svc"] = map[string]Instance{
		"a": {InstanceID: "a", Weight: 1},
		"b": {InstanceID: "b", Weight: 9},
	}
	r.mu.Unlock()

	counts := map[string]int{}
	for i := 0; i < 20; i++ {
		inst, err := r.Pick("svc", "round_robin")
		require.NoError(t, err)
		counts[inst.InstanceID]++
	}
	assert.Equal(t, 10, counts["a"])
	assert.Equal(t, 10, counts["b"])
}

// TestPick_WeightedRandom_PrefersHigherWeight 高权重被选更多
func TestPick_WeightedRandom_PrefersHigherWeight(t *testing.T) {
	r := NewResolver(nil, "services/", "dev")
	r.mu.Lock()
	r.byService["svc"] = map[string]Instance{
		"low":  {InstanceID: "low", Weight: 1},
		"high": {InstanceID: "high", Weight: 9},
	}
	r.mu.Unlock()

	counts := map[string]int{}
	const n = 2000
	for i := 0; i < n; i++ {
		inst, err := r.Pick("svc", "weighted_random")
		require.NoError(t, err)
		counts[inst.InstanceID]++
	}
	assert.Greater(t, counts["high"], counts["low"])
	assert.Greater(t, counts["high"], n*6/10) // 粗阈值：高权重约 90%
}

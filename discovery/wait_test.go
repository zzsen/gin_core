package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPickWait_SucceedsAfterInject 空列表等待后成功
//
// 【功能点】PickWait 被 notify 唤醒
// 【测试流程】
// 1. 后台 PickWait
// 2. 注入正权重实例
// 3. 返回成功
func TestPickWait_SucceedsAfterInject(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan struct{})
	var inst Instance
	var err error
	go func() {
		inst, err = res.PickWait(ctx, "svc", "round_robin")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "i1", IP: "127.0.0.1", Port: 9, Weight: 1})

	select {
	case <-done:
		require.NoError(t, err)
		assert.Equal(t, "i1", inst.InstanceID)
	case <-time.After(3 * time.Second):
		t.Fatal("PickWait timeout")
	}
}

// TestPickWait_ContextCancel 取消立即返回
func TestPickWait_ContextCancel(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := res.PickWait(ctx, "svc", "round_robin")
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

// TestPickWait_Timeout ErrWaitTimeout
func TestPickWait_Timeout(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := res.PickWait(ctx, "svc", "round_robin")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrWaitTimeout) || errors.Is(err, context.DeadlineExceeded))
}

// TestPick_StillImmediateEmpty Pick 仍立即失败
func TestPick_StillImmediateEmpty(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	start := time.Now()
	_, err := res.Pick("svc", "round_robin")
	require.ErrorIs(t, err, ErrNoInstances)
	assert.Less(t, time.Since(start), 100*time.Millisecond)
}

// TestGetWait_IgnoresZeroWeight weight<=0 不算就绪
func TestGetWait_IgnoresZeroWeight(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "z", IP: "1.1.1.1", Port: 1, Weight: 0})
	_, err := res.GetWait(ctx, "svc")
	require.Error(t, err)
}

// TestGetWait_StopReturnsErrResolverStopped Stop 唤醒 waiter
func TestGetWait_StopReturnsErrResolverStopped(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := res.GetWait(ctx, "svc")
		done <- err
	}()
	time.Sleep(50 * time.Millisecond)
	res.Stop()

	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrResolverStopped)
	case <-time.After(3 * time.Second):
		t.Fatal("GetWait not unblocked by Stop")
	}
}

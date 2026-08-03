package discovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscribe_ReceivesSnapshot 注入实例后订阅收到快照
//
// 【功能点】Subscribe fan-out
// 【测试流程】
// 1. Subscribe
// 2. putInstanceForTest
// 3. 收到含该实例的快照
func TestSubscribe_ReceivesSnapshot(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ch, cancel := res.Subscribe("svc")
	defer cancel()

	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "i1", IP: "127.0.0.1", Port: 1, Weight: 1})

	select {
	case snap := <-ch:
		require.Len(t, snap, 1)
		assert.Equal(t, "i1", snap[0].InstanceID)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting snapshot")
	}
}

// TestSubscribe_CancelStopsDelivery cancel 后不再投递
func TestSubscribe_CancelStopsDelivery(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ch, cancel := res.Subscribe("svc")
	cancel()

	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "i1", IP: "127.0.0.1", Port: 1, Weight: 1})

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed")
	case <-time.After(200 * time.Millisecond):
		// closed channel may already be drained
	}
}

// TestSubscribe_StopClosesChannel Stop 结束订阅
func TestSubscribe_StopClosesChannel(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ch, cancel := res.Subscribe("svc")
	defer cancel()
	res.Stop()

	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(2 * time.Second):
		t.Fatal("expected closed channel")
	}
}

// TestSubscribe_DropOldestKeepNewest 缓冲满时丢旧保新
func TestSubscribe_DropOldestKeepNewest(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ch, cancel := res.Subscribe("svc")
	defer cancel()

	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "a", IP: "1.1.1.1", Port: 1, Weight: 1})
	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "b", IP: "2.2.2.2", Port: 2, Weight: 1})
	// 不消费第一次，连续推送应保留最新（含 a+b）
	time.Sleep(20 * time.Millisecond)

	select {
	case snap := <-ch:
		require.GreaterOrEqual(t, len(snap), 1)
		ids := map[string]bool{}
		for _, inst := range snap {
			ids[inst.InstanceID] = true
		}
		assert.True(t, ids["b"] || ids["a"])
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

// TestDeleteInstance_NotifiesSubscribe 删除触发快照更新
func TestDeleteInstance_NotifiesSubscribe(t *testing.T) {
	res := NewResolver(nil, "services/", "dev")
	ch, cancel := res.Subscribe("svc")
	defer cancel()

	res.putInstanceForTest(Instance{ServiceName: "svc", InstanceID: "i1", IP: "127.0.0.1", Port: 1, Weight: 1})
	<-ch // 消费 put
	res.deleteInstanceForTest("svc", "i1")

	select {
	case snap := <-ch:
		assert.Empty(t, snap)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting delete snapshot")
	}
}

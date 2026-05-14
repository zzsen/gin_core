// Package ratelimit 限流统计类型定义
package ratelimit

import (
	"sort"
	"sync"
	"sync/atomic"
)

// StatsProvider 限流统计接口
// 实现此接口的限流器可以提供详细的请求统计信息
type StatsProvider interface {
	DetailedStats() *LimiterStats
}

// LimiterStats 限流器详细统计
type LimiterStats struct {
	Type          string     `json:"type"`
	TotalAllowed  int64      `json:"total_allowed"`
	TotalRejected int64      `json:"total_rejected"`
	ActiveKeys    int        `json:"active_keys"`
	TopKeys       []KeyStats `json:"top_keys"`
}

// KeyStats 单个限流键的统计
type KeyStats struct {
	Key      string `json:"key"`
	Allowed  int64  `json:"allowed"`
	Rejected int64  `json:"rejected"`
}

// keyCounter 单个 key 的原子计数器
type keyCounter struct {
	allowed  atomic.Int64
	rejected atomic.Int64
}

// statsCollector 统计收集器，内嵌于 MemoryLimiter/RedisLimiter
type statsCollector struct {
	totalAllowed  atomic.Int64
	totalRejected atomic.Int64
	keyStats      sync.Map // map[string]*keyCounter
}

func (sc *statsCollector) recordAllow(key string) {
	sc.totalAllowed.Add(1)
	sc.getOrCreateKey(key).allowed.Add(1)
}

func (sc *statsCollector) recordReject(key string) {
	sc.totalRejected.Add(1)
	sc.getOrCreateKey(key).rejected.Add(1)
}

func (sc *statsCollector) getOrCreateKey(key string) *keyCounter {
	if v, ok := sc.keyStats.Load(key); ok {
		return v.(*keyCounter)
	}
	kc := &keyCounter{}
	actual, _ := sc.keyStats.LoadOrStore(key, kc)
	return actual.(*keyCounter)
}

func (sc *statsCollector) snapshot(typeName string) *LimiterStats {
	keys := make(map[string]*keyCounter)
	sc.keyStats.Range(func(k, v interface{}) bool {
		keys[k.(string)] = v.(*keyCounter)
		return true
	})
	return &LimiterStats{
		Type:          typeName,
		TotalAllowed:  sc.totalAllowed.Load(),
		TotalRejected: sc.totalRejected.Load(),
		ActiveKeys:    len(keys),
		TopKeys:       topNKeys(keys, 10),
	}
}

// topNKeys 从 keyStats map 中提取 top N 的键（按 allowed+rejected 降序）
func topNKeys(stats map[string]*keyCounter, n int) []KeyStats {
	if len(stats) == 0 {
		return nil
	}
	result := make([]KeyStats, 0, len(stats))
	for k, v := range stats {
		result = append(result, KeyStats{
			Key:      k,
			Allowed:  v.allowed.Load(),
			Rejected: v.rejected.Load(),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		totalI := result[i].Allowed + result[i].Rejected
		totalJ := result[j].Allowed + result[j].Rejected
		return totalI > totalJ
	})
	if n > 0 && n < len(result) {
		result = result[:n]
	}
	return result
}

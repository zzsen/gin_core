package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/metrics"
	"github.com/zzsen/gin_core/model/config"
)

// ReadyProbe 就绪探测（与 GET /healthy/ready 同源语义时可注入）
type ReadyProbe func(ctx context.Context) error

// DefaultReadyProbe 对接 app.IsAllHealthy（与 /healthy/ready 一致）
func DefaultReadyProbe(ctx context.Context) error {
	_ = ctx
	if app.IsAllHealthy() {
		return nil
	}
	return fmt.Errorf("discovery: readiness not ok")
}

const (
	healthStateHealthy  = "healthy"
	healthStateDegraded = "degraded"
	healthStateUnlinked = "unlinked"
)

// HealthUnlinker ready 驱动的两阶段摘除循环
type HealthUnlinker struct {
	reg    *Registry
	cfg    *config.EtcdDiscoveryConfig
	probe  ReadyProbe
	weight int // 配置侧正权重

	mu         sync.Mutex
	state      string
	failStreak int
	okStreak   int

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StartHealthUnlink 启动探测循环（调用方保证 unlink=true 且已注册）
func StartHealthUnlink(reg *Registry, cfg *config.EtcdDiscoveryConfig, probe ReadyProbe) *HealthUnlinker {
	if reg == nil || cfg == nil {
		return nil
	}
	if probe == nil {
		probe = DefaultReadyProbe
	}
	w := cfg.Weight
	if w <= 0 {
		w = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	h := &HealthUnlinker{
		reg:    reg,
		cfg:    cfg,
		probe:  probe,
		weight: w,
		state:  healthStateHealthy,
		cancel: cancel,
	}
	h.wg.Add(1)
	go h.loop(ctx)
	return h
}

// Stop 停止探测循环
func (h *HealthUnlinker) Stop() {
	if h == nil {
		return
	}
	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
}

// State 当前健康摘除状态（测试用）
func (h *HealthUnlinker) State() string {
	if h == nil {
		return ""
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state
}

func (h *HealthUnlinker) loop(ctx context.Context) {
	defer h.wg.Done()
	interval := time.Duration(h.cfg.HealthIntervalSeconds()) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.tick(ctx)
		}
	}
}

// tick 单次探测与状态迁移
//
// 【流程】
//  1. 调用 ReadyProbe
//  2. 失败累加 failStreak；达阈值则 healthy→degraded(weight=0) 或 degraded→unlinked
//  3. 成功累加 okStreak；degraded 达成功阈值则恢复 weight；unlinked 则重新 Register
func (h *HealthUnlinker) tick(ctx context.Context) {
	err := h.probe(ctx)
	if err != nil {
		metrics.DiscoveryHealthProbeTotal.WithLabelValues("fail").Inc()
		h.onFail(ctx)
		return
	}
	metrics.DiscoveryHealthProbeTotal.WithLabelValues("ok").Inc()
	h.onOK(ctx)
}

func (h *HealthUnlinker) onFail(ctx context.Context) {
	failTh := h.cfg.HealthFailThreshold()
	h.mu.Lock()
	h.failStreak++
	h.okStreak = 0
	state := h.state
	streak := h.failStreak
	h.mu.Unlock()

	if state == healthStateHealthy && streak >= failTh {
		if err := h.reg.UpdateWeight(ctx, 0); err != nil {
			logger.Warn("[discovery] health degrade update weight failed: %v", err)
			return
		}
		h.transition(healthStateHealthy, healthStateDegraded)
		h.mu.Lock()
		h.failStreak = 0
		h.mu.Unlock()
		logger.Warn("[discovery] health degraded: weight set to 0")
		return
	}
	if state == healthStateDegraded && streak >= failTh {
		if err := h.reg.Deregister(ctx); err != nil {
			logger.Warn("[discovery] health unlink deregister failed: %v", err)
			return
		}
		h.transition(healthStateDegraded, healthStateUnlinked)
		h.mu.Lock()
		h.failStreak = 0
		h.mu.Unlock()
		logger.Warn("[discovery] health unlinked: deregistered")
	}
}

func (h *HealthUnlinker) onOK(ctx context.Context) {
	okTh := h.cfg.HealthSuccessThreshold()
	h.mu.Lock()
	h.okStreak++
	h.failStreak = 0
	state := h.state
	streak := h.okStreak
	h.mu.Unlock()

	if state == healthStateDegraded && streak >= okTh {
		if err := h.reg.UpdateWeight(ctx, h.weight); err != nil {
			logger.Warn("[discovery] health recover update weight failed: %v", err)
			return
		}
		h.transition(healthStateDegraded, healthStateHealthy)
		h.mu.Lock()
		h.okStreak = 0
		h.mu.Unlock()
		logger.Info("[discovery] health recovered: weight restored")
		return
	}
	if state == healthStateUnlinked && streak >= okTh {
		if err := h.reg.Register(ctx); err != nil {
			logger.Warn("[discovery] health recover register failed: %v", err)
			return
		}
		h.transition(healthStateUnlinked, healthStateHealthy)
		h.mu.Lock()
		h.okStreak = 0
		h.mu.Unlock()
		logger.Info("[discovery] health recovered: re-registered")
	}
}

func (h *HealthUnlinker) transition(from, to string) {
	h.mu.Lock()
	h.state = to
	h.mu.Unlock()
	metrics.DiscoveryHealthTransitionTotal.WithLabelValues(from, to).Inc()
}

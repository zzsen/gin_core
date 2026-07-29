package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/metrics"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Registry 管理本实例在 Etcd 上的注册。
//
// 使用独立 Lease / KeepAlive，不与 distlock Session 共用，避免互相牵连。
type Registry struct {
	cli  *clientv3.Client
	cfg  *config.EtcdDiscoveryConfig
	self Instance

	mu         sync.Mutex
	leaseID    clientv3.LeaseID
	key        string
	value      []byte
	ttl        int64
	cancelKA   context.CancelFunc
	stopSuperv bool
	wg         sync.WaitGroup
}

// NewRegistry 创建注册器（尚未写入 Etcd，需再调用 Register）
func NewRegistry(cli *clientv3.Client, cfg *config.EtcdDiscoveryConfig, self Instance) *Registry {
	return &Registry{cli: cli, cfg: cfg, self: self}
}

// Register 创建 Lease、写入实例 Key 并启动 KeepAlive 监督循环
//
// 【功能】将本实例注册到 Etcd，并在进程存活期间续约；断流可按配置重建
// 【流程】
//  1. 校验并规范化字段
//  2. 序列化 value、拼装 Key
//  3. Grant Lease + Put
//  4. 启动 kaSupervisor（KeepAlive + 可选重建）
func (r *Registry) Register(ctx context.Context) error {
	if r == nil || r.cli == nil {
		return fmt.Errorf("discovery: registry client is nil")
	}
	if r.cfg == nil {
		return fmt.Errorf("discovery: config is nil")
	}

	// 步骤 1：规范化
	ttl := int64(r.cfg.TTLSeconds)
	if ttl <= 0 {
		ttl = 30
	}
	env := r.cfg.Env
	if env == "" {
		env = "default"
	}
	svc := r.cfg.ServiceName
	if svc == "" {
		svc = r.self.ServiceName
	}
	id := r.self.InstanceID
	if id == "" {
		return fmt.Errorf("discovery: instanceID is empty")
	}

	self := r.self
	self.ServiceName = svc
	self.InstanceID = id
	if self.Weight <= 0 {
		self.Weight = 1
	}
	if self.StartedAt.IsZero() {
		self.StartedAt = time.Now().UTC()
	}

	// 步骤 2：序列化与 Key
	val, err := self.MarshalValue()
	if err != nil {
		return err
	}
	key := BuildKey(r.cfg.EffectivePrefix(), env, svc, id)

	// 步骤 3：Grant + Put
	leaseResp, err := r.cli.Grant(ctx, ttl)
	if err != nil {
		return fmt.Errorf("discovery: grant lease: %w", err)
	}
	if _, err = r.cli.Put(ctx, key, string(val), clientv3.WithLease(leaseResp.ID)); err != nil {
		_, _ = r.cli.Revoke(ctx, leaseResp.ID)
		return fmt.Errorf("discovery: put instance: %w", err)
	}

	// 步骤 4：启动监督
	kaCtx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.leaseID = leaseResp.ID
	r.key = key
	r.value = val
	r.ttl = ttl
	r.cancelKA = cancel
	r.stopSuperv = false
	r.self = self
	r.mu.Unlock()

	r.wg.Add(1)
	go r.kaSupervisor(kaCtx, leaseResp.ID)
	return nil
}

// kaSupervisor 消费 KeepAlive；断流后按配置退避重建 Lease
//
// 【流程】
//  1. KeepAlive 当前 lease，直到 channel 关闭或 ctx 取消
//  2. 若已 stopSuperv / rebuild=false / ctx 取消 → 退出
//  3. 从 Registry 状态读取 key/value/ttl，退避后 Grant+Put，继续 KeepAlive
func (r *Registry) kaSupervisor(ctx context.Context, leaseID clientv3.LeaseID) {
	defer r.wg.Done()

	currentLease := leaseID
	backoff := time.Second
	maxBack := time.Duration(r.cfg.KeepaliveMaxBackoffSeconds()) * time.Second
	if maxBack <= 0 {
		maxBack = 30 * time.Second
	}

	for {
		ch, err := r.cli.KeepAlive(ctx, currentLease)
		if err == nil {
			for range ch {
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		r.mu.Lock()
		stopped := r.stopSuperv
		rebuild := r.cfg != nil && r.cfg.KeepaliveRebuild()
		key := r.key
		val := append([]byte(nil), r.value...)
		ttl := r.ttl
		r.mu.Unlock()
		if stopped || !rebuild || key == "" {
			return
		}

		logger.Warn("[discovery] keepalive ended, rebuilding lease key=%s err=%v", key, err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < maxBack {
			backoff *= 2
			if backoff > maxBack {
				backoff = maxBack
			}
		}

		leaseResp, gerr := r.cli.Grant(ctx, ttl)
		if gerr != nil {
			metrics.DiscoveryKeepaliveRebuildTotal.WithLabelValues("fail").Inc()
			logger.Warn("[discovery] keepalive rebuild grant failed: %v", gerr)
			continue
		}
		if _, perr := r.cli.Put(ctx, key, string(val), clientv3.WithLease(leaseResp.ID)); perr != nil {
			_, _ = r.cli.Revoke(ctx, leaseResp.ID)
			metrics.DiscoveryKeepaliveRebuildTotal.WithLabelValues("fail").Inc()
			logger.Warn("[discovery] keepalive rebuild put failed: %v", perr)
			continue
		}

		r.mu.Lock()
		if r.stopSuperv {
			r.mu.Unlock()
			_, _ = r.cli.Revoke(context.Background(), leaseResp.ID)
			return
		}
		r.leaseID = leaseResp.ID
		currentLease = leaseResp.ID
		r.mu.Unlock()

		metrics.DiscoveryKeepaliveRebuildTotal.WithLabelValues("success").Inc()
		logger.Info("[discovery] keepalive lease rebuilt key=%s", key)
		backoff = time.Second
	}
}

// UpdateWeight 更新注册 JSON 中的 weight（同 Key、同 Lease）
//
// 【功能】健康降权 / 恢复时使用；未注册返回 error
func (r *Registry) UpdateWeight(ctx context.Context, weight int) error {
	if r == nil || r.cli == nil {
		return fmt.Errorf("discovery: registry client is nil")
	}
	r.mu.Lock()
	key := r.key
	leaseID := r.leaseID
	self := r.self
	r.mu.Unlock()
	if key == "" || leaseID == 0 {
		return fmt.Errorf("discovery: not registered")
	}
	self.Weight = weight
	val, err := self.MarshalValue()
	if err != nil {
		return err
	}
	if _, err = r.cli.Put(ctx, key, string(val), clientv3.WithLease(leaseID)); err != nil {
		return fmt.Errorf("discovery: update weight: %w", err)
	}
	r.mu.Lock()
	r.self = self
	r.value = val
	r.mu.Unlock()
	return nil
}

// Deregister 停止监督并撤销 Lease / 删除 Key
//
// 【流程】
//  1. 标记 stopSuperv，取出 cancel/lease/key
//  2. cancel + 等待 supervisor 退出
//  3. Revoke 或 Delete
func (r *Registry) Deregister(ctx context.Context) error {
	if r == nil || r.cli == nil {
		return nil
	}

	// 步骤 1：停止监督
	r.mu.Lock()
	r.stopSuperv = true
	cancel := r.cancelKA
	leaseID := r.leaseID
	key := r.key
	r.cancelKA = nil
	r.leaseID = 0
	r.key = ""
	r.mu.Unlock()

	// 步骤 2：取消并等待
	if cancel != nil {
		cancel()
	}
	r.wg.Wait()

	// 步骤 3：撤销 Lease
	if leaseID != 0 {
		if _, err := r.cli.Revoke(ctx, leaseID); err != nil {
			if key != "" {
				_, _ = r.cli.Delete(ctx, key)
			}
			return fmt.Errorf("discovery: revoke lease: %w", err)
		}
		return nil
	}
	if key != "" {
		_, err := r.cli.Delete(ctx, key)
		return err
	}
	return nil
}

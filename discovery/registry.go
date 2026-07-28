package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

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

	mu       sync.Mutex
	leaseID  clientv3.LeaseID
	key      string
	cancelKA context.CancelFunc
}

// NewRegistry 创建注册器（尚未写入 Etcd，需再调用 Register）
func NewRegistry(cli *clientv3.Client, cfg *config.EtcdDiscoveryConfig, self Instance) *Registry {
	return &Registry{cli: cli, cfg: cfg, self: self}
}

// Register 创建 Lease、写入实例 Key 并启动 KeepAlive
//
// 【功能】将本实例注册到 Etcd，并在进程存活期间续约；失败返回 error（钩子侧通常仅 warn）
// 【流程】
//  1. 校验客户端与配置，规范化 TTL / env / serviceName / instanceID / weight
//  2. 序列化 Instance JSON，拼装注册 Key
//  3. Grant Lease
//  4. Put Key（绑定 Lease）；失败则 Revoke
//  5. 启动 KeepAlive 后台消费；失败则 Revoke
//  6. 保存 leaseID / key / cancel 到 Registry 状态
func (r *Registry) Register(ctx context.Context) error {
	if r == nil || r.cli == nil {
		return fmt.Errorf("discovery: registry client is nil")
	}
	if r.cfg == nil {
		return fmt.Errorf("discovery: config is nil")
	}

	// 步骤 1：规范化 TTL / env / 服务名 / 实例字段
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

	// 步骤 2：序列化 value 并拼装 Key
	val, err := self.MarshalValue()
	if err != nil {
		return err
	}
	key := BuildKey(r.cfg.EffectivePrefix(), env, svc, id)

	// 步骤 3：申请 Lease
	leaseResp, err := r.cli.Grant(ctx, ttl)
	if err != nil {
		return fmt.Errorf("discovery: grant lease: %w", err)
	}

	// 步骤 4：写入注册 Key（绑定 Lease）
	if _, err = r.cli.Put(ctx, key, string(val), clientv3.WithLease(leaseResp.ID)); err != nil {
		_, _ = r.cli.Revoke(ctx, leaseResp.ID)
		return fmt.Errorf("discovery: put instance: %w", err)
	}

	// 步骤 5：启动 KeepAlive；失败则撤销 Lease
	kaCtx, cancel := context.WithCancel(context.Background())
	ch, err := r.cli.KeepAlive(kaCtx, leaseResp.ID)
	if err != nil {
		cancel()
		_, _ = r.cli.Revoke(ctx, leaseResp.ID)
		return fmt.Errorf("discovery: keepalive: %w", err)
	}
	go func() {
		for range ch {
		}
	}()

	// 步骤 6：持久化本 Registry 运行态
	r.mu.Lock()
	r.leaseID = leaseResp.ID
	r.key = key
	r.cancelKA = cancel
	r.self = self
	r.mu.Unlock()
	return nil
}

// Deregister 停止 KeepAlive 并撤销 Lease / 删除 Key
//
// 【功能】优雅注销本实例，使发现侧尽快看不到该节点（不必仅等 TTL 过期）
// 【流程】
//  1. 取出并清空本地 leaseID / key / KeepAlive cancel
//  2. 停止 KeepAlive
//  3. 优先 Revoke Lease；失败则尝试 Delete Key
//  4. 无 Lease 时回退为 Delete Key
func (r *Registry) Deregister(ctx context.Context) error {
	if r == nil || r.cli == nil {
		return nil
	}

	// 步骤 1：快照并清空状态，避免并发重复注销
	r.mu.Lock()
	cancel := r.cancelKA
	leaseID := r.leaseID
	key := r.key
	r.cancelKA = nil
	r.leaseID = 0
	r.key = ""
	r.mu.Unlock()

	// 步骤 2：停止续约
	if cancel != nil {
		cancel()
	}

	// 步骤 3：优先撤销 Lease（Key 随 Lease 消失）
	if leaseID != 0 {
		if _, err := r.cli.Revoke(ctx, leaseID); err != nil {
			if key != "" {
				_, _ = r.cli.Delete(ctx, key)
			}
			return fmt.Errorf("discovery: revoke lease: %w", err)
		}
		return nil
	}

	// 步骤 4：无 Lease 时直接删 Key
	if key != "" {
		_, err := r.cli.Delete(ctx, key)
		return err
	}
	return nil
}

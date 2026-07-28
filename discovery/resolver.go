package discovery

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ErrNoInstances 目标服务无可用实例
var ErrNoInstances = errors.New("discovery: no instances")

// ErrUnsupportedStrategy 不支持的 Pick 策略
var ErrUnsupportedStrategy = errors.New("discovery: unsupported pick strategy")

// InstancePicker Pick 抽象（供 HTTPPicker 使用）
type InstancePicker interface {
	Pick(service, strategy string) (Instance, error)
}

// Resolver Watch 服务前缀并维护本地缓存。
type Resolver struct {
	cli    *clientv3.Client
	prefix string // EffectivePrefix()；完整 watch 前缀见 watchPrefix()
	env    string

	mu        sync.RWMutex
	byService map[string]map[string]Instance // service -> instanceID -> Instance

	rrCounters sync.Map // service -> *uint64
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewResolver 创建解析器
//
// prefix 为 discovery.EffectivePrefix()；env 为环境段（空则 default）。
// 创建后需调用 Start 才会拉取快照并 Watch。
func NewResolver(cli *clientv3.Client, prefix, env string) *Resolver {
	if env == "" {
		env = "default"
	}
	return &Resolver{
		cli:       cli,
		prefix:    prefix,
		env:       env,
		byService: make(map[string]map[string]Instance),
	}
}

// watchPrefix 返回实际 Watch 的前缀：{prefix}{env}/
func (r *Resolver) watchPrefix() string {
	return r.prefix + r.env + "/"
}

// Start 拉取当前快照并启动增量 Watch
//
// 【功能】用前缀 Get 初始化本地缓存，再从下一 revision 起 Watch 变更
// 【流程】
//  1. 校验客户端
//  2. 前缀 Get 全量快照
//  3. 加锁将快照写入 byService
//  4. 启动 watchLoop 协程（从 revision+1 开始）
func (r *Resolver) Start(ctx context.Context) error {
	// 步骤 1：校验
	if r == nil || r.cli == nil {
		return fmt.Errorf("discovery: resolver client is nil")
	}

	// 步骤 2：拉取快照
	wp := r.watchPrefix()
	resp, err := r.cli.Get(ctx, wp, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("discovery: list prefix: %w", err)
	}

	// 步骤 3：应用快照到本地缓存
	r.mu.Lock()
	for _, kv := range resp.Kvs {
		r.applyPutLocked(string(kv.Key), kv.Value)
	}
	r.mu.Unlock()

	// 步骤 4：启动增量 Watch
	wctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.watchLoop(wctx, wp, resp.Header.Revision+1)
	}()
	return nil
}

// Stop 取消 Watch 并等待 watchLoop 退出
func (r *Resolver) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
}

// watchLoop 消费 Etcd Watch 事件并更新本地缓存
//
// 【功能】持续处理 Put/Delete，直至 ctx 取消或 Watch channel 关闭/出错
// 【流程】
//  1. 建立从指定 revision 起的前缀 Watch
//  2. 逐批处理事件：Put → applyPut；Delete → applyDelete
//  3. Watch 报错则退出循环
func (r *Resolver) watchLoop(ctx context.Context, prefix string, rev int64) {
	// 步骤 1：建立 Watch
	wch := r.cli.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(rev))

	for wresp := range wch {
		// 步骤 3：出错则结束（由外层 Stop/重启策略决定是否重建）
		if wresp.Err() != nil {
			return
		}

		// 步骤 2：批量应用本批事件
		r.mu.Lock()
		for _, ev := range wresp.Events {
			key := string(ev.Kv.Key)
			switch ev.Type {
			case clientv3.EventTypePut:
				r.applyPutLocked(key, ev.Kv.Value)
			case clientv3.EventTypeDelete:
				r.applyDeleteLocked(key)
			}
		}
		r.mu.Unlock()
	}
}

// applyPutLocked 将 Put 事件写入缓存（调用方必须已持有 r.mu 写锁）
//
// 【流程】
//  1. 从 Key 解析 service / instanceID
//  2. 反序列化 Instance；补全 ServiceName / InstanceID
//  3. 写入 byService[service][instanceID]
func (r *Resolver) applyPutLocked(key string, val []byte) {
	// 步骤 1：解析 Key
	svc, id, ok := parseServiceInstance(r.watchPrefix(), key)
	if !ok {
		return
	}

	// 步骤 2：反序列化并补全字段
	inst, err := UnmarshalInstance(val)
	if err != nil {
		return
	}
	if inst.ServiceName == "" {
		inst.ServiceName = svc
	}
	if inst.InstanceID == "" {
		inst.InstanceID = id
	}

	// 步骤 3：写入缓存
	m := r.byService[svc]
	if m == nil {
		m = make(map[string]Instance)
		r.byService[svc] = m
	}
	m[id] = inst
}

// applyDeleteLocked 将 Delete 事件从缓存移除（调用方必须已持有 r.mu 写锁）
//
// 【流程】
//  1. 从 Key 解析 service / instanceID
//  2. 删除实例；若该服务无实例则删除服务桶
func (r *Resolver) applyDeleteLocked(key string) {
	// 步骤 1：解析 Key
	svc, id, ok := parseServiceInstance(r.watchPrefix(), key)
	if !ok {
		return
	}

	// 步骤 2：删除并清理空桶
	if m := r.byService[svc]; m != nil {
		delete(m, id)
		if len(m) == 0 {
			delete(r.byService, svc)
		}
	}
}

// parseServiceInstance 从完整 Key 解析 serviceName 与 instanceID
//
// Key 约定：{watchPrefix}{service}/{instanceID}
// 若路径段不足 2 段或前缀不匹配则 ok=false。
func parseServiceInstance(watchPrefix, key string) (service, instanceID string, ok bool) {
	if !strings.HasPrefix(key, watchPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(key, watchPrefix)
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	return parts[0], parts[len(parts)-1], true
}

// GetInstances 返回某服务全部实例的副本快照（只读锁）
func (r *Resolver) GetInstances(service string) []Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.byService[service]
	if len(m) == 0 {
		return nil
	}
	out := make([]Instance, 0, len(m))
	for _, inst := range m {
		out = append(out, inst)
	}
	return out
}

// Pick 按策略从本地缓存选择一个实例
//
// 【功能】负载选取；空列表立即返回 ErrNoInstances（不阻塞）
// 【流程】
//  1. 默认策略 round_robin
//  2. GetInstances；空则 ErrNoInstances
//  3. round_robin：原子计数取模；random：随机下标；其它：ErrUnsupportedStrategy
func (r *Resolver) Pick(service, strategy string) (Instance, error) {
	// 步骤 1：默认策略
	if strategy == "" {
		strategy = "round_robin"
	}

	// 步骤 2：取快照
	list := r.GetInstances(service)
	if len(list) == 0 {
		return Instance{}, ErrNoInstances
	}

	// 步骤 3：按策略选取
	switch strategy {
	case "round_robin":
		v, _ := r.rrCounters.LoadOrStore(service, new(uint64))
		n := atomic.AddUint64(v.(*uint64), 1)
		return list[int((n-1)%uint64(len(list)))], nil
	case "random":
		return list[rand.Intn(len(list))], nil
	default:
		return Instance{}, fmt.Errorf("%w: %s", ErrUnsupportedStrategy, strategy)
	}
}

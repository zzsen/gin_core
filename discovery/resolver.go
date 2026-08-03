package discovery

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
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

// WaitPicker 支持空列表等待的 Pick（供 HTTPPicker.DoWait）
type WaitPicker interface {
	InstancePicker
	PickWait(ctx context.Context, service, strategy string) (Instance, error)
}

type subscriber struct {
	ch     chan []Instance
	closed bool
}

// Resolver Watch 服务前缀并维护本地缓存。
type Resolver struct {
	cli    *clientv3.Client
	prefix string // EffectivePrefix()；完整 watch 前缀见 watchPrefix()
	env    string

	mu        sync.RWMutex
	byService map[string]map[string]Instance // service -> instanceID -> Instance
	subs      map[string]map[*subscriber]struct{}
	waiters   map[string][]chan struct{}
	stopped   bool

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
		subs:      make(map[string]map[*subscriber]struct{}),
		waiters:   make(map[string][]chan struct{}),
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

// Stop 取消 Watch、唤醒 waiter、关闭订阅并等待 watchLoop 退出
func (r *Resolver) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()

	r.mu.Lock()
	r.stopped = true
	// 唤醒全部 waiter
	for svc, list := range r.waiters {
		for _, ch := range list {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		delete(r.waiters, svc)
	}
	// 关闭全部订阅
	for svc, m := range r.subs {
		for sub := range m {
			if !sub.closed {
				sub.closed = true
				close(sub.ch)
			}
		}
		delete(r.subs, svc)
	}
	r.mu.Unlock()
}

// watchLoop 消费 Etcd Watch 事件并更新本地缓存
//
// 【功能】持续处理 Put/Delete，直至 ctx 取消或 Watch channel 关闭/出错
// 【流程】
//  1. 建立从指定 revision 起的前缀 Watch
//  2. 逐批处理事件：Put → applyPut；Delete → applyDelete
//  3. 解锁后 notify 变更服务
//  4. Watch 报错则退出循环
func (r *Resolver) watchLoop(ctx context.Context, prefix string, rev int64) {
	// 步骤 1：建立 Watch
	wch := r.cli.Watch(ctx, prefix, clientv3.WithPrefix(), clientv3.WithRev(rev))

	for wresp := range wch {
		if wresp.Err() != nil {
			return
		}

		changed := make(map[string]struct{})
		// 步骤 2：批量应用本批事件
		r.mu.Lock()
		for _, ev := range wresp.Events {
			key := string(ev.Kv.Key)
			svc, _, ok := parseServiceInstance(r.watchPrefix(), key)
			switch ev.Type {
			case clientv3.EventTypePut:
				r.applyPutLocked(key, ev.Kv.Value)
			case clientv3.EventTypeDelete:
				r.applyDeleteLocked(key)
			}
			if ok {
				changed[svc] = struct{}{}
			}
		}
		r.mu.Unlock()

		// 步骤 3：通知订阅者与 waiter
		for svc := range changed {
			r.notifyService(svc)
		}
	}
}

// applyPutLocked 将 Put 事件写入缓存（调用方必须已持有 r.mu 写锁）
func (r *Resolver) applyPutLocked(key string, val []byte) {
	svc, id, ok := parseServiceInstance(r.watchPrefix(), key)
	if !ok {
		return
	}

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

	m := r.byService[svc]
	if m == nil {
		m = make(map[string]Instance)
		r.byService[svc] = m
	}
	m[id] = inst
}

// applyDeleteLocked 将 Delete 事件从缓存移除（调用方必须已持有 r.mu 写锁）
func (r *Resolver) applyDeleteLocked(key string) {
	svc, id, ok := parseServiceInstance(r.watchPrefix(), key)
	if !ok {
		return
	}

	if m := r.byService[svc]; m != nil {
		delete(m, id)
		if len(m) == 0 {
			delete(r.byService, svc)
		}
	}
}

// parseServiceInstance 从完整 Key 解析 serviceName 与 instanceID
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

// snapshotLocked 返回服务实例副本（调用方持锁）
func (r *Resolver) snapshotLocked(service string) []Instance {
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

// GetInstances 返回某服务全部实例的副本快照（只读锁）
func (r *Resolver) GetInstances(service string) []Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.snapshotLocked(service)
}

// Pick 按策略从本地缓存选择一个实例
//
// 【功能】负载选取；过滤 weight<=0；空列表立即 ErrNoInstances
func (r *Resolver) Pick(service, strategy string) (Instance, error) {
	if strategy == "" {
		strategy = "round_robin"
	}

	list := filterPositiveWeight(r.GetInstances(service))
	if len(list) == 0 {
		return Instance{}, ErrNoInstances
	}
	sort.Slice(list, func(i, j int) bool { return list[i].InstanceID < list[j].InstanceID })

	switch strategy {
	case "round_robin":
		v, _ := r.rrCounters.LoadOrStore(service, new(uint64))
		n := atomic.AddUint64(v.(*uint64), 1)
		return list[int((n-1)%uint64(len(list)))], nil
	case "random":
		return list[rand.Intn(len(list))], nil
	case "weighted_random":
		return pickWeightedRandom(list), nil
	default:
		return Instance{}, fmt.Errorf("%w: %s", ErrUnsupportedStrategy, strategy)
	}
}

// Subscribe 订阅服务实例集快照变更（buffer=1，丢旧保新）
//
// 【流程】
//  1. 注册订阅 channel
//  2. 若已有实例则尝试推送当前快照
//  3. 返回 channel 与 cancel（cancel/Stop 关闭 channel）
func (r *Resolver) Subscribe(service string) (<-chan []Instance, func()) {
	sub := &subscriber{ch: make(chan []Instance, 1)}
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		close(sub.ch)
		return sub.ch, func() {}
	}
	if r.subs[service] == nil {
		r.subs[service] = make(map[*subscriber]struct{})
	}
	r.subs[service][sub] = struct{}{}
	snap := r.snapshotLocked(service)
	r.mu.Unlock()

	if len(snap) > 0 {
		pushSnapshot(sub, snap)
	}

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if m := r.subs[service]; m != nil {
				delete(m, sub)
				if len(m) == 0 {
					delete(r.subs, service)
				}
			}
			if !sub.closed {
				sub.closed = true
				close(sub.ch)
			}
		})
	}
	return sub.ch, cancel
}

// GetWait 等待至少 1 个正权重实例后返回快照副本
//
// 【流程】
//  1. 检查 stopped / 正权重列表
//  2. 空则挂起 waiter，直至 notify / ctx 取消 / Stop
//  3. 超时包装为 ErrWaitTimeout
func (r *Resolver) GetWait(ctx context.Context, service string) ([]Instance, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		r.mu.Lock()
		if r.stopped {
			r.mu.Unlock()
			return nil, ErrResolverStopped
		}
		list := filterPositiveWeight(r.snapshotLocked(service))
		if len(list) > 0 {
			out := append([]Instance(nil), list...)
			r.mu.Unlock()
			return out, nil
		}
		ch := make(chan struct{}, 1)
		r.waiters[service] = append(r.waiters[service], ch)
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			r.removeWaiter(service, ch)
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("%w: %w", ErrWaitTimeout, ctx.Err())
			}
			return nil, ctx.Err()
		case <-ch:
			// 重新检查
		}
	}
}

// PickWait 等待可用实例后按策略 Pick
func (r *Resolver) PickWait(ctx context.Context, service, strategy string) (Instance, error) {
	if _, err := r.GetWait(ctx, service); err != nil {
		return Instance{}, err
	}
	return r.Pick(service, strategy)
}

// notifyService 向订阅者推送快照并唤醒 waiter（写锁外调用）
func (r *Resolver) notifyService(service string) {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	snap := r.snapshotLocked(service)
	var subs []*subscriber
	for sub := range r.subs[service] {
		subs = append(subs, sub)
	}
	waiters := append([]chan struct{}(nil), r.waiters[service]...)
	r.waiters[service] = nil
	r.mu.Unlock()

	for _, sub := range subs {
		pushSnapshot(sub, snap)
	}
	for _, ch := range waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func pushSnapshot(sub *subscriber, snap []Instance) {
	if sub == nil {
		return
	}
	// 与 cancel/Stop 关闭 channel 竞态时吞 panic
	defer func() { _ = recover() }()
	if sub.closed {
		return
	}
	cp := append([]Instance(nil), snap...)
	select {
	case sub.ch <- cp:
		return
	default:
	}
	// 丢旧保新
	select {
	case <-sub.ch:
	default:
	}
	select {
	case sub.ch <- cp:
	default:
	}
}

func (r *Resolver) removeWaiter(service string, target chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	list := r.waiters[service]
	out := list[:0]
	for _, ch := range list {
		if ch != target {
			out = append(out, ch)
		}
	}
	if len(out) == 0 {
		delete(r.waiters, service)
	} else {
		r.waiters[service] = out
	}
}

// putInstanceForTest 测试辅助：写入缓存并 notify（同包单测用）
func (r *Resolver) putInstanceForTest(inst Instance) {
	if inst.ServiceName == "" || inst.InstanceID == "" {
		return
	}
	key := r.watchPrefix() + inst.ServiceName + "/" + inst.InstanceID
	val, err := inst.MarshalValue()
	if err != nil {
		return
	}
	r.mu.Lock()
	r.applyPutLocked(key, val)
	r.mu.Unlock()
	r.notifyService(inst.ServiceName)
}

// deleteInstanceForTest 测试辅助：从缓存删除并 notify
func (r *Resolver) deleteInstanceForTest(service, instanceID string) {
	key := r.watchPrefix() + service + "/" + instanceID
	r.mu.Lock()
	r.applyDeleteLocked(key)
	r.mu.Unlock()
	r.notifyService(service)
}

// filterPositiveWeight 去掉 weight<=0 的实例
func filterPositiveWeight(in []Instance) []Instance {
	out := make([]Instance, 0, len(in))
	for _, inst := range in {
		if inst.Weight > 0 {
			out = append(out, inst)
		}
	}
	return out
}

// pickWeightedRandom 按正 weight 比例随机选取
func pickWeightedRandom(list []Instance) Instance {
	total := 0
	for _, inst := range list {
		total += inst.Weight
	}
	if total <= 0 {
		return list[0]
	}
	x := rand.Intn(total)
	for _, inst := range list {
		x -= inst.Weight
		if x < 0 {
			return inst
		}
	}
	return list[len(list)-1]
}

package configcenter

import (
	"context"
	"sync"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/metrics"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

// StartWatch 监听配置 key，防抖后拉取并白名单 Apply。
//
// 未启用配置中心、关闭 Watch 或客户端为空时返回空 stop（noop）。
//
// 【流程】
// 1. 校验开关与客户端，派生可取消 ctx
// 2. 后台 goroutine Watch key；事件触发重置 debounce timer
// 3. 防抖到期后调用 reloadAndApply；校验失败仅 warn，保留旧运行态
// 4. 返回 stop，用于取消 Watch（幂等）
func StartWatch(ctx context.Context, cli *clientv3.Client, etcdCfg *config.EtcdInfo, cipherKey string) (stop func()) {
	// 步骤1：短路与可取消上下文
	if cli == nil || etcdCfg == nil || !etcdCfg.ConfigCenterEnabled() || !etcdCfg.ConfigCenterWatch() {
		return func() {}
	}

	wctx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop = func() { once.Do(cancel) }

	key := etcdCfg.ConfigCenterPrefix()
	debounce := time.Duration(etcdCfg.ConfigCenterDebounceMs()) * time.Millisecond
	whitelist := etcdCfg.ConfigCenterWhitelist()

	// 步骤2–3：Watch 循环 + 防抖热更
	go func() {
		var timer *time.Timer
		var mu sync.Mutex
		reset := func() {
			mu.Lock()
			defer mu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, func() {
				if err := reloadAndApply(wctx, cli, key, cipherKey, whitelist); err != nil {
					logger.Warn("[configcenter] watch reload rejected: %v", err)
				}
			})
		}

		ch := cli.Watch(wctx, key)
		for wresp := range ch {
			if wresp.Canceled {
				return
			}
			if wresp.Err() != nil {
				logger.Warn("[configcenter] watch error: %v", wresp.Err())
				continue
			}
			if len(wresp.Events) == 0 {
				continue
			}
			reset()
		}
	}()

	// 步骤4：返回 stop
	return stop
}

// reloadAndApply 热更单次执行：Get → ENV/CIPHER → YAML → ApplyWhitelist → 写回 app.BaseConfig。
//
// 失败时增加 config_reload_total{result=error|reject}，不修改已应用的运行态（Apply 前失败）
// 或在 Apply 失败时返回 error（ApplyWhitelist 失败前不应部分写回全局）。
//
// 【流程】
// 1. Get key；缺失或错误 → reject/error
// 2. prepareYAML + Unmarshal
// 3. ApplyWhitelist 到 BaseConfig 副本并写回
// 4. 成功打点 success
func reloadAndApply(ctx context.Context, kv KVGet, key, cipherKey string, whitelist []string) error {
	// 步骤1：拉取
	resp, err := kv.Get(ctx, key)
	if err != nil {
		metrics.ConfigReloadTotal.WithLabelValues("error").Inc()
		return err
	}
	if resp == nil || len(resp.Kvs) == 0 {
		metrics.ConfigReloadTotal.WithLabelValues("reject").Inc()
		return errMissingKey(key)
	}
	// 步骤2：预处理与反序列化
	raw, err := prepareYAML(resp.Kvs[0].Value, cipherKey)
	if err != nil {
		metrics.ConfigReloadTotal.WithLabelValues("reject").Inc()
		return err
	}
	var next config.BaseConfig
	if err := yaml.Unmarshal(raw, &next); err != nil {
		metrics.ConfigReloadTotal.WithLabelValues("reject").Inc()
		return err
	}

	// 步骤3：白名单应用
	prev := app.BaseConfig
	if err := ApplyWhitelist(&prev, &next, whitelist); err != nil {
		metrics.ConfigReloadTotal.WithLabelValues("reject").Inc()
		return err
	}
	app.BaseConfig = prev
	// 步骤4：成功指标
	metrics.ConfigReloadTotal.WithLabelValues("success").Inc()
	logger.Info("[configcenter] hot-reload applied key=%s", key)
	return nil
}

// errMissingKey 构造 Watch 热更时 key 不存在的 reject 错误。
func errMissingKey(key string) error {
	return &overlayError{msg: "configcenter: watch key missing: " + key}
}

// overlayError 配置中心热更/校验失败的轻量错误类型。
type overlayError struct{ msg string }

// Error 实现 error 接口。
func (e *overlayError) Error() string { return e.msg }

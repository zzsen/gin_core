// Package configcenter 提供基于 Etcd 的配置 overlay 与白名单热更（opt-in）
package configcenter

import (
	"context"
	"errors"
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/metrics"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

// ErrOverlayRequired 配置中心 required=true 时加载失败
var ErrOverlayRequired = errors.New("configcenter: overlay required but failed")

// KVGet Etcd Get 窄接口（便于单测）
type KVGet interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
}

// LoadOverlay 从 Etcd 拉取整包 YAML 并合并进 app.BaseConfig（etcd > file）。
//
// 【流程】
// 1. enabled=false 或 cc=nil → 直接返回
// 2. Get(key)；失败按 required 处理（failOverlay）
// 3. ENV / CIPHER 预处理
// 4. Unmarshal → MergeBaseConfig → 写回 app.BaseConfig
func LoadOverlay(ctx context.Context, kv KVGet, cc *config.EtcdConfigCenterConfig, cipherKey string) error {
	// 步骤1：开关短路
	if cc == nil || !cc.Enabled {
		return nil
	}
	if kv == nil {
		return failOverlay(cc, fmt.Errorf("configcenter: kv client is nil"))
	}

	key := cc.Prefix
	if key == "" {
		key = config.DefaultConfigCenterPrefix
	}

	// 步骤2：Get
	resp, err := kv.Get(ctx, key)
	if err != nil {
		return failOverlay(cc, fmt.Errorf("configcenter: get %s: %w", key, err))
	}
	if resp == nil || len(resp.Kvs) == 0 {
		return failOverlay(cc, fmt.Errorf("configcenter: key %s not found", key))
	}

	// 步骤3：ENV / CIPHER
	raw, err := prepareYAML(resp.Kvs[0].Value, cipherKey)
	if err != nil {
		return failOverlay(cc, err)
	}

	// 步骤4：合并写回
	var overlay config.BaseConfig
	if err := yaml.Unmarshal(raw, &overlay); err != nil {
		return failOverlay(cc, fmt.Errorf("configcenter: yaml: %w", err))
	}

	dst := app.BaseConfig
	if err := MergeBaseConfig(&dst, &overlay); err != nil {
		return failOverlay(cc, err)
	}
	app.BaseConfig = dst
	metrics.ConfigReloadTotal.WithLabelValues("success").Inc()
	logger.Info("[configcenter] overlay merged from key=%s", key)
	return nil
}

// failOverlay 统一处理启动 overlay 失败：打点 error；required 时包装 ErrOverlayRequired，否则 warn 并返回 nil。
func failOverlay(cc *config.EtcdConfigCenterConfig, err error) error {
	metrics.ConfigReloadTotal.WithLabelValues("error").Inc()
	required := cc != nil && cc.Required != nil && *cc.Required
	if required {
		return fmt.Errorf("%w: %v", ErrOverlayRequired, err)
	}
	logger.Warn("[configcenter] overlay skipped: %v", err)
	return nil
}

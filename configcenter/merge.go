package configcenter

import (
	"fmt"

	"github.com/zzsen/gin_core/model/config"
	"gopkg.in/yaml.v3"
)

// MergeBaseConfig 将 src 深度合并进 dst（src 覆盖冲突字段），并恢复 dst 的 Etcd 连接 bootstrap。
//
// 保护字段：endpoints / username / password / required / keyPrefix / dial / tls，
// 避免 overlay 改写本进程已建立连接所用的连接配置。
//
// 【流程】
// 1. 快照 dst 的 etcd bootstrap
// 2. 结构体转 map，剥离 src 中的 bootstrap 字段后 deepMerge
// 3. map 回写 BaseConfig，再 restore bootstrap
func MergeBaseConfig(dst, src *config.BaseConfig) error {
	if dst == nil || src == nil {
		return fmt.Errorf("configcenter: merge nil BaseConfig")
	}

	// 步骤1：快照 bootstrap
	bootstrap := snapshotEtcdBootstrap(dst.Etcd)

	// 步骤2：map 合并
	dstMap, err := structToMap(dst)
	if err != nil {
		return err
	}
	srcMap, err := structToMap(src)
	if err != nil {
		return err
	}
	stripEtcdBootstrap(srcMap)
	deepMergeMap(dstMap, srcMap)

	// 步骤3：还原结构体并恢复 bootstrap
	merged, err := mapToBaseConfig(dstMap)
	if err != nil {
		return err
	}
	*dst = merged
	restoreEtcdBootstrap(dst, bootstrap)
	return nil
}

// etcdBootstrap 保存合并前本进程 Etcd 连接相关字段。
type etcdBootstrap struct {
	present   bool
	endpoints []string
	username  string
	password  string
	required  *bool
	keyPrefix string
	dial      *config.EtcdDialConfig
	tls       *config.EtcdTLSConfig
}

// snapshotEtcdBootstrap 从 EtcdInfo 复制连接 bootstrap 字段（含 endpoints 切片副本）。
func snapshotEtcdBootstrap(e *config.EtcdInfo) etcdBootstrap {
	if e == nil {
		return etcdBootstrap{}
	}
	eps := append([]string(nil), e.Endpoints...)
	return etcdBootstrap{
		present:   true,
		endpoints: eps,
		username:  e.Username,
		password:  e.Password,
		required:  e.Required,
		keyPrefix: e.KeyPrefix,
		dial:      e.Dial,
		tls:       e.TLS,
	}
}

// restoreEtcdBootstrap 将快照写回 dst.Etcd，覆盖合并结果中的同名字段。
func restoreEtcdBootstrap(dst *config.BaseConfig, b etcdBootstrap) {
	if !b.present {
		return
	}
	if dst.Etcd == nil {
		dst.Etcd = &config.EtcdInfo{}
	}
	dst.Etcd.Endpoints = append([]string(nil), b.endpoints...)
	dst.Etcd.Username = b.username
	dst.Etcd.Password = b.password
	dst.Etcd.Required = b.required
	dst.Etcd.KeyPrefix = b.keyPrefix
	dst.Etcd.Dial = b.dial
	dst.Etcd.TLS = b.tls
}

// stripEtcdBootstrap 从 overlay map 中删除不得覆盖的 etcd 连接字段。
func stripEtcdBootstrap(m map[string]any) {
	raw, ok := m["etcd"]
	if !ok || raw == nil {
		return
	}
	em, ok := raw.(map[string]any)
	if !ok {
		return
	}
	delete(em, "endpoints")
	delete(em, "username")
	delete(em, "password")
	delete(em, "required")
	delete(em, "keyPrefix")
	delete(em, "dial")
	delete(em, "tls")
}

// structToMap 经 YAML 编解码将结构体转为通用 map（便于深度合并）。
func structToMap(v any) (map[string]any, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// mapToBaseConfig 将通用 map 反序列化为 BaseConfig。
func mapToBaseConfig(m map[string]any) (config.BaseConfig, error) {
	b, err := yaml.Marshal(m)
	if err != nil {
		return config.BaseConfig{}, err
	}
	var cfg config.BaseConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return config.BaseConfig{}, err
	}
	return cfg, nil
}

// deepMergeMap 递归合并：双方均为 map 则递归，否则 src 覆盖 dst；src 中 nil 值跳过。
func deepMergeMap(dst, src map[string]any) {
	for k, sv := range src {
		if sv == nil {
			continue
		}
		dv, ok := dst[k]
		srcMap, srcIsMap := sv.(map[string]any)
		dstMap, dstIsMap := dv.(map[string]any)
		if ok && srcIsMap && dstIsMap {
			deepMergeMap(dstMap, srcMap)
			dst[k] = dstMap
			continue
		}
		dst[k] = sv
	}
}

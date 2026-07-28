// Package discovery 提供基于 Etcd 的服务注册与发现（可选，默认关闭）
package discovery

import (
	"encoding/json"
	"fmt"
	"time"
)

// Instance 服务实例元数据（写入 Etcd 的 JSON value）
type Instance struct {
	ServiceName string            `json:"serviceName"`
	InstanceID  string            `json:"instanceID"`
	IP          string            `json:"ip"`
	Port        int               `json:"port"`
	Weight      int               `json:"weight"`
	Meta        map[string]string `json:"meta,omitempty"`
	StartedAt   time.Time         `json:"startedAt,omitempty"`
	Version     string            `json:"version,omitempty"`
}

// MarshalValue 将 Instance 序列化为 Etcd 写入用的 JSON 字节
func (i Instance) MarshalValue() ([]byte, error) {
	return json.Marshal(i)
}

// UnmarshalInstance 从 Etcd value 反序列化为 Instance
func UnmarshalInstance(b []byte) (Instance, error) {
	var inst Instance
	if err := json.Unmarshal(b, &inst); err != nil {
		return Instance{}, err
	}
	return inst, nil
}

// BuildKey 拼装 discovery 注册 Key
//
// 格式：{prefix}{env}/{service}/{instanceID}
// 其中 prefix 一般为 discovery.EffectivePrefix()（可叠在 etcd.keyPrefix namespace 之上）
func BuildKey(prefix, env, service, instanceID string) string {
	return fmt.Sprintf("%s%s/%s/%s", prefix, env, service, instanceID)
}

// DefaultInstanceID 生成默认实例 ID：{hostname}-{port}
func DefaultInstanceID(hostname string, port int) string {
	return fmt.Sprintf("%s-%d", hostname, port)
}

package logger

import "github.com/zzsen/gin_core/model/config"

// EnrichFields 合并业务字段与 resource 公共字段，并按 fieldMap 重命名键。
//
// 【功能】为远程日志补充 serviceName/env/host/version，并支持键名映射（如 serviceName→service）。
// 【流程】
// 1. 复制 base（可为 nil）到新 map，不修改入参
// 2. 写入 resource 中非空字段
// 3. 按 fieldMap 将源键重命名为目标键（存在才改）
//
// 返回：enrich 后的新 map
func EnrichFields(base map[string]any, res *config.LogResourceConfig, fieldMap map[string]string) map[string]any {
	// 1. 复制业务字段
	out := make(map[string]any, len(base)+4)
	for k, v := range base {
		out[k] = v
	}

	// 2. 注入公共资源字段
	if res != nil {
		if res.ServiceName != "" {
			out["serviceName"] = res.ServiceName
		}
		if res.Env != "" {
			out["env"] = res.Env
		}
		if res.Host != "" {
			out["host"] = res.Host
		}
		if res.Version != "" {
			out["version"] = res.Version
		}
	}

	// 3. 字段重命名
	if len(fieldMap) == 0 {
		return out
	}
	for from, to := range fieldMap {
		if from == "" || to == "" || from == to {
			continue
		}
		if v, ok := out[from]; ok {
			out[to] = v
			delete(out, from)
		}
	}
	return out
}

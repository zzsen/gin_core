package logger

import "github.com/zzsen/gin_core/model/config"

// resolveEffectiveOutputs 解析生效的输出列表。
//
// 【功能】将配置中的 outputs 规范为 InitLogger 实际装配清单。
// 【流程】
// 1. nil 或空切片 → 默认 file + stdout（与现网「未配 outputs」行为一致）
// 2. 非空 → 仅保留 OutputEnabled 为 true 的项（enabled 省略视为启用）
func resolveEffectiveOutputs(outputs []config.LogOutputConfig) []config.LogOutputConfig {
	// 1. 未配置：默认本地双写
	if len(outputs) == 0 {
		return []config.LogOutputConfig{
			{Type: "file"},
			{Type: "stdout"},
		}
	}
	// 2. 严格按列表：过滤未启用项
	out := make([]config.LogOutputConfig, 0, len(outputs))
	for _, o := range outputs {
		if config.OutputEnabled(o) {
			out = append(out, o)
		}
	}
	return out
}

// outputFlags 从生效列表推导装配开关
type outputFlags struct {
	file   bool
	stdout bool
	remote bool
}

// flagsFromEffective 扫描生效 outputs 得到 file/stdout/remote 开关
func flagsFromEffective(effective []config.LogOutputConfig) outputFlags {
	var f outputFlags
	for _, o := range effective {
		switch o.Type {
		case "file":
			f.file = true
		case "stdout":
			f.stdout = true
		case "remote":
			f.remote = true
		}
	}
	return f
}

package logger

import (
	"bytes"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/logger/sink"
	"github.com/zzsen/gin_core/model/config"
)

// remoteHook 将 logrus 日志条目异步投递到远程 Sink Pipeline 的 Hook。
// 实现 logrus.Hook：Levels 决定哪些级别触发，Fire 仅做字段 enrichment、序列化与入队，不发起网络 I/O。
type remoteHook struct {
	pipeline *sink.Pipeline            // 异步投递管线
	levels   []logrus.Level            // 生效级别列表；空则等同 AllLevels
	format   logrus.Formatter          // 远程行序列化格式（建议 JSON）
	resource *config.LogResourceConfig // 公共资源字段
	fieldMap map[string]string         // 可选字段重命名
}

// Levels 返回本 Hook 关心的日志级别。
//
// 【流程】
// 1. levels 未配置时返回 logrus.AllLevels
// 2. 否则返回装配时按 minLevel 过滤后的列表
func (h *remoteHook) Levels() []logrus.Level {
	if len(h.levels) == 0 {
		return logrus.AllLevels
	}
	return h.levels
}

// Fire 处理单条 logrus 日志：enrich → 序列化 → 非阻塞入队。
//
// 【功能】将 Entry 转为 sink.FormattedEntry 并 Enqueue，保证热路径不发起 HTTP。
// 【流程】
// 1. pipeline 为空则直接返回
// 2. 复制 Entry.Data，合并 resource / fieldMap
// 3. 用 Formatter 序列化为 Line（失败则回退 Message）
// 4. 补齐时间戳后 Enqueue，始终返回 nil（投递失败由管线计数，不反压调用方）
func (h *remoteHook) Fire(entry *logrus.Entry) error {
	// 1. 无管线则跳过
	if h.pipeline == nil {
		return nil
	}

	// 2. 复制业务字段并注入公共字段
	base := make(map[string]any, len(entry.Data)+1)
	for k, v := range entry.Data {
		base[k] = v
	}
	fields := EnrichFields(base, h.resource, h.fieldMap)

	// 3. 序列化为远程行内容
	var line []byte
	if h.format != nil {
		tmp := &logrus.Entry{
			Logger:  entry.Logger,
			Data:    logrus.Fields{},
			Time:    entry.Time,
			Level:   entry.Level,
			Message: entry.Message,
		}
		for k, v := range fields {
			tmp.Data[k] = v
		}
		b, err := h.format.Format(tmp)
		if err == nil {
			line = bytes.TrimSpace(b)
		}
	}
	if len(line) == 0 {
		line = []byte(entry.Message)
	}

	// 4. 入队（非阻塞）
	ts := entry.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	h.pipeline.Enqueue(sink.FormattedEntry{
		Level:   entry.Level.String(),
		Time:    ts,
		Message: entry.Message,
		Fields:  fields,
		Line:    line,
	})
	return nil
}

// levelsFromMin 根据最低级别名生成 Hook 生效级别列表。
//
// 【功能】解析 minLevel，返回「该级别及以上」（logrus 数值越小越严重，即 lv <= min）。
// 【流程】
// 1. 空或非法 → AllLevels
// 2. 遍历 AllLevels，保留 lv <= min 的级别
func levelsFromMin(minLevel string) []logrus.Level {
	// 1. 未配置或非法：不过滤
	if minLevel == "" {
		return logrus.AllLevels
	}
	min, err := logrus.ParseLevel(minLevel)
	if err != nil {
		return logrus.AllLevels
	}

	// 2. 保留不低于 min 严重度的级别（Panic=0 … Trace=6）
	out := make([]logrus.Level, 0, len(logrus.AllLevels))
	for _, lv := range logrus.AllLevels {
		if lv <= min {
			out = append(out, lv)
		}
	}
	return out
}

// resourceLabelValues 从 resource 配置提取非空字段，供 Loki labels 使用。
//
// 【流程】
// 1. res 为 nil 返回空 map
// 2. 依次写入 serviceName / env / host / version（仅非空）
func resourceLabelValues(res *config.LogResourceConfig) map[string]string {
	m := map[string]string{}
	if res == nil {
		return m
	}
	if res.ServiceName != "" {
		m["serviceName"] = res.ServiceName
	}
	if res.Env != "" {
		m["env"] = res.Env
	}
	if res.Host != "" {
		m["host"] = res.Host
	}
	if res.Version != "" {
		m["version"] = res.Version
	}
	return m
}

// attachRemoteSink 按 log.outputs 装配远程 Sink 与 Hook（首期仅 Loki）。
//
// 【功能】在 InitLogger 末尾调用；无启用 remote 时为空操作。
// 【流程】
// 1. 遍历 outputs，跳过非 remote / 未启用 / 缺少 Remote 配置的项
// 2. 仅接受 driver 为空或 loki，且 loki 子配置存在
// 3. 创建 LokiSink、Pipeline（含重试参数），注册为全局管线
// 4. 向 logrus 添加 remoteHook 后返回（首期只装配第一个启用的 remote）
func attachRemoteSink(logger *logrus.Logger, cfg config.LoggersConfig) {
	for _, o := range cfg.Outputs {
		// 1. 过滤非远程或未启用输出
		if o.Type != "remote" || !config.OutputEnabled(o) || o.Remote == nil {
			continue
		}
		remote := o.Remote

		// 2. 驱动与 Loki 配置校验
		if remote.Driver != "" && remote.Driver != "loki" {
			continue
		}
		if remote.Loki == nil {
			continue
		}

		// 3. 创建 Loki 适配器与异步管线
		ls, err := sink.NewLokiSink(*remote.Loki, resourceLabelValues(cfg.Resource))
		if err != nil {
			logger.Errorf("[logger] init loki sink failed: %v", err)
			continue
		}
		ls.SetMinLevel(o.MinLevel)

		qSize := remote.QueueSize
		batch := remote.BatchSize
		flushMs := remote.FlushIntervalMs
		maxAttempts := 1
		var backoff time.Duration
		if remote.Retry != nil {
			if remote.Retry.MaxAttempts > 0 {
				maxAttempts = remote.Retry.MaxAttempts
			}
			if remote.Retry.BackoffMs > 0 {
				backoff = time.Duration(remote.Retry.BackoffMs) * time.Millisecond
			}
		}
		p := sink.NewPipeline(ls, sink.PipelineConfig{
			QueueSize:     qSize,
			BatchSize:     batch,
			FlushInterval: time.Duration(flushMs) * time.Millisecond,
			MaxAttempts:   maxAttempts,
			Backoff:       backoff,
		})

		// 4. 注册全局管线与 Hook
		SetRemotePipeline(p)
		fmtName := resolveFormat(cfg.Format, remote.Format)
		logger.AddHook(&remoteHook{
			pipeline: p,
			levels:   levelsFromMin(o.MinLevel),
			format:   newFormatter(fmtName),
			resource: cfg.Resource,
			fieldMap: cfg.FieldMap,
		})
		return // 首期仅装配第一个启用的 remote
	}
}

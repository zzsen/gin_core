package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/zzsen/gin_core/model/config"
)

// LokiSink Loki Push HTTP 适配器，实现 Sink 接口。
// 将 FormattedEntry 批量编码为 Loki `/loki/api/v1/push` JSON 并 POST。
type LokiSink struct {
	url         string
	client      *http.Client
	labels      map[string]string
	bearerToken string
	basicUser   string
	basicPass   string
	minLevel    string
}

// NewLokiSink 根据配置创建 Loki Sink。
//
// 【参数】
//   - cfg：URL / 超时 / labels 白名单 / 鉴权
//   - labelValues：静态 label 取值（通常来自 log.resource）
//
// 【流程】
// 1. 校验 URL 非空
// 2. 解析超时，默认 3s
// 3. 按 Labels 白名单从 labelValues 取标签；皆空则回退全部非空值，再否则 job=gin_core
// 4. 构造带超时的 http.Client
func NewLokiSink(cfg config.LokiOutputConfig, labelValues map[string]string) (*LokiSink, error) {
	// 1. URL 必填
	if cfg.URL == "" {
		return nil, fmt.Errorf("loki url is required")
	}

	// 2. 超时
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	// 3. 组装 labels
	labels := make(map[string]string)
	for _, k := range cfg.Labels {
		if v, ok := labelValues[k]; ok && v != "" {
			labels[k] = v
		}
	}
	if len(labels) == 0 && len(labelValues) > 0 {
		for k, v := range labelValues {
			if v != "" {
				labels[k] = v
			}
		}
	}
	if len(labels) == 0 {
		labels["job"] = "gin_core"
	}

	// 4. 返回实例
	return &LokiSink{
		url:         cfg.URL,
		client:      &http.Client{Timeout: timeout},
		labels:      labels,
		bearerToken: cfg.BearerToken,
		basicUser:   cfg.BasicUser,
		basicPass:   cfg.BasicPassword,
	}, nil
}

// Name 返回适配器名称。
func (s *LokiSink) Name() string { return "loki" }

// MinLevel 返回装配层写入的最低级别名。
func (s *LokiSink) MinLevel() string { return s.minLevel }

// SetMinLevel 设置最低级别过滤名（由 attachRemoteSink 写入，供观测/扩展）。
func (s *LokiSink) SetMinLevel(level string) { s.minLevel = level }

// Close 关闭 Loki Sink；当前无长连接，恒返回 nil。
func (s *LokiSink) Close(context.Context) error { return nil }

// lokiPushBody Loki push API 请求体。
type lokiPushBody struct {
	Streams []lokiStream `json:"streams"`
}

// lokiStream 单条 stream：labels + [timestamp_ns, line] 列表。
type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// Send 批量推送日志到 Loki Push API。
//
// 【流程】
// 1. entries 为空直接返回
// 2. 将每条转为 [nanosecond, line]，优先使用 Line
// 3. 组装 streams JSON 并 POST
// 4. 附加 Bearer 或 Basic Auth（若配置）
// 5. 非 2xx 返回 error，供 Pipeline 重试
func (s *LokiSink) Send(ctx context.Context, entries []FormattedEntry) error {
	// 1. 空批次
	if len(entries) == 0 {
		return nil
	}

	// 2. 编码 values
	values := make([][]string, 0, len(entries))
	for _, e := range entries {
		ts := e.Time
		if ts.IsZero() {
			ts = time.Now()
		}
		line := string(e.Line)
		if line == "" {
			line = e.Message
		}
		values = append(values, []string{strconv.FormatInt(ts.UnixNano(), 10), line})
	}

	// 3. 序列化请求体
	body := lokiPushBody{Streams: []lokiStream{{Stream: s.labels, Values: values}}}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// 4. 鉴权
	if s.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
	} else if s.basicUser != "" {
		req.SetBasicAuth(s.basicUser, s.basicPass)
	}

	// 5. 发送并校验状态码
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("loki push status %d", resp.StatusCode)
	}
	return nil
}

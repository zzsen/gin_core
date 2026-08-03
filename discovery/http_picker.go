package discovery

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPPicker 基于 InstancePicker 发起 HTTP 请求（不修改 utils/http_client）。
type HTTPPicker struct {
	picker   InstancePicker
	client   *http.Client
	strategy string
}

// HTTPOption HTTPPicker 可选配置
type HTTPOption func(*HTTPPicker)

// WithHTTPClient 自定义 http.Client
func WithHTTPClient(c *http.Client) HTTPOption {
	return func(h *HTTPPicker) { h.client = c }
}

// WithPickStrategy 设置默认 Pick 策略（round_robin / random）
func WithPickStrategy(strategy string) HTTPOption {
	return func(h *HTTPPicker) { h.strategy = strategy }
}

// WithHTTPTimeout 设置请求超时（覆盖 client.Timeout）
func WithHTTPTimeout(d time.Duration) HTTPOption {
	return func(h *HTTPPicker) {
		if h.client == nil {
			h.client = &http.Client{}
		}
		cp := *h.client
		cp.Timeout = d
		h.client = &cp
	}
}

// NewHTTPPicker 创建 HTTPPicker（默认 Timeout=10s、strategy=round_robin）
func NewHTTPPicker(p InstancePicker, opts ...HTTPOption) *HTTPPicker {
	h := &HTTPPicker{
		picker:   p,
		client:   &http.Client{Timeout: 10 * time.Second},
		strategy: "round_robin",
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Do 选择实例并请求 http://{ip}:{port}{path}
//
// 【功能】按服务名 Pick 实例后发起 HTTP；无实例时立即返回错误（不阻塞）
// 【流程】
//  1. 校验 HTTPPicker / picker
//  2. Pick 实例（失败直接返回，含 ErrNoInstances）
//  3. 规范化 path 并拼装 URL
//  4. 构造带 ctx 的 Request，拷贝 Header
//  5. client.Do 发起请求
func (h *HTTPPicker) Do(ctx context.Context, service, method, path string, body io.Reader, header http.Header) (*http.Response, error) {
	// 步骤 1：校验
	if h == nil || h.picker == nil {
		return nil, fmt.Errorf("discovery: HTTPPicker not configured")
	}

	// 步骤 2：选取实例
	inst, err := h.picker.Pick(service, h.strategy)
	if err != nil {
		return nil, err
	}

	// 步骤 3：拼装 URL
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := fmt.Sprintf("http://%s:%d%s", inst.IP, inst.Port, path)

	// 步骤 4：构造请求
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header.Clone()
	}

	// 步骤 5：发送
	return h.client.Do(req)
}

// DoWait 等待可用实例后再发起 HTTP（空列表可阻塞至 ctx 结束）
//
// picker 须实现 WaitPicker；否则返回明确错误。Do 语义不变（立即失败）。
func (h *HTTPPicker) DoWait(ctx context.Context, service, method, path string, body io.Reader, header http.Header) (*http.Response, error) {
	if h == nil || h.picker == nil {
		return nil, fmt.Errorf("discovery: HTTPPicker not configured")
	}
	wp, ok := h.picker.(WaitPicker)
	if !ok {
		return nil, fmt.Errorf("discovery: picker does not support PickWait")
	}
	inst, err := wp.PickWait(ctx, service, h.strategy)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url := fmt.Sprintf("http://%s:%d%s", inst.IP, inst.Port, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header.Clone()
	}
	return h.client.Do(req)
}

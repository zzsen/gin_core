// Package http_client 提供高性能的 HTTP 客户端工具
// 支持连接池复用、链路追踪、请求重试等功能
package http_client

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/zzsen/gin_core/tracing"
)

// ClientConfig HTTP 客户端配置
type ClientConfig struct {
	// 连接池配置
	MaxIdleConns        int           // 最大空闲连接数，默认 100
	MaxIdleConnsPerHost int           // 每个主机最大空闲连接数，默认 10
	MaxConnsPerHost     int           // 每个主机最大连接数，默认 100
	IdleConnTimeout     time.Duration // 空闲连接超时时间，默认 90s

	// 超时配置
	DialTimeout         time.Duration // 连接建立超时，默认 30s
	TLSHandshakeTimeout time.Duration // TLS 握手超时，默认 10s
	ResponseTimeout     time.Duration // 响应头超时，默认 30s

	// 重试配置
	MaxRetries    int           // 最大重试次数，默认 3
	RetryInterval time.Duration // 重试间隔，默认 100ms

	// 功能开关
	EnableTracing bool // 是否启用链路追踪，默认 true
}

// DefaultClientConfig 返回默认的客户端配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		DialTimeout:         30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseTimeout:     30 * time.Second,
		MaxRetries:          3,
		RetryInterval:       100 * time.Millisecond,
		EnableTracing:       true,
	}
}

// Client 高性能 HTTP 客户端
// 支持连接池复用、链路追踪、请求重试
type Client struct {
	httpClient *http.Client
	config     *ClientConfig
}

var (
	// defaultClient 默认的全局客户端实例
	defaultClient *Client
	clientOnce    sync.Once
)

// GetDefaultClient 获取默认的全局 HTTP 客户端
// 使用单例模式，线程安全
func GetDefaultClient() *Client {
	clientOnce.Do(func() {
		defaultClient = NewClient(nil)
	})
	return defaultClient
}

// NewClient 创建新的 HTTP 客户端
// 参数：
//   - config: 客户端配置，为 nil 时使用默认配置
//
// 返回：
//   - *Client: HTTP 客户端实例
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	// 创建自定义的 Transport
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseTimeout,
		ForceAttemptHTTP2:     true,
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Transport: transport,
	}

	// 如果启用链路追踪，包装 Transport
	if config.EnableTracing {
		httpClient = tracing.WrapHTTPClient(httpClient)
	}

	return &Client{
		httpClient: httpClient,
		config:     config,
	}
}

// Do 执行 HTTP 请求（带重试）
// 参数：
//   - ctx: 上下文，用于超时控制和取消
//   - req: HTTP 请求对象
//
// 返回：
//   - *http.Response: HTTP 响应
//   - error: 错误信息
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 设置请求上下文
	req = req.WithContext(ctx)

	// 重试逻辑
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		resp, err = c.httpClient.Do(req)

		// 请求成功或不可重试的错误，直接返回
		if err == nil {
			return resp, nil
		}

		// 检查是否可重试
		if !isRetryableError(err) {
			return nil, err
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 最后一次尝试不需要等待
		if attempt < c.config.MaxRetries {
			time.Sleep(c.config.RetryInterval)
		}
	}

	return nil, err
}

// GetHTTPClient 获取底层的 http.Client
// 用于需要直接操作 http.Client 的场景
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// isRetryableError 判断错误是否可重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 网络超时错误可重试
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// 连接被拒绝可重试
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Op == "dial" {
			return true
		}
	}

	return false
}

// CloseIdleConnections 关闭所有空闲连接
// 用于优雅关闭时释放资源
func (c *Client) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}

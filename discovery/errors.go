package discovery

import "errors"

// ErrWaitTimeout 等待可用实例超时（可与 context.DeadlineExceeded 组合 unwrap）
var ErrWaitTimeout = errors.New("discovery: wait timeout")

// ErrResolverStopped Resolver 已 Stop，等待/订阅不可继续
var ErrResolverStopped = errors.New("discovery: resolver stopped")

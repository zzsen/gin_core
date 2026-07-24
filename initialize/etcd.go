// Package initialize 提供各种服务的初始化功能
// 本文件专门负责 Etcd 客户端的初始化配置
package initialize

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/namespace"
)

// ErrEtcdConfig 表示 Etcd 配置校验失败（不可降级，无论 required）
var ErrEtcdConfig = errors.New("etcd config invalid")

// InitEtcd 初始化 Etcd 客户端。
//
// 流程：
// 1. 校验配置（nil / endpoints / dial / health strategy / tls 文件）
// 2. 组装 clientv3.Config（拨号、KeepAlive、TLS）
// 3. 创建客户端；keyPrefix 非空时包装 KV/Watcher/Lease
// 4. 写入 app.Etcd 并打脱敏日志
func InitEtcd() error {
	cfg := app.BaseConfig.Etcd
	if cfg == nil {
		return fmt.Errorf("%w: 未找到Etcd配置", ErrEtcdConfig)
	}
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("%w: etcd.endpoints 不能为空", ErrEtcdConfig)
	}
	for _, ep := range cfg.Endpoints {
		if strings.TrimSpace(ep) == "" {
			return fmt.Errorf("%w: etcd.endpoints 含空地址", ErrEtcdConfig)
		}
	}
	strategy := cfg.HealthStrategy()
	if strategy != "any" && strategy != "all" {
		logger.Warn("[Etcd] 不支持的 etcd.health.strategy=%q，回退为默认 any", strategy)
	}

	timeoutSec, keepAliveSec, keepAliveTimeoutSec, autoSyncSec, permitWithoutStream, err := resolveEtcdDial(cfg.Dial)
	if err != nil {
		return err
	}

	tlsConfig, err := buildEtcdTLS(cfg.TLS)
	if err != nil {
		return err
	}

	cliCfg := clientv3.Config{
		Endpoints:            cfg.Endpoints,
		Username:             cfg.Username,
		Password:             cfg.Password,
		DialTimeout:          time.Duration(timeoutSec) * time.Second,
		DialKeepAliveTime:    time.Duration(keepAliveSec) * time.Second,
		DialKeepAliveTimeout: time.Duration(keepAliveTimeoutSec) * time.Second,
		PermitWithoutStream:  permitWithoutStream,
		TLS:                  tlsConfig,
	}
	if autoSyncSec > 0 {
		cliCfg.AutoSyncInterval = time.Duration(autoSyncSec) * time.Second
	}

	cli, err := clientv3.New(cliCfg)
	if err != nil {
		return fmt.Errorf("创建etcd客户端失败: %w", err)
	}

	// ready 探测：New 对不可达地址可能懒成功，用 Status 确认连通性
	readyCtx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	_, statusErr := cli.Status(readyCtx, cfg.Endpoints[0])
	cancel()
	if statusErr != nil {
		_ = cli.Close()
		return fmt.Errorf("etcd ready 检查失败: %w", statusErr)
	}

	if p := strings.TrimSpace(cfg.KeyPrefix); p != "" {
		cli.KV = namespace.NewKV(cli.KV, p)
		cli.Watcher = namespace.NewWatcher(cli.Watcher, p)
		cli.Lease = namespace.NewLease(cli.Lease, p)
	}

	app.Etcd = cli
	logger.Info("[etcd] etcd已初始化 endpoints=%v keyPrefix=%q required=%v health=%s",
		cfg.Endpoints, cfg.KeyPrefix, cfg.IsRequired(), strategy)
	return nil
}

// resolveEtcdDial 解析拨号秒数；负数返回 ErrEtcdConfig
func resolveEtcdDial(dial *config.EtcdDialConfig) (timeout, keepAlive, keepAliveTimeout, autoSync int, permitWithoutStream bool, err error) {
	timeout = constant.DefaultEtcdTimeout
	keepAlive = constant.DefaultEtcdKeepAliveTime
	keepAliveTimeout = constant.DefaultEtcdKeepAliveTimeout
	autoSync = 0
	permitWithoutStream = false

	if dial == nil {
		return timeout, keepAlive, keepAliveTimeout, autoSync, permitWithoutStream, nil
	}
	if dial.Timeout != nil {
		if *dial.Timeout < 0 {
			return 0, 0, 0, 0, false, fmt.Errorf("%w: etcd.dial.timeout 不能为负", ErrEtcdConfig)
		}
		if *dial.Timeout > 0 {
			timeout = *dial.Timeout
		}
	}
	if dial.KeepAliveTime != nil {
		if *dial.KeepAliveTime < 0 {
			return 0, 0, 0, 0, false, fmt.Errorf("%w: etcd.dial.keepAliveTime 不能为负", ErrEtcdConfig)
		}
		keepAlive = *dial.KeepAliveTime
	}
	if dial.KeepAliveTimeout != nil {
		if *dial.KeepAliveTimeout < 0 {
			return 0, 0, 0, 0, false, fmt.Errorf("%w: etcd.dial.keepAliveTimeout 不能为负", ErrEtcdConfig)
		}
		keepAliveTimeout = *dial.KeepAliveTimeout
	}
	if dial.AutoSyncInterval != nil {
		if *dial.AutoSyncInterval < 0 {
			return 0, 0, 0, 0, false, fmt.Errorf("%w: etcd.dial.autoSyncInterval 不能为负", ErrEtcdConfig)
		}
		autoSync = *dial.AutoSyncInterval
	}
	if dial.PermitWithoutStream != nil {
		permitWithoutStream = *dial.PermitWithoutStream
	}
	return timeout, keepAlive, keepAliveTimeout, autoSync, permitWithoutStream, nil
}

// buildEtcdTLS 根据配置构造 TLS；enabled=false 时返回 nil
func buildEtcdTLS(tlsCfg *config.EtcdTLSConfig) (*tls.Config, error) {
	if tlsCfg == nil || !tlsCfg.Enabled {
		return nil, nil
	}
	out := &tls.Config{
		InsecureSkipVerify: tlsCfg.InsecureSkipVerify, //nolint:gosec // 由配置显式控制
		MinVersion:         tls.VersionTLS12,
	}
	if tlsCfg.CAFile != "" {
		pem, err := os.ReadFile(tlsCfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("%w: 读取 etcd.tls.caFile 失败: %v", ErrEtcdConfig, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("%w: 解析 etcd.tls.caFile 失败", ErrEtcdConfig)
		}
		out.RootCAs = pool
	}
	if tlsCfg.CertFile != "" || tlsCfg.KeyFile != "" {
		if tlsCfg.CertFile == "" || tlsCfg.KeyFile == "" {
			return nil, fmt.Errorf("%w: etcd.tls.certFile 与 keyFile 需同时配置", ErrEtcdConfig)
		}
		cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: 加载客户端证书失败: %v", ErrEtcdConfig, err)
		}
		out.Certificates = []tls.Certificate{cert}
	}
	return out, nil
}

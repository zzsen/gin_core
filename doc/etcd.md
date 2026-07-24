# Etcd 客户端

本文描述 `gin_core` 内置 Etcd **客户端基建**能力。完整配置字段见 [配置 §5.12](./config.md#512-etcd-客户端-etcd)。

## 能力矩阵

| 能力 | 状态 | 说明 |
|------|------|------|
| 分层配置 | ✅ | `endpoints` / `dial` / `tls` / `health` / `required` / `keyPrefix` |
| 初始化 | ✅ | `initialize.InitEtcd() error`；成功写入 `app.Etcd` |
| ready 探测 | ✅ | 创建客户端后对首个 endpoint 做 `Status` |
| TLS | ✅ | `tls.enabled` + ca/cert/key / insecureSkipVerify |
| Namespace | ✅ | `keyPrefix` 非空时包装 KV / Watcher / Lease |
| 健康检查 | ✅ | `health.strategy`: `any` \| `all`（默认 `any`） |
| 分布式锁 | ✅ | `distlock.NewEtcdLocker(app.Etcd, …)`，见 [分布式锁](./distlock.md) |
| 服务注册 / 发现 | ❌ | 未内置 |
| 配置中心 / 热更新 | ❌ | 未内置；本地仍用 `conf/*.yml` |

## 启用

```yaml
system:
  useEtcd: true

etcd:
  endpoints:
    - "http://127.0.0.1:2379"
  username: ""
  password: ""
  required: false          # 生产建议 true
  keyPrefix: ""
  dial:
    timeout: 5             # 秒
    keepAliveTime: 30
    keepAliveTimeout: 10
    autoSyncInterval: 0    # 0=关闭
  tls:
    enabled: false
  health:
    strategy: any          # any | all
```

## required 语义

| `required` | 连接 / ready 失败时 |
|------------|---------------------|
| `false`（默认） | 打 warn，`app.Etcd=nil`，应用继续启动 |
| `true` | `EtcdService.Init` 返回 error，阻断启动 |

配置校验失败（如空 `endpoints`、负数 timeout、坏 TLS 文件）**始终**返回错误，不受 `required` 影响。非法 `health.strategy` 打 warn 后回退默认 `any`，不阻断。

## keyPrefix / Namespace

- 非空：对 `app.Etcd` 的 KV、Watcher、Lease 使用 `clientv3/namespace` 包装；**Status / Maintenance 不包装**（探活仍用真实 endpoint）。
- 空：不包装。
- 使用 `distlock` 且设置了 `keyPrefix` 时，锁相关 key 会落在该前缀下。

## 健康检查

调用链：健康检查引擎 → [`EtcdService.HealthCheck`](../core/services/etcd_service.go)

| strategy | 行为 |
|----------|------|
| `any`（默认） | 任一 endpoint `Status` 成功即健康 |
| `all` | 全部成功才健康 |
| 其他（含 `first`） | warn 后回退 `any` 继续探活 |

## 调用链（初始化）

[`core.Start`](../core/server.go) → 服务初始化 → [`EtcdService.Init`](../core/services/etcd_service.go) → [`initialize.InitEtcd`](../initialize/etcd.go) → `app.Etcd`

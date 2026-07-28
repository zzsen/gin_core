# Etcd 客户端与服务发现

本文描述 `gin_core` 内置 Etcd **客户端基建**与可选 **服务注册 / 发现**。完整配置字段见 [配置 §5.12](./config.md#512-etcd-客户端-etcd)。

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
| 服务注册 / 发现 | ✅ | opt-in：`etcd.discovery.enabled`；包 `discovery/` |
| HTTP 按服务名调用 | ✅ | `discovery.HTTPPicker.Do`（不改 `http_client`） |
| 配置中心 / 热更新 | ❌ | 未内置；本地仍用 `conf/*.yml` |
| 健康摘除注册 | ❌ | 本波未做 |

## 启用客户端

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

## 启用服务发现（opt-in）

`useEtcd: true` **不会**自动开启发现。需显式：

```yaml
etcd:
  discovery:
    enabled: true
    register: true
    serviceName: "user-svc"   # 必填
    env: "dev"                # 空则实现侧用 default（或后续对接框架环境名）
    prefix: "services/"
    ttlSeconds: 30
    instanceID: ""            # 空则 {hostname}-{port}
    weight: 1
    # advertiseIP / advertisePort 默认取 service.ip / service.port
```

| 行为 | 说明 |
|------|------|
| 注册时机 | [`AppOnReady`](../core/lifecycle/interface.go)（端口已绑定） |
| 注销时机 | [`AppBeforeShutdown`](../core/lifecycle/interface.go) |
| 注册失败 | **仅 warn**，不阻断进程 |
| Key | `{prefix}{env}/{serviceName}/{instanceID}`，再叠 `etcd.keyPrefix` namespace |
| Lease | 独立于 `distlock` Session |

### 最小调用示例

```go
import (
    "context"
    "net/http"

    "github.com/zzsen/gin_core/app"
    "github.com/zzsen/gin_core/discovery"
    "github.com/zzsen/gin_core/model/config"
)

func example(ctx context.Context) error {
    dcfg := app.BaseConfig.Etcd.Discovery
    res := discovery.NewResolver(app.Etcd, dcfg.EffectivePrefix(), dcfg.Env)
    if err := res.Start(ctx); err != nil {
        return err
    }
    defer res.Stop()

    // 查询 / 负载均衡
    _ = res.GetInstances("user-svc")
    inst, err := res.Pick("user-svc", "round_robin") // 或 "random"
    if err != nil {
        return err
    }
    _ = inst

    // 按服务名 HTTP（独立 Picker，非 http_client）
    picker := discovery.NewHTTPPicker(res)
    resp, err := picker.Do(ctx, "user-svc", http.MethodGet, "/api/v1/ping", nil, nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}

// 配置侧请保证 discovery.enabled + serviceName；自动注册由框架钩子完成。
var _ = config.EtcdDiscoveryConfig{}
```

## required 语义

| `required` | 连接 / ready 失败时 |
|------------|---------------------|
| `false`（默认） | 打 warn，`app.Etcd=nil`，应用继续启动 |
| `true` | `EtcdService.Init` 返回 error，阻断启动 |

配置校验失败（如空 `endpoints`、负数 timeout、坏 TLS 文件）**始终**返回错误，不受 `required` 影响。非法 `health.strategy` 打 warn 后回退默认 `any`，不阻断。

## keyPrefix / Namespace

- 非空：对 `app.Etcd` 的 KV、Watcher、Lease 使用 `clientv3/namespace` 包装；**Status / Maintenance 不包装**。
- 空：不包装。
- `discovery` 与 `distlock` 的业务前缀会落在该 namespace 之下。

## 健康检查

调用链：健康检查引擎 → [`EtcdService.HealthCheck`](../core/services/etcd_service.go)

| strategy | 行为 |
|----------|------|
| `any`（默认） | 任一 endpoint `Status` 成功即健康 |
| `all` | 全部成功才健康 |
| 其他（含 `first`） | warn 后回退 `any` 继续探活 |

## 调用链

**客户端**：[`core.Start`](../core/server.go) → [`EtcdService.Init`](../core/services/etcd_service.go) → [`discovery.InstallHooks`](../discovery/hooks.go) + [`initialize.InitEtcd`](../initialize/etcd.go) → `app.Etcd`

**注册**：listen 成功 → `AppOnReady` → [`Registry.Register`](../discovery/registry.go)

**注销**：信号关闭 → `AppBeforeShutdown` → [`Registry.Deregister`](../discovery/registry.go)

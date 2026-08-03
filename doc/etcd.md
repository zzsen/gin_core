# Etcd 客户端、服务发现与配置中心

本文描述 `gin_core` 内置 Etcd **客户端基建**、可选 **服务注册 / 发现** 与可选 **配置中心**。完整配置字段见 [配置 §5.12](./config.md#512-etcd-客户端-etcd)。

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
| KeepAlive 重建 | ✅ | `discovery.keepalive.rebuild`（默认 true）；断流退避重建 Lease |
| 健康两阶段摘除 | ✅ | `discovery.health.unlink`（默认 false）；对接 `/healthy/ready`：先 `weight=0` 再注销 |
| Pick 策略 | ✅ | `round_robin` / `random`（等权）+ `weighted_random`；跳过 `weight<=0` |
| Subscribe 快照推送 | ✅ | `Resolver.Subscribe`；buffer=1 丢旧保新；`cancel`/`Stop` 清理 |
| 空列表等待 | ✅ | `GetWait` / `PickWait` / `HTTPPicker.DoWait`（`context` 可取消/超时）；`Pick`/`Do` 仍立即失败 |
| Discovery 指标 | ✅ | `discovery_keepalive_rebuild_total` / `discovery_health_*`（Prometheus） |
| 配置中心 overlay / 白名单热更 | ✅ | opt-in：`etcd.configCenter.enabled`；包 `configcenter/`；默认关闭 |
| 缓存快照落盘降级 | ❌ | 未做 |

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
    keepalive:
      rebuild: true           # KeepAlive 断流后重建
      maxBackoffSeconds: 30
    health:
      unlink: false           # true 时启用 ready 两阶段摘除
      intervalSeconds: 5
      failThreshold: 3
      successThreshold: 2
    waitTimeoutSeconds: 0     # >0 时 config.ContextWithDiscoveryWaitTimeout 可套超时；不改变 Pick/Do
```

| 行为 | 说明 |
|------|------|
| 注册时机 | [`AppOnReady`](../core/lifecycle/interface.go)（端口已绑定） |
| 注销时机 | [`AppBeforeShutdown`](../core/lifecycle/interface.go) |
| 注册失败 | **仅 warn**，不阻断进程 |
| Key | `{prefix}{env}/{serviceName}/{instanceID}`，再叠 `etcd.keyPrefix` namespace |
| Lease | 独立于 `distlock` Session；可按 `keepalive.rebuild` 重建 |
| 健康摘除 | `health.unlink=true` 时周期性调用与 `/healthy/ready` 同源的就绪聚合；失败达阈值先 `weight=0`，再 `Deregister` |
| Pick / Do | 空列表 **立即** `ErrNoInstances`（不阻塞） |
| Subscribe | `Resolver.Subscribe(service)` → `<-chan []Instance` + cancel；buffer=1 丢旧保新 |
| Wait | `GetWait` / `PickWait` / `HTTPPicker.DoWait`：等正权重实例；超时可 `errors.Is(err, ErrWaitTimeout)` |

### Subscribe / Wait 调用要点

```go
ch, cancel := res.Subscribe("user-svc")
defer cancel()

ctx, cancelWait := context.WithTimeout(context.Background(), 3*time.Second)
defer cancelWait()
inst, err := res.PickWait(ctx, "user-svc", "round_robin")
// 或：picker.DoWait(ctx, "user-svc", http.MethodGet, "/ping", nil, nil)
```

## 启用配置中心（opt-in）

`useEtcd: true` **不会**自动开启配置中心。需显式 `etcd.configCenter.enabled: true`。本地 `conf/*.yml` 仍为 bootstrap 与主路径；Etcd 为 overlay（整包 YAML，单 key）。

前置条件：`system.useEtcd: true` 且 Etcd 客户端可用（`app.Etcd != nil`）。若仅开启 `configCenter` 而客户端未就绪：`required=false` 时 warn 跳过；`required=true` 时阻断启动。

```yaml
etcd:
  configCenter:
    enabled: true
    prefix: "config/app.yml"   # Etcd key，整包 YAML
    required: false            # Get/解析失败是否阻断启动
    debounceMs: 300
    watch: true                # enabled 时默认 true；可显式 false 仅做启动 overlay
    # whitelist:               # 省略或空 → 内置默认
    #   - "log.level"          # → log.loggers[].level + logger.SetLevel
    #   - "rateLimit.*"        # → 整体替换 rateLimit
```

### 配置项说明

| 字段 | 类型 | 默认值 | 含义 |
|------|------|--------|------|
| `enabled` | bool | `false` | 总开关。`false` / 省略整块：不 Get、不 Watch、不改本地配置行为 |
| `prefix` | string | `"config/app.yml"` | overlay 的 Etcd **完整 key**（非目录前缀）；值为整包 YAML。若配置了 `etcd.keyPrefix`，该 key 落在 namespace 之下 |
| `required` | bool | `false`（省略同 false） | 启动时 Get / 解析 / 合并失败是否阻断。`false`：warn 后跳过 overlay 继续启动；`true`：`ConfigCenterService.Init` 返回 error |
| `debounceMs` | int | `300` | Watch 防抖毫秒。`<=0` 或未写时回退 `300`，避免连续 Put 触发风暴式热更 |
| `watch` | bool | `true`（`enabled=true` 且字段省略时） | 是否在启动 overlay 成功后启动 Watch。显式 `false`：仅启动合并，不热更 |
| `whitelist` | string[] | `["log.level", "rateLimit.*"]` | 运行期热更允许的字段模式。省略或空数组 → 使用内置默认；自定义时**完全替换**默认列表（需自行列出要热更的项） |

#### `whitelist` 约定（当前实现）

| 模式 | 热更行为 |
|------|----------|
| `log.level` | 取 overlay 中第一个非空 `log.loggers[].level`，写入本地全部 logger level，并调用 `logger.SetLevel` |
| `rateLimit` / `rateLimit.*` | 用 overlay 的整块 `rateLimit` 覆盖 `app.BaseConfig.RateLimit`（中间件按请求读配置） |
| 其他路径 | 当前版本**不会**应用（即使写在 whitelist 里也无效果）；冷字段（DSN、监听端口等）运行期忽略 |

> 启动 overlay 会对白名单外的可合并字段生效（etcd > file），从而影响后续服务 `Init`；运行期热更**仅**走 whitelist。

### 行为摘要

| 行为 | 说明 |
|------|------|
| 启动合并 | lifecycle 服务 `configcenter`（Priority 25，依赖 `etcd`）；`mysql` / `redis` / `elasticsearch` / `rabbitmq` 声明依赖 `configcenter`（未启用时被拓扑过滤）；在存储类服务前 `LoadOverlay`；**etcd > file** |
| 保护字段 | 不覆盖本进程已用的 `etcd.endpoints` / credentials / dial / tls |
| 热更 | Watch + 防抖；仅白名单；非法 YAML **reject** 并保留旧运行态；指标 `result=reject` |
| 冷字段 | 端口、DSN 等运行期忽略（仅启动 overlay 可影响后续 `init`） |
| 指标 | `config_reload_total{result=success\|reject\|error}` |
| ENV / CIPHER | 与本地配置相同：`{{ENV}}` / `CIPHER()`，密钥来自 `-cipherKey`（`app.CipherKey`） |

依赖边（启用时）：`logger` → `etcd` → `configcenter` → 存储类服务（`mysql` / `redis` / …）。

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
    inst, err := res.Pick("user-svc", "round_robin") // 或 "random" / "weighted_random"
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

**配置中心（opt-in）**：`etcd` 就绪后 → [`ConfigCenterService.Init`](../core/services/configcenter_service.go) → [`configcenter.LoadOverlay`](../configcenter/load.go)（可选 [`StartWatch`](../configcenter/watch.go)）→ 再初始化依赖 `configcenter` 的存储类服务

**注册**：listen 成功 → `AppOnReady` → [`Registry.Register`](../discovery/registry.go)

**注销**：信号关闭 → `AppBeforeShutdown` → [`Registry.Deregister`](../discovery/registry.go)

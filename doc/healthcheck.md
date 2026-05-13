# 健康检查 (Health Check)

## 一、概述

框架内置了健康检查端点，提供存活检查（Liveness）、就绪检查（Readiness）和连接池统计信息，便于负载均衡器、Kubernetes 探针和监控系统检测服务状态。

## 二、端点

| 路径 | 说明 | 适用场景 |
|------|------|----------|
| `GET /healthy` | 存活检查，始终返回 200 | Kubernetes `livenessProbe` |
| `GET /healthy/ready` | 就绪检查，校验所有依赖服务 | Kubernetes `readinessProbe` |
| `GET /healthy/stats` | 连接池统计信息 | 运维监控、性能分析 |

> 如果配置了路由前缀（`service.routePrefix`），健康检查路径会自动添加前缀。例如前缀为 `/api/v1` 时，路径变为 `/api/v1/healthy`。

## 三、存活检查

调用链：[healthDetectEngine](../core/engine.go) → `GET /healthy`

```json
{
  "code": 20000,
  "msg": "healthy",
  "data": {
    "status": "healthy"
  }
}
```

存活检查仅确认服务进程正在运行，不检测依赖服务。

## 四、就绪检查

调用链：[healthDetectEngine](../core/engine.go) → `GET /healthy/ready` → [CheckPoolHealth](../app/pool_stats.go) → [IsAllHealthy](../app/pool_stats.go)

### 4.1 所有服务健康（200）

```json
{
  "code": 20000,
  "msg": "ready",
  "data": {
    "status": "ready",
    "services": {
      "mysql": { "healthy": true, "stats": { "max_open": 100, "open": 5, "in_use": 2, "idle": 3 } },
      "redis": { "healthy": true, "stats": { "total": 10, "idle": 8 } }
    }
  }
}
```

### 4.2 部分服务异常（503）

```json
{
  "code": 50300,
  "msg": "not ready",
  "data": {
    "status": "not ready",
    "services": {
      "mysql": { "healthy": true },
      "redis": { "healthy": false, "error": "connection refused" }
    }
  }
}
```

### 4.3 检查范围

[CheckPoolHealth](../app/pool_stats.go) 会检查以下服务（仅当配置启用且实例存在时）：

| 服务 | 键名格式 | 检查方式 |
|------|----------|----------|
| 主数据库 | `mysql` | `sqlDB.PingContext` + 连接池使用率告警 |
| 数据库解析器 | `mysql_resolver` | 同上 |
| 多数据库列表 | `mysql:<aliasName>` | 同上 |
| 主 Redis | `redis` | `client.Ping` |
| 多 Redis 列表 | `redis:<aliasName>` | 同上 |
| Elasticsearch | `elasticsearch` | `ES.Info().Do` |
| Etcd | `etcd` | `Etcd.Status` |

所有检查均使用 3 秒超时上下文。

## 五、连接池统计

调用链：[healthDetectEngine](../core/engine.go) → `GET /healthy/stats` → [GetPoolStats](../app/pool_stats.go)

返回所有连接池的实时统计数据：

```json
{
  "code": 20000,
  "msg": "stats",
  "data": {
    "db_max_open_conns": 100,
    "db_open_conns": 10,
    "db_in_use": 3,
    "db_idle": 7,
    "db_wait_count": 0,
    "db_wait_duration_ms": 0,
    "db_max_idle_closed": 0,
    "db_max_lifetime_closed": 0,
    "redis_pool_size": 10,
    "redis_active_conns": 2,
    "redis_idle_conns": 8
  }
}
```

### 5.1 数据库连接池字段

| 字段 | 说明 |
|------|------|
| `db_max_open_conns` | 最大打开连接数 |
| `db_open_conns` | 当前打开连接数 |
| `db_in_use` | 使用中的连接数 |
| `db_idle` | 空闲连接数 |
| `db_wait_count` | 累计等待获取连接的次数 |
| `db_wait_duration_ms` | 累计等待连接的时间（毫秒） |

### 5.2 Redis 连接池字段

| 字段 | 说明 |
|------|------|
| `redis_pool_size` | 连接池大小 |
| `redis_active_conns` | 活跃连接数 |
| `redis_idle_conns` | 空闲连接数 |

## 六、连接池告警

当数据库连接使用率超过 80% 时，[checkDBHealth](../app/pool_stats.go) 会自动输出 Warn 级别日志：

```
[连接池] MySQL 连接使用率过高: 85/100 (85.0%)
```

## 七、Kubernetes 配置示例

```yaml
livenessProbe:
  httpGet:
    path: /healthy
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthy/ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
```

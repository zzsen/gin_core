# 指标监控

`metrics` 模块提供 Prometheus 指标监控功能，支持 HTTP 请求指标、连接池指标和自定义业务指标。

## 目录

- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [内置指标](#内置指标)
- [自定义业务指标](#自定义业务指标)
- [Prometheus 集成](#prometheus-集成)
- [Grafana 可视化](#grafana-可视化)
- [常用 PromQL 查询](#常用-promql-查询)

## 快速开始

### 1. 启用指标监控

在 `config.yaml` 中启用：

```yaml
metrics:
  enabled: true
  path: "/metrics"
  excludePaths:
    - "/healthy"
    - "/metrics"

service:
  middlewares:
    - "prometheusHandler"  # 确保在列表中
    - "exceptionHandler"
    - "traceIdHandler"
    - "traceLogHandler"
    - "timeoutHandler"
```

### 2. 访问指标端点

启动服务后，访问 `http://localhost:8055/metrics` 查看所有指标：

```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/api/users",status="200"} 1234

# HELP http_request_duration_seconds HTTP request duration in seconds
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="GET",path="/api/users",le="0.1"} 1000
...
```

## 配置说明

```yaml
metrics:
  enabled: true           # 是否启用指标监控
  path: "/metrics"        # 指标端点路径
  excludePaths:           # 不统计的路径列表
    - "/healthy"
    - "/healthy/ready"
    - "/healthy/stats"
    - "/metrics"
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | false | 是否启用指标监控 |
| `path` | string | `/metrics` | 指标端点路径 |
| `excludePaths` | []string | - | 不统计的路径列表 |

## 内置指标

### HTTP 请求指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `http_requests_total` | Counter | method, path, status | HTTP 请求总数 |
| `http_request_duration_seconds` | Histogram | method, path | 请求耗时分布（秒） |
| `http_requests_in_flight` | Gauge | - | 当前处理中的请求数 |

### 数据库连接池指标

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `db_pool_open_connections` | Gauge | 数据库打开连接数 |
| `db_pool_idle_connections` | Gauge | 数据库空闲连接数 |
| `db_pool_in_use_connections` | Gauge | 数据库使用中连接数 |
| `db_pool_wait_count_total` | Counter | 等待连接总次数 |

### Redis 连接池指标

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `redis_pool_hits_total` | Counter | 连接池命中次数 |
| `redis_pool_misses_total` | Counter | 连接池未命中次数 |
| `redis_pool_total_connections` | Gauge | 连接池总连接数 |
| `redis_pool_idle_connections` | Gauge | 连接池空闲连接数 |

## 自定义业务指标

### 创建计数器（Counter）

计数器只能增加，适用于统计请求数、订单数等。

```go
import "github.com/zzsen/gin_core/metrics"

// 方式1：简单计数器
var orderCounter = metrics.NewCounter(
    "orders_created_total",
    "Total number of orders created",
)

// 在业务代码中使用
func CreateOrder() {
    // 创建订单逻辑...
    orderCounter.Inc()  // 计数+1
}

// 方式2：带标签的计数器
var orderCounterByType = metrics.NewCounterVec(
    "orders_by_type_total",
    "Total number of orders by type",
    []string{"order_type", "payment_method"},
)

// 在业务代码中使用
func CreateOrder(orderType, paymentMethod string) {
    // 创建订单逻辑...
    orderCounterByType.WithLabelValues(orderType, paymentMethod).Inc()
}
```

### 创建仪表（Gauge）

仪表可增可减，适用于统计当前值，如在线用户数、队列长度等。

```go
import "github.com/zzsen/gin_core/metrics"

var activeUsers = metrics.NewGauge(
    "active_users",
    "Number of currently active users",
)

// 在业务代码中使用
func UserLogin() {
    activeUsers.Inc()  // +1
}

func UserLogout() {
    activeUsers.Dec()  // -1
}

func SetActiveUsers(count int) {
    activeUsers.Set(float64(count))  // 设置具体值
}
```

### 创建直方图（Histogram）

直方图用于统计数据分布，适用于响应时间、请求大小等。

```go
import "github.com/zzsen/gin_core/metrics"

var orderProcessingDuration = metrics.NewHistogram(
    "order_processing_duration_seconds",
    "Order processing duration in seconds",
    []float64{0.1, 0.5, 1, 2, 5, 10},  // 分桶边界
)

// 在业务代码中使用
func ProcessOrder() {
    start := time.Now()
    
    // 处理订单逻辑...
    
    duration := time.Since(start).Seconds()
    orderProcessingDuration.Observe(duration)
}
```

### 完整示例：订单服务指标

```go
package service

import (
    "time"
    "github.com/zzsen/gin_core/metrics"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// 定义订单相关指标
var (
    // 订单创建总数（按类型分组）
    ordersCreated = metrics.NewCounterVec(
        "orders_created_total",
        "Total number of orders created",
        []string{"order_type"},
    )
    
    // 订单处理耗时
    orderProcessingTime = metrics.NewHistogram(
        "order_processing_seconds",
        "Time spent processing orders",
        []float64{0.1, 0.25, 0.5, 1, 2.5, 5},
    )
    
    // 待处理订单数
    pendingOrders = metrics.NewGauge(
        "orders_pending",
        "Number of pending orders",
    )
)

type OrderService struct {}

func (s *OrderService) CreateOrder(orderType string) error {
    start := time.Now()
    
    // 业务逻辑...
    pendingOrders.Inc()
    
    // 记录指标
    ordersCreated.WithLabelValues(orderType).Inc()
    orderProcessingTime.Observe(time.Since(start).Seconds())
    
    return nil
}

func (s *OrderService) CompleteOrder(orderId string) error {
    // 完成订单...
    pendingOrders.Dec()
    return nil
}
```

## Prometheus 集成

### 1. 安装 Prometheus

**Docker 方式：**

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v /path/to/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

**二进制方式：**

下载地址：https://prometheus.io/download/

### 2. 配置 Prometheus

创建 `prometheus.yml`：

```yaml
global:
  scrape_interval: 15s      # 采集间隔
  evaluation_interval: 15s  # 规则评估间隔

scrape_configs:
  # 采集应用指标
  - job_name: 'gin_core_app'
    static_configs:
      - targets: ['localhost:8055']  # 应用地址
    metrics_path: '/metrics'         # 指标端点
    
  # 多实例配置
  - job_name: 'gin_core_cluster'
    static_configs:
      - targets:
        - 'app1.example.com:8055'
        - 'app2.example.com:8055'
        - 'app3.example.com:8055'
```

### 3. 启动 Prometheus

```bash
./prometheus --config.file=prometheus.yml
```

访问 `http://localhost:9090` 打开 Prometheus Web UI。

### 4. 验证数据采集

在 Prometheus Web UI 中：

1. 点击 **Status** → **Targets**
2. 检查应用 target 状态是否为 **UP**
3. 在 **Graph** 页面输入 `http_requests_total` 查询

## Grafana 可视化

### 1. 安装 Grafana

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

访问 `http://localhost:3000`，默认账号密码：`admin/admin`

### 2. 添加 Prometheus 数据源

1. 点击 **Configuration** → **Data Sources**
2. 点击 **Add data source**
3. 选择 **Prometheus**
4. 填写 URL：`http://prometheus:9090`（或实际地址）
5. 点击 **Save & Test**

### 3. 创建仪表盘

#### HTTP 请求 QPS 面板

```
rate(http_requests_total[5m])
```

#### HTTP 请求延迟 P99 面板

```
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

#### 数据库连接池使用率面板

```
db_pool_in_use_connections / db_pool_open_connections * 100
```

#### 错误率面板

```
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100
```

### 4. 导入预置仪表盘

可以从 Grafana 官方仪表盘库导入：https://grafana.com/grafana/dashboards/

推荐仪表盘 ID：
- **6671** - Go 运行时指标
- **10991** - HTTP 请求仪表盘

## 常用 PromQL 查询

### 请求相关

```promql
# 每秒请求数（QPS）
rate(http_requests_total[1m])

# 按路径分组的 QPS
sum(rate(http_requests_total[1m])) by (path)

# 按状态码分组的请求数
sum(rate(http_requests_total[1m])) by (status)

# 错误请求数（5xx）
sum(rate(http_requests_total{status=~"5.."}[1m]))

# 错误率
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100
```

### 延迟相关

```promql
# 平均延迟
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# P50 延迟
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))

# P90 延迟
histogram_quantile(0.90, rate(http_request_duration_seconds_bucket[5m]))

# P99 延迟
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# 按路径分组的 P99 延迟
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (path, le))
```

### 连接池相关

```promql
# 数据库连接使用率
db_pool_in_use_connections / db_pool_open_connections * 100

# Redis 连接池命中率
redis_pool_hits_total / (redis_pool_hits_total + redis_pool_misses_total) * 100

# 数据库等待连接次数增长率
rate(db_pool_wait_count_total[5m])
```

## 新增指标开发指南

### 步骤 1：定义指标

在 `metrics/metrics.go` 或业务包中定义：

```go
// 在 metrics/metrics.go 中添加（框架级指标）
var (
    MyNewMetric = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "my_new_metric_total",
            Help: "Description of my new metric",
        },
    )
)

// 或在业务包中添加（业务级指标）
var myBusinessMetric = metrics.NewCounter(
    "my_business_metric_total",
    "Description of business metric",
)
```

### 步骤 2：在业务代码中使用

```go
func MyBusinessFunction() {
    // 业务逻辑...
    
    // 记录指标
    metrics.MyNewMetric.Inc()
    // 或
    myBusinessMetric.Inc()
}
```

### 步骤 3：验证指标

1. 重启应用
2. 访问 `/metrics` 端点
3. 搜索新增的指标名称

### 步骤 4：在 Prometheus 中查询

```promql
rate(my_new_metric_total[5m])
```

### 步骤 5：添加 Grafana 面板

在 Grafana 仪表盘中添加新面板，使用相应的 PromQL 查询。

## 最佳实践

### 1. 指标命名规范

```
# 格式：{namespace}_{subsystem}_{name}_{unit}

# 好的命名
http_requests_total
http_request_duration_seconds
db_pool_connections_count

# 不好的命名
requests           # 太笼统
httpRequestsTotal  # 不使用驼峰
request_time       # 没有单位
```

### 2. 标签使用建议

```go
// ✅ 好的：有限的标签值
ordersCreated.WithLabelValues("normal", "alipay").Inc()
ordersCreated.WithLabelValues("vip", "wechat").Inc()

// ❌ 不好的：无限的标签值（会导致指标爆炸）
ordersCreated.WithLabelValues(orderId).Inc()  // orderId 是无限的
ordersCreated.WithLabelValues(userId).Inc()   // userId 是无限的
```

### 3. 指标类型选择

| 场景 | 推荐类型 |
|------|----------|
| 统计总数（只增不减） | Counter |
| 统计当前值（可增可减） | Gauge |
| 统计分布（延迟、大小） | Histogram |
| 统计分位数（需要精确值） | Summary |

### 4. 避免高基数

```go
// ❌ 不好的：path 包含动态参数
http_requests_total{path="/users/123/orders/456"}
http_requests_total{path="/users/789/orders/012"}

// ✅ 好的：使用路由模板
http_requests_total{path="/users/:id/orders/:orderId"}
```

## 相关链接

- [Prometheus 官方文档](https://prometheus.io/docs/)
- [Grafana 官方文档](https://grafana.com/docs/)
- [PromQL 教程](https://prometheus.io/docs/prometheus/latest/querying/basics/)
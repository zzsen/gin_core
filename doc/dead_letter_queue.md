# 死信队列（Dead Letter Queue）

## 一、概述

### 1.1 什么是死信队列

**死信队列（Dead Letter Queue，简称 DLQ）** 是消息队列系统中一种特殊的队列，用于存储那些无法被正常消费的消息。当消息在正常队列中无法被成功处理时，会被转发到死信队列中进行特殊处理。

在 RabbitMQ 中，当消息变成"死信"（Dead Letter）后，会被重新发布到另一个交换机（Dead Letter Exchange，DLX），然后路由到死信队列中。

### 1.2 消息变成死信的场景

消息会在以下情况下变成死信：

| 场景 | 说明 |
|------|------|
| **消息被拒绝** | 消费者使用 `basic.reject` 或 `basic.nack` 拒绝消息，且 `requeue=false` |
| **消息过期** | 消息在队列中存活时间超过设置的 TTL（Time To Live） |
| **队列达到最大长度** | 队列中的消息数量或字节数超过设置的最大值 |

### 1.3 死信队列的核心价值

```
┌─────────────────────────────────────────────────────────────────┐
│                        死信队列的核心价值                         │
├─────────────────────────────────────────────────────────────────┤
│  ✓ 消息不丢失    - 无法处理的消息有了"归宿"                        │
│  ✓ 问题可追溯    - 集中存储问题消息，便于分析和排查                  │
│  ✓ 系统更健壮    - 避免问题消息无限重试，影响正常消息处理            │
│  ✓ 支持人工干预  - 可以人工检查、修复后重新投递                      │
│  ✓ 监控告警      - 通过监控死信队列长度，及时发现系统异常            │
└─────────────────────────────────────────────────────────────────┘
```

## 二、工作原理

### 2.1 整体架构

```
                              正常消费流程
                    ┌────────────────────────────┐
                    │                            ▼
┌──────────┐    ┌───┴────────┐    ┌─────────┐    ┌──────────────┐
│ Producer │───▶│  Exchange  │───▶│  Queue  │───▶│   Consumer   │
└──────────┘    └────────────┘    └─────────┘    └──────────────┘
                                       │
                                       │ 消息被拒绝/过期/队列满
                                       ▼
                    ┌────────────┐    ┌─────────────┐
                    │ DL Exchange│───▶│  DL Queue   │
                    └────────────┘    └─────────────┘
                                            │
                                            ▼
                                   ┌──────────────────┐
                                   │  DL Consumer     │
                                   │  - 日志记录       │
                                   │  - 告警通知       │
                                   │  - 人工处理       │
                                   │  - 重新投递       │
                                   └──────────────────┘
```

### 2.2 消息流转过程

1. **生产者**发送消息到**交换机**
2. 交换机根据路由规则将消息投递到**正常队列**
3. **消费者**从队列获取消息并处理
4. 处理成功：消息被确认（ACK），从队列删除
5. 处理失败：
   - 如果重试次数未超限：重新入队（requeue=true）
   - 如果重试次数超限：拒绝消息（requeue=false），消息进入**死信交换机**
6. 死信交换机将消息路由到**死信队列**
7. 死信队列消费者对问题消息进行特殊处理

## 三、使用场景

### 3.1 常见应用场景

| 场景 | 说明 | 处理方式 |
|------|------|----------|
| **订单超时处理** | 订单30分钟未支付自动取消 | 设置消息 TTL，过期后进入死信队列触发取消逻辑 |
| **消息重试机制** | 处理失败的消息进行有限次重试 | 重试N次后进入死信队列，避免无限重试 |
| **异常消息隔离** | 格式错误或无法解析的消息 | 隔离到死信队列，防止影响正常消息 |
| **延迟队列实现** | 延迟执行的任务 | 利用 TTL + 死信队列实现延迟消费 |
| **问题排查分析** | 分析失败原因 | 集中存储便于事后分析 |

### 3.2 延迟队列实现原理

利用死信队列可以实现延迟队列功能：

```
┌──────────┐    ┌───────────────────────┐    ┌─────────────┐
│ Producer │───▶│  延迟队列（设置TTL）   │───▶│  死信交换机  │
└──────────┘    │  无消费者              │    └──────┬──────┘
                └───────────────────────┘           │
                                                    ▼
                                            ┌─────────────┐
                                            │  实际队列   │───▶ Consumer
                                            └─────────────┘

说明：
1. 消息发送到延迟队列，该队列不设置消费者
2. 消息在队列中等待至 TTL 过期
3. 过期后消息被转发到死信交换机
4. 死信交换机路由到实际处理队列
5. 消费者从实际队列消费消息
```

## 四、在 gin_core 中使用死信队列

### 4.1 配置结构

框架提供了以下配置结构来支持死信队列和高级消费功能：

#### DeadLetterConfig - 死信队列配置

```go
// DeadLetterConfig 死信队列配置
type DeadLetterConfig struct {
    Enabled    bool   // 是否启用死信队列
    Exchange   string // 死信交换机名称，为空时自动生成（原交换机名称 + ".dlx"）
    RoutingKey string // 死信路由键，为空时使用原路由键
    QueueName  string // 死信队列名称，为空时自动生成（原队列名称 + ".dlq"）
    MessageTTL int64  // 消息在死信队列中的存活时间（毫秒），0表示永不过期
}
```

#### ConsumeConfig - 消费者配置

```go
// ConsumeConfig 消费者配置
type ConsumeConfig struct {
    PrefetchCount int           // 预取数量，控制消费者一次从队列获取的消息数量
    MaxRetry      int           // 最大重试次数，超过后消息将被发送到死信队列
    RetryDelay    time.Duration // 重试延迟时间
}
```

#### PublishConfirmConfig - 发布确认配置

```go
// PublishConfirmConfig Publisher Confirms 配置
type PublishConfirmConfig struct {
    Enabled bool          // 是否启用发布确认
    Timeout time.Duration // 确认超时时间
}
```

### 4.2 配置参数说明

#### 死信队列配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Enabled` | bool | false | 是否启用死信队列 |
| `Exchange` | string | `{主交换机}.dlx` | 死信交换机名称，为空时自动生成 |
| `RoutingKey` | string | 原路由键 | 死信路由键，为空时使用原路由键 |
| `QueueName` | string | `{主队列名}.dlq` | 死信队列名称，为空时自动生成 |
| `MessageTTL` | int64 | 0（永不过期） | 消息过期时间（毫秒） |

#### 消费者配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `PrefetchCount` | int | 1 | 预取数量 |
| `MaxRetry` | int | 3 | 最大重试次数，超过后进入死信队列 |
| `RetryDelay` | time.Duration | 0 | 重试延迟时间 |

### 4.3 调用链

消费者初始化死信队列的调用链：

[InitialRabbitMqWithContext](../initialize/rabbitmq_consumer.go) → [startMqConsumeWithContext](../initialize/rabbitmq_consumer.go) → [ConsumeWithContext](../model/config/rabbitmq.go) → [initChannel](../model/config/rabbitmq.go) → [initDeadLetterQueue](../model/config/rabbitmq.go)

消息处理与死信转发的调用链：

[ConsumeWithContext](../model/config/rabbitmq.go) → [handleMessage](../model/config/rabbitmq.go) → [getRetryCount](../model/config/rabbitmq.go) → `msg.Nack(false, false)` → 消息进入死信队列

消费者优雅关闭的调用链：

`context.Cancel()` → [ConsumeWithContext](../model/config/rabbitmq.go) 退出循环 → [StopAllConsumers](../initialize/rabbitmq_consumer.go)

### 4.4 完整使用示例

#### 4.4.1 定义消费者配置

```go
package mq

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/zzsen/gin_core/logger"
    "github.com/zzsen/gin_core/model/config"
)

// OrderMessage 订单消息结构
type OrderMessage struct {
    OrderID   string  `json:"orderId"`
    UserID    string  `json:"userId"`
    Amount    float64 `json:"amount"`
    Timestamp int64   `json:"timestamp"`
}

// 订单处理队列配置（带死信队列）
var OrderQueue = config.MessageQueue{
    QueueName:    "order.process",
    ExchangeName: "order.exchange",
    ExchangeType: "direct",
    RoutingKey:   "order.create",
    
    // 消费者配置
    ConsumeConfig: config.ConsumeConfig{
        PrefetchCount: 10,          // 预取10条消息
        MaxRetry:      3,           // 最多重试3次
        RetryDelay:    time.Second, // 重试间隔1秒
    },
    
    // 死信队列配置
    DeadLetter: config.DeadLetterConfig{
        Enabled:    true,
        MessageTTL: 86400000,       // 消息24小时后过期
        // 以下字段可选，为空时自动生成
        // Exchange:   "order.exchange.dlx",
        // QueueName:  "order.process.dlq",
        // RoutingKey: "order.create",
    },
    
    // 消费函数（带 context，支持优雅关闭）
    FunWithCtx: processOrderMessage,
}

// processOrderMessage 处理订单消息
func processOrderMessage(ctx context.Context, msg string) error {
    var order OrderMessage
    if err := json.Unmarshal([]byte(msg), &order); err != nil {
        // 消息格式错误，直接返回错误，消息将进入死信队列
        return fmt.Errorf("消息解析失败: %w", err)
    }
    
    // 检查 context 是否已取消（优雅关闭）
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // 处理订单逻辑
    if err := processOrder(ctx, &order); err != nil {
        return fmt.Errorf("订单处理失败: %w", err)
    }
    
    logger.Info("[订单队列] 订单处理成功, orderID: %s", order.OrderID)
    return nil
}

func processOrder(ctx context.Context, order *OrderMessage) error {
    // 具体的订单处理逻辑
    return nil
}
```

#### 4.4.2 初始化消费者

```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    
    "demo/mq"
    "github.com/zzsen/gin_core/initialize"
    "github.com/zzsen/gin_core/logger"
)

func main() {
    // 创建可取消的 context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // 初始化消费者
    initialize.InitialRabbitMqWithContext(ctx,
        mq.OrderQueue,
        // 可以添加更多队列...
    )
    
    // 优雅关闭
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    logger.Info("正在关闭服务...")
    cancel() // 触发优雅关闭
}
```

#### 4.4.3 发送消息到队列

```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "demo/mq"
    "github.com/zzsen/gin_core/app"
    "github.com/zzsen/gin_core/logger"
)

// CreateOrder 创建订单并发送消息
func CreateOrder(ctx context.Context, orderID, userID string, amount float64) error {
    // 构建消息
    msg := mq.OrderMessage{
        OrderID:   orderID,
        UserID:    userID,
        Amount:    amount,
        Timestamp: time.Now().Unix(),
    }
    
    msgBytes, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    // 方式1：使用全局方法发送（推荐）
    err = app.SendRabbitMqMsg(
        "order.process",   // 队列名
        "order.exchange",  // 交换机名
        "direct",          // 交换机类型
        "order.create",    // 路由键
        string(msgBytes),  // 消息内容
    )
    if err != nil {
        logger.Error("发送订单消息失败: %v", err)
        return err
    }
    
    logger.Info("订单消息发送成功, orderID: %s", orderID)
    return nil
}

// CreateOrderWithConfirm 创建订单并发送消息（启用发布确认）
func CreateOrderWithConfirm(ctx context.Context, orderID, userID string, amount float64) error {
    msg := mq.OrderMessage{
        OrderID:   orderID,
        UserID:    userID,
        Amount:    amount,
        Timestamp: time.Now().Unix(),
    }
    
    msgBytes, _ := json.Marshal(msg)
    
    // 使用 Publisher Confirms 确保消息投递成功
    err := app.SendRabbitMqMsgWithConfirm(
        "order.process",
        "order.exchange",
        "direct",
        "order.create",
        string(msgBytes),
        5*time.Second, // 确认超时时间
    )
    if err != nil {
        return fmt.Errorf("消息发送失败（确认模式）: %w", err)
    }
    
    return nil
}

// BatchCreateOrders 批量创建订单
func BatchCreateOrders(ctx context.Context, orders []mq.OrderMessage) error {
    messages := make([]string, len(orders))
    for i, order := range orders {
        msgBytes, _ := json.Marshal(order)
        messages[i] = string(msgBytes)
    }
    
    // 批量发送消息
    err := app.SendRabbitMqMsgBatch(
        "order.process",
        "order.exchange",
        "direct",
        "order.create",
        messages,
    )
    if err != nil {
        return fmt.Errorf("批量消息发送失败: %w", err)
    }
    
    logger.Info("批量订单消息发送成功, 数量: %d", len(orders))
    return nil
}
```

#### 4.4.4 死信队列消费者（可选）

如果需要对死信队列中的消息进行特殊处理，可以单独定义死信队列消费者：

```go
package mq

import (
    "context"
    
    "github.com/zzsen/gin_core/logger"
    "github.com/zzsen/gin_core/model/config"
)

// 死信队列消费者配置
var OrderDeadLetterQueue = config.MessageQueue{
    QueueName:    "order.process.dlq",
    ExchangeName: "order.exchange.dlx",
    ExchangeType: "direct",
    RoutingKey:   "order.create",
    
    ConsumeConfig: config.ConsumeConfig{
        PrefetchCount: 1, // 死信队列建议设置较小的预取数量
    },
    
    FunWithCtx: handleDeadLetter,
}

// handleDeadLetter 处理死信消息
func handleDeadLetter(ctx context.Context, msg string) error {
    // 1. 记录日志
    logger.Warn("[死信队列] 收到死信消息: %s", msg)
    
    // 2. 发送告警通知（邮件、钉钉、企业微信等）
    sendAlert("订单处理失败", msg)
    
    // 3. 存储到数据库便于后续人工处理
    saveToDeadLetterTable(msg)
    
    // 4. 可选：尝试重新投递到原队列
    // redeliverToOriginalQueue(msg)
    
    return nil
}

func sendAlert(title, content string) {
    // 发送告警通知逻辑
}

func saveToDeadLetterTable(msg string) {
    // 保存到数据库逻辑
}
```

## 五、新增功能说明

### 5.1 Publisher Confirms（发布确认）

Publisher Confirms 机制确保消息成功投递到 RabbitMQ 服务器：

```go
// 配置启用 Publisher Confirms
producer := config.MessageQueue{
    QueueName:    "order.process",
    ExchangeName: "order.exchange",
    ExchangeType: "direct",
    RoutingKey:   "order.create",
    
    PublishConfirm: config.PublishConfirmConfig{
        Enabled: true,
        Timeout: 5 * time.Second,
    },
}

// 发送消息时会等待服务器确认
err := producer.PublishWithContext(ctx, message)
if err != nil {
    // 消息发送失败或确认超时
    log.Error("消息发送失败: %v", err)
}
```

### 5.2 批量发布消息

批量发布可以提高消息发送效率：

```go
messages := []string{
    `{"orderId": "001", "amount": 100}`,
    `{"orderId": "002", "amount": 200}`,
    `{"orderId": "003", "amount": 300}`,
}

// 批量发送
err := app.SendRabbitMqMsgBatch(
    "order.process",
    "order.exchange", 
    "direct",
    "order.create",
    messages,
)
```

### 5.3 消费者优雅关闭

通过 context 实现消费者的优雅关闭：

```go
// 创建可取消的 context
ctx, cancel := context.WithCancel(context.Background())

// 初始化消费者
initialize.InitialRabbitMqWithContext(ctx, orderQueue)

// 接收关闭信号
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// 优雅关闭：取消 context，消费者会完成当前消息处理后退出
cancel()

// 或者停止所有消费者
initialize.StopAllConsumers()

// 或者停止指定消费者
initialize.StopConsumer(orderQueue.GetInfo())
```

## 六、消息处理流程详解

### 6.1 消息处理状态机

```
                    ┌─────────────┐
                    │   新消息    │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
           ┌───────│   处理中    │───────┐
           │       └─────────────┘       │
           │                             │
        成功│                             │失败
           │                             │
           ▼                             ▼
    ┌─────────────┐              ┌─────────────┐
    │   ACK确认   │              │  检查重试   │
    │  消息删除   │              │    次数     │
    └─────────────┘              └──────┬──────┘
                                        │
                           ┌────────────┴────────────┐
                           │                         │
                      未超限│                         │已超限
                           │                         │
                           ▼                         ▼
                    ┌─────────────┐          ┌─────────────┐
                    │  重新入队   │          │   NACK拒绝  │
                    │ requeue=true│          │requeue=false│
                    └──────┬──────┘          └──────┬──────┘
                           │                        │
                           │                        ▼
                           │                 ┌─────────────┐
                           │                 │  死信队列   │
                           │                 └─────────────┘
                           │
                           └──────▶ 返回"处理中"状态
```

### 6.2 重试次数获取

框架通过消息头中的 `x-death` 字段获取重试次数：

```go
// getRetryCount 获取消息重试次数
func (m *MessageQueue) getRetryCount(msg amqp.Delivery) int {
    if msg.Headers == nil {
        return 0
    }
    if deaths, ok := msg.Headers["x-death"].([]interface{}); ok && len(deaths) > 0 {
        if death, ok := deaths[0].(amqp.Table); ok {
            if count, ok := death["count"].(int64); ok {
                return int(count)
            }
        }
    }
    return 0
}
```

## 七、最佳实践

### 7.1 设计建议

| 建议 | 说明 |
|------|------|
| **合理设置重试次数** | 一般设置 3-5 次，避免无限重试消耗资源 |
| **设置消息 TTL** | 防止死信队列无限膨胀，建议设置 24-72 小时 |
| **监控死信队列** | 监控队列长度，超过阈值及时告警 |
| **记录完整上下文** | 死信消息应包含完整的业务上下文信息 |
| **分类处理死信** | 根据错误类型分类处理，如格式错误、业务异常、系统异常 |
| **定期清理** | 定期清理已处理或过期的死信消息 |

### 7.2 命名规范

建议采用统一的命名规范：

| 类型 | 命名规范 | 示例 |
|------|----------|------|
| 死信交换机 | `{原交换机}.dlx` | `order.exchange.dlx` |
| 死信队列 | `{原队列}.dlq` | `order.process.dlq` |
| 死信路由键 | `{原路由键}.dead` | `order.create.dead` |

### 7.3 错误分类处理

```go
func handleDeadLetter(ctx context.Context, msg string) error {
    // 解析消息，获取错误信息
    var deadMsg DeadMessage
    json.Unmarshal([]byte(msg), &deadMsg)
    
    switch classifyError(deadMsg.Error) {
    case ErrorTypeFormat:
        // 格式错误：记录日志，不重试
        logger.Error("消息格式错误，已丢弃: %s", msg)
        
    case ErrorTypeBusiness:
        // 业务错误：通知相关人员处理
        notifyBusinessTeam(deadMsg)
        
    case ErrorTypeSystem:
        // 系统错误：可能是临时故障，稍后重试
        scheduleRetry(deadMsg, 30*time.Minute)
        
    default:
        // 未知错误：保存供后续分析
        saveForAnalysis(deadMsg)
    }
    
    return nil
}
```

## 八、注意事项

### 8.1 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 消息未进入死信队列 | 死信队列配置错误 | 检查 `Enabled` 是否为 true，交换机和队列是否正确声明 |
| 重试次数不准确 | `x-death` 头未正确传递 | 确保使用框架提供的消息处理方法 |
| 死信队列堆积 | 未设置 TTL 或未消费 | 设置合理的 MessageTTL，添加死信队列消费者 |
| 队列参数冲突 | 修改了已存在队列的配置 | 删除原队列重新创建，或使用新的队列名 |

### 8.2 性能考虑

- 死信队列消费者应设置较小的 `PrefetchCount`（建议为 1），避免批量处理失败消息
- 死信消息处理应加入限流机制，防止系统压力过大
- 对于高频失败的消息，考虑增加处理延迟，避免立即重试

### 8.3 监控指标

建议监控以下指标：

```
死信队列相关监控指标
├── 死信队列消息数量
│   └── 告警阈值：> 100 条
├── 死信消息产生速率
│   └── 告警阈值：> 10 条/分钟
├── 死信消息处理延迟
│   └── 告警阈值：> 5 分钟
└── 按错误类型分类统计
    └── 分析各类错误占比
```

## 九、总结

死信队列是消息队列系统中保障消息可靠性的重要机制。通过合理配置和使用死信队列，可以：

1. **避免消息丢失** - 无法处理的消息有了安全的存储位置
2. **提高系统健壮性** - 问题消息不会阻塞正常消息处理
3. **便于问题排查** - 集中存储便于分析和追溯
4. **支持灵活处理** - 可以人工干预或自动重试

在 gin_core 框架中，只需简单配置 `DeadLetterConfig` 即可启用死信队列功能，框架会自动处理交换机和队列的声明、消息的重试逻辑以及死信转发等操作。

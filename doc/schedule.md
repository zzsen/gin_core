# 定时任务（Schedule）

## 一、概述

在 `gin_core` 框架中，定时任务是系统自动化运行的重要组成部分，可用于周期性执行特定任务，如数据备份、缓存清理、定时通知等。框架支持灵活配置定时任务，可根据不同的部署环境选择是否启动定时任务，同时提供了方便的方式来编写和管理定时任务。

## 二、定时任务配置
考虑到同一份代码在不同部署环境下的需求差异，框架支持在配置文件中设置是否启动定时任务。在配置文件里，可通过 `system` 下的 `useSchedule` 字段来控制：
```yaml
system: # 系统配置
  useSchedule: true # 是否启动定时任务
```

## 三、定时任务编写
建议将所有的定时任务，都存放于`schedule`目录下，每个文件是一个单独的定时任务，最后统一通过import时调用init的方式，将定时任务添加到定时任务列表，并启动定时任务。
### 3.1 定义定时任务方法
    ```golang
    // schedule/print.go
    package schedule

    func Print() {
        logger.Info("schedule run")
    }
    ```

### 3.2 将定时任务添加到框架的定时任务列表
    ```golang
    // schedule/schedule.go
    package schedule

    import (
        "github.com/zzsen/gin_core/model/config"
    )
    func init() {
        core.AddSchedule(config.ScheduleInfo{
            Cron: "@every 10s",
            Cmd:  Print,
        })
    }
    ```
    > cron表达式可参见: [cron](https://github.com/robfig/cron)

### 3.3 引包并调用init方法
    ```golang
    // main.go
    package main

    import (
        // import时，会调用init方法
        _ "demo/schedule"
        "github.com/zzsen/gin_core/core"
    )

    func main() {
        //启动服务
        core.Start(clearCache)
    }
    ```

### 四、注意事项
* **cron 表达式**：在配置 Cron 字段时，要确保 cron 表达式的正确性，否则可能导致任务无法按预期执行。
* **任务异常处理**：在编写定时任务方法时，建议添加适当的异常处理机制，避免因单个任务异常导致整个系统崩溃。
* **资源占用**：定时任务的执行可能会占用一定的系统资源，要合理安排任务的执行周期和频率，避免对系统性能造成影响。
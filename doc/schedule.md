# 定时任务（Schedule）
虽然我们通过框架开发的 HTTP Server 是请求响应模型的，但是仍然还会有许多场景需要执行一些定时任务，例如：
* 定时上报应用状态。
* 定时从远程接口更新本地缓存。
* 定时进行文件切割、临时文件删除。
框架提供了一套机制来让定时任务的编写和维护更加优雅。

## 定时任务配置
考虑到同一份代码，部署环境不同，可能存在有的服务需要启动定时任务，有的服务不需要启动定时任务，故在配置中，支持配置是否启动定时任务。
```yaml
system: # 系统配置
  useSchedule: true # 是否启动定时任务
```

## 定时任务编写
建议将所有的定时任务，都存放于`schedule`目录下，每个文件是一个单独的定时任务，最后统一通过import时调用init的方式，将定时任务添加到定时任务列表，并启动定时任务。
1. 定义定时任务方法
    ```golang
    // schedule/print.go
    package schedule

    func Print() {
        logger.Info("schedule run")
    }
    ```

2. 将定时任务添加到框架的定时任务列表
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

3. 引包并调用init方法
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

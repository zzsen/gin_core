# gin_core

gin 框架, 用于快速搭建项目

# 安装

## 新项目使用

1. 新建工程目录

   `mkdir [projectName] && cd [projectName]`

   > `projectName`替换为项目工程的名称

2. 初始化 go.mod

   `go mod init [projectName]`

   > `projectName`替换为项目工程的名称
 
3. 拉取`gin_core`依赖包

   `go get -u github.com/zzsen/gin_core`

## 旧项目使用

1. 拉取`gin_core`依赖包

   `go get -u github.com/zzsen/gin_core`

# 文档

- [目录结构](./doc/structure.md)
- [运行参数](./doc/args.md)
- [运行环境](./doc/env.md)
- [配置](./doc/config.md)
- [中间件](./doc/middleware.md)
- [路由](./doc/router.md)
- [控制器](./doc/controller.md)
- [服务](./doc/service.md)
- [服务注册](./doc/service_register.md)
- [定时任务](./doc/schedule.md)
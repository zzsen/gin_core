# 路由 (Router)

## 一、概述

`Router`主要用于描述请求url和具体执行动作的controller的映射关系，框架建议统一将根路由文件`router.go`置于`router`文件夹中，子路由根据业务在`router`文件夹内新建文件夹和文件存放，并统一由根路由文件`router/router.go`统一添加到服务路由中。

## 二、框架路由原理
框架内部路由组织原理，是用到了`*gin.Engine`的`With`方法, 往其中传递`...gin.OptionFunc`类型的方法。
```golang
// With returns a new Engine instance with the provided options.
func (engine *Engine) With(opts ...OptionFunc) *Engine {
	for _, opt := range opts {
		opt(engine)
	}

	return engine
}
```

在gin中，`OptionFunc`是一个接收`*Engine`类型的**方法**
```golang
// gin.go
type OptionFunc func(*Engine)
```

框架暴露了`AddOptionFunc`方法, 运行使用框架的项目将组装好的路由组传递给`core`包中的`optionFuncList`对象, 再通过调用`*gin.Engine`的`With`方法, 将整个`optionFuncList`添加到路由中去。
```golang
// ...
var optionFuncList = make([]gin.OptionFunc, 0)

func AddOptionFunc(optionFunc ...gin.OptionFunc) {
	optionFuncList = append(optionFuncList, optionFunc...)
}

// ...

engine.With(optionFuncList...)

// ...
```

具体步骤如下：
1. 定义返回 `OptionFunc` 类型的路由配置方法。
2. 使用 `AddOptionFunc` 方法将路由配置方法添加到框架中。
3. 启动服务，框架会自动应用所有添加的路由配置方法。

## 三、统一路由前缀
在 `initEngine` 方法中，如果配置文件里设置了 `service.routePrefix`，那么所有的路由都会加上这个前缀。
```yml
service:
  routePrefix: "routePrefix" # 统一路由前缀
```


## 四、路由添加方式
### 1. 简单添加路由
先以框架内自带的健康检查路由的添加方式为例，展示一个简单的路由添加方式
```golang
// 定义多个OptionFunc路由配置方法
func getCustomRouter1() gin.OptionFunc {
	return func(e *gin.Engine) {
		r := e.Group("customRouter1")
		r.GET("test", func(c *gin.Context) {
			response.Ok(c)
		})
	}
}

func getCustomRouter2() gin.OptionFunc {
	return func(e *gin.Engine) {
		r := e.Group("customRouter2")
		r.GET("test", func(c *gin.Context) {
			response.Ok(c)
		})
	}
}

// 方式1.多次调用core.AddOptionFunc添加单个路由组
core.AddOptionFunc(getCustomRouter1())
core.AddOptionFunc(getCustomRouter2())


// 方式2.单次调用core.AddOptionFunc添加多个路由组
opts := []gin.OptionFunc{}
opts = append(opts, getCustomRouter1())
opts = append(opts, getCustomRouter2())
core.AddOptionFunc(opts...)
```
上述是个简单的添加路由的例子，若业务较简单，只有几个接口，按上述方式添加即可。

### 2. 模块划分
但是，随着业务复杂度的增加，`controller`方法和`middleware`方法的数量也在增加，仍使用上述添加方式的话，会显得臃肿，且不利于维护。此时，建议采用模块划分的添加方式，即：router内容统一存放于`router`目录下，controller的内容统一存放于`controller`目录下，其他如middleware、service和schedule等也如此。

#### 2.1 controller定义

`controller`中，实现controller方法，若业务较复杂，可以在`controller`内根据业务拆分成不同的子controller文件夹
```golang
// controller/healthy/healthy.go
package healthy

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)
func Healthy (ctx *gin.Context) {
	response.OkWithDetail(c, "healthy", gin.H{
		"status": "healthy",
	})
}
```

#### 2.2 路由定义
在`router`中，实现定义返回 `OptionFunc` 类型的路由配置方法，若业务较复杂，可以在`router`内根据业务拆分成不同的子router文件夹
```golang
// router/healthy/healthy.go
package healthy

import (
	"github.com/gin-gonic/gin"

	v1 "demo/controller/healthy"
)
func InitRouter(handlers ...gin.HandlerFunc) func(e *gin.Engine) {
	return func(engine *gin.Engine) {
		router := engine.Group("health", handlers...)
		router.GET("healthy", v1.Healthy)
	}
}
```

#### 2.3 路由添加
在`router/router.go`中，统一将路由添加至`core`的`OptionFunc`列表中
```golang
// router/router.go
package router

import (
	
	"demo/router/healthy"
)

func init() {
	core.AddOptionFuncs(healthy.InitRouter())
	// core.AddOptionFuncs(healthy.InitRouter(demoMiddleware)) // 若需要添加中间件
}
```

#### 2.4 路由注册
根目录下, 新建`main.go`主入口文件, 引入router包时, 通过调用router的init方法, 将路由添加到core的路由链上。
```golang
// main.go
package main

import (
	// import时，会调用init方法
	_ "demo/router"
	"github.com/zzsen/gin_core/core"
)

func main() {
	//启动服务
	core.Start(clearCache)
}
```

## 五、内置路由
框架中, 内置了healthy路由, 用于服务健康检测。若配置了`统一路由前缀`, 假设当前配置为`routerPrefix`, 则健康监测路由为`routerPrefix/healthy`。

## 六、注意事项
* **路由文件组织**：按照框架建议的目录结构组织路由文件，便于维护和管理。
* **中间件使用**：在路由定义时，可以根据需要添加中间件，增强路由的功能。
* **路由前缀配置**：配置统一路由前缀时，确保其符合业务需求，避免出现路由冲突。
# 路由 (Router)

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
1. 定义返回 func(e *gin.Engine) 类型的路由配置函数。
2. 使用 AddOptionFunc 方法将路由配置函数添加到框架中。
3. 启动服务，框架会自动应用所有添加的路由配置函数。

## 统一路由前缀
在 initEngine 方法中，如果配置文件里设置了 service.routePrefix，那么所有的路由都会加上这个前缀。
```yml
service:
  routePrefix: "routePrefix" # 统一路由前缀
```

## 路由添加方式
### 添加单个路由组
`AddOptionFunc`方法支持传递单个
```golang
// 健康检查
var healthDetactEngine = func(e *gin.Engine) {
	r := e.Group("healthy")
	r.GET("", func(c *gin.Context) {
		response.OkWithDetail(c, "healthy", gin.H{
			"status": "healthy",
		})
	})
}

// 添加路由组到 core 的 optionFuncList 对象
core.AddOptionFunc(healthDetactEngine())
```

### 添加多个路由组
```golang

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

## 内置路由
框架中, 内置了healthy路由, 用于服务健康检测。若配置了`统一路由前缀`, 假设当前配置为`routerPrefix`, 则健康监测路由为`routerPrefix/healthy`。
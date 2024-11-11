package config

type ServiceInfo struct {
	Ip            string   `yaml:"ip"`            // 服务ip
	Port          int      `yaml:"port"`          // 服务端口
	RoutePrefix   string   `yaml:"routePrefix"`   // 路由前缀
	SessionExpire int      `yaml:"sessionExpire"` // 缓存的有效时长
	SessionPrefix string   `yaml:"sessionPrefix"` // redis中缓存前缀
	CookieKey     string   `yaml:"cookieKey"`     // session对应的cookie，目前用cookieKey
	Middlewares   []string `yaml:"middlewares"`   // 中间件，顺序对应中间件调用顺序
	ReadTimeout   int      `yaml:"readTimeout"`   // 读取超时时间
	WriteTimeout  int      `yaml:"writeTimeout"`  // 写入超时时间
}

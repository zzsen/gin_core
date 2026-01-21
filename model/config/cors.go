package config

// CORSConfig 跨域资源共享配置
// 用于配置 CORS 中间件的行为
type CORSConfig struct {
	// Enabled 是否启用 CORS 中间件
	Enabled bool `yaml:"enabled"`

	// AllowOrigins 允许的来源列表
	// 支持精确匹配和通配符匹配：
	// - "*" 表示允许所有来源（不建议在生产环境使用）
	// - "http://localhost:3000" 精确匹配
	// - "*.example.com" 通配符匹配（匹配所有 example.com 的子域名）
	AllowOrigins []string `yaml:"allowOrigins"`

	// AllowMethods 允许的 HTTP 方法列表
	// 默认值：GET, POST, PUT, PATCH, DELETE, OPTIONS
	AllowMethods []string `yaml:"allowMethods"`

	// AllowHeaders 允许的请求头列表
	// 默认值：Content-Type, Authorization, X-Trace-Id, X-Request-Id
	AllowHeaders []string `yaml:"allowHeaders"`

	// ExposeHeaders 暴露给浏览器的响应头列表
	// 允许浏览器 JavaScript 访问这些响应头
	ExposeHeaders []string `yaml:"exposeHeaders"`

	// AllowCredentials 是否允许携带凭证（Cookie、HTTP 认证等）
	// 注意：当设置为 true 时，AllowOrigins 不能为 "*"
	AllowCredentials bool `yaml:"allowCredentials"`

	// MaxAge 预检请求结果的缓存时间（秒）
	// 默认值：86400（24小时）
	MaxAge int `yaml:"maxAge"`
}

// GetAllowOrigins 获取允许的来源列表
func (c *CORSConfig) GetAllowOrigins() []string {
	if len(c.AllowOrigins) == 0 {
		return []string{"*"}
	}
	return c.AllowOrigins
}

// GetAllowMethods 获取允许的方法列表
func (c *CORSConfig) GetAllowMethods() []string {
	if len(c.AllowMethods) == 0 {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	return c.AllowMethods
}

// GetAllowHeaders 获取允许的请求头列表
func (c *CORSConfig) GetAllowHeaders() []string {
	if len(c.AllowHeaders) == 0 {
		return []string{"Content-Type", "Authorization", "X-Trace-Id", "X-Request-Id"}
	}
	return c.AllowHeaders
}

// GetMaxAge 获取预检请求缓存时间
func (c *CORSConfig) GetMaxAge() int {
	if c.MaxAge <= 0 {
		return 86400
	}
	return c.MaxAge
}

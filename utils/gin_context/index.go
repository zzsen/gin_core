package ginContext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

// Get 从Gin上下文中获取指定键的值
// 按照优先级顺序依次从以下位置获取：
// 1. URL查询参数 (Query)
// 2. POST表单数据 (PostForm)
// 3. JSON请求体 (RawData)
// 4. URL路径参数 (Param)
//
// 参数:
//   - ctx: Gin上下文对象
//   - key: 要获取的键名
//
// 返回值:
//   - value: 键对应的值，如果未找到则返回空字符串
func Get(ctx *gin.Context, key string) (value string) {
	// 首先尝试从URL查询参数中获取
	value = ctx.Query(key)
	if value != "" {
		return value
	}

	// 如果查询参数为空，尝试从POST表单数据中获取
	value = ctx.PostForm(key)
	if value != "" {
		return value
	}

	// 如果表单数据也为空，尝试从JSON请求体中获取
	b, _ := ctx.GetRawData()
	// 关键点：将读取的数据重新写入到请求体中
	// 因为GetRawData()会消耗请求体，需要重新设置以便后续中间件使用
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(b))

	// 解析JSON数据
	m := map[string]any{}
	json.Unmarshal(b, &m)

	// 检查JSON中是否包含指定的键
	if str, declared := m[key]; declared {
		value = fmt.Sprint(str)
		return value
	}

	// 最后尝试从URL路径参数中获取
	value = ctx.Param(key)
	return value
}

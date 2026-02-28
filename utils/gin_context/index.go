package ginContext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

const parsedBodyKey = "_ginCtx_parsedBody"

// Get 从Gin上下文中获取指定键的值
// 按照优先级顺序依次从以下位置获取：
// 1. URL查询参数 (Query)
// 2. POST表单数据 (PostForm)
// 3. JSON请求体 (RawData)，解析结果会缓存到 context 中避免重复解析
// 4. URL路径参数 (Param)
//
// 参数:
//   - ctx: Gin上下文对象
//   - key: 要获取的键名
//
// 返回值:
//   - value: 键对应的值，如果未找到则返回空字符串
func Get(ctx *gin.Context, key string) (value string) {
	value = ctx.Query(key)
	if value != "" {
		return value
	}

	value = ctx.PostForm(key)
	if value != "" {
		return value
	}

	m := getParsedBody(ctx)

	if str, declared := m[key]; declared {
		value = fmt.Sprint(str)
		return value
	}

	value = ctx.Param(key)
	return value
}

// getParsedBody 获取并缓存 JSON 请求体的解析结果
// 首次调用时读取 RawData 并解析 JSON，结果缓存到 gin.Context 中；
// 后续调用直接返回缓存，避免重复 IO 和反序列化开销。
func getParsedBody(ctx *gin.Context) map[string]any {
	if cached, exists := ctx.Get(parsedBodyKey); exists {
		if m, ok := cached.(map[string]any); ok {
			return m
		}
	}

	b, _ := ctx.GetRawData()
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(b))

	m := map[string]any{}
	json.Unmarshal(b, &m)

	ctx.Set(parsedBodyKey, m)
	return m
}

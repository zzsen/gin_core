package ginContext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

func Get(ctx *gin.Context, key string) (value string) {
	value = ctx.Query(key)
	if value == "" {
		value = ctx.PostForm(key)
	}
	if value == "" {
		b, _ := ctx.GetRawData()
		// 关键点：将读取的数据重新写入到请求体中
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(b))
		m := map[string]interface{}{}
		json.Unmarshal(b, &m)
		if str, declared := m[key]; declared {
			value = fmt.Sprint(str)
		}
	}
	if value == "" {
		value = ctx.Param(key)
	}
	return value
}

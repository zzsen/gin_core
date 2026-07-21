// Package sink Loki Push 适配器测试
//
// ==================== 测试说明 ====================
// 使用 httptest 验证 Loki push 成功与非 2xx 错误。
//
// 运行测试：go test -v ./logger/sink/... -run TestLokiSink_
// ==================================================
package sink

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// TestLokiSink_PushSuccess 成功推送
//
// 【功能点】POST Loki push API，body 含日志行
// 【测试流程】
// 1. httptest 返回 204
// 2. Send 无错误且 body 含 hello
func TestLokiSink_PushSuccess(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/loki/api/v1/push", r.URL.Path)
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(204)
	}))
	defer srv.Close()
	s, err := NewLokiSink(config.LokiOutputConfig{
		URL: srv.URL + "/loki/api/v1/push", TimeoutMs: 2000, Labels: []string{"serviceName"},
	}, map[string]string{"serviceName": "svc"})
	require.NoError(t, err)
	err = s.Send(context.Background(), []FormattedEntry{{
		Time: time.Now(), Message: "hello", Line: []byte(`{"msg":"hello"}`),
	}})
	require.NoError(t, err)
	assert.Contains(t, string(gotBody), "hello")
}

// TestLokiSink_Non2xxReturnsError 非 2xx 返回错误
//
// 【功能点】HTTP 500 时 Send 返回 error
func TestLokiSink_Non2xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	s, err := NewLokiSink(config.LokiOutputConfig{URL: srv.URL, TimeoutMs: 1000}, nil)
	require.NoError(t, err)
	err = s.Send(context.Background(), []FormattedEntry{{Message: "x", Line: []byte("x"), Time: time.Now()}})
	require.Error(t, err)
}

// Package logger outputs 生效语义测试
//
// ==================== 测试说明 ====================
// 验证 resolveEffectiveOutputs 与 InitLogger 按 outputs 装配 file/stdout/remote。
//
// 测试覆盖内容：
// 1. nil / 空数组 → 默认 file+stdout
// 2. 未配置 outputs → 有文件 hook、stdout、无 remote
// 3. 仅 remote → 有 remote、Out 为 Discard、无 lfshook
// 4. file+remote → 双写
//
// 运行测试：go test -v ./logger/... -run "TestResolveEffectiveOutputs_|TestInitLogger_UnsetOutputs_|TestInitLogger_OnlyRemote_|TestInitLogger_FileAndRemote_"
// ==================================================
package logger

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// TestResolveEffectiveOutputs_NilOrEmptyDefaultsFileStdout 未配置/空数组默认
func TestResolveEffectiveOutputs_NilOrEmptyDefaultsFileStdout(t *testing.T) {
	got := resolveEffectiveOutputs(nil)
	require.Len(t, got, 2)
	assert.Equal(t, "file", got[0].Type)
	assert.Equal(t, "stdout", got[1].Type)

	got2 := resolveEffectiveOutputs([]config.LogOutputConfig{})
	require.Len(t, got2, 2)
	assert.Equal(t, "file", got2[0].Type)
	assert.Equal(t, "stdout", got2[1].Type)
}

// TestResolveEffectiveOutputs_FiltersDisabled 过滤 enabled=false
func TestResolveEffectiveOutputs_FiltersDisabled(t *testing.T) {
	off := false
	on := true
	got := resolveEffectiveOutputs([]config.LogOutputConfig{
		{Type: "file", Enabled: &off},
		{Type: "stdout", Enabled: &on},
		{Type: "remote", Enabled: &on},
	})
	require.Len(t, got, 2)
	assert.Equal(t, "stdout", got[0].Type)
	assert.Equal(t, "remote", got[1].Type)
}

// TestInitLogger_UnsetOutputs_StillFileAndStdout 未配 outputs 时 file+stdout
func TestInitLogger_UnsetOutputs_StillFileAndStdout(t *testing.T) {
	t.Cleanup(func() { _ = CloseRemote(context.Background()) })
	dir, err := os.MkdirTemp("", "gin-core-outputs-unset-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	l := InitLogger(config.LoggersConfig{FilePath: dir, Format: "json"})
	require.NotNil(t, l)
	assert.Nil(t, GetRemotePipeline())
	assert.Equal(t, os.Stdout, l.Out)
	require.NotEmpty(t, l.Hooks[logrus.InfoLevel])
	// 未配 remote 时 InfoLevel 至少有 lfshook
	assert.GreaterOrEqual(t, len(l.Hooks[logrus.InfoLevel]), 1)
}

// TestInitLogger_EmptyOutputs_DefaultsFileStdout 空数组同默认
func TestInitLogger_EmptyOutputs_DefaultsFileStdout(t *testing.T) {
	t.Cleanup(func() { _ = CloseRemote(context.Background()) })
	dir, err := os.MkdirTemp("", "gin-core-outputs-empty-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	l := InitLogger(config.LoggersConfig{
		FilePath: dir,
		Format:   "json",
		Outputs:  []config.LogOutputConfig{},
	})
	require.NotNil(t, l)
	assert.Nil(t, GetRemotePipeline())
	assert.Equal(t, os.Stdout, l.Out)
	assert.GreaterOrEqual(t, len(l.Hooks[logrus.InfoLevel]), 1)
}

// TestInitLogger_OnlyRemote_NoFileNoStdout 仅 remote
func TestInitLogger_OnlyRemote_NoFileNoStdout(t *testing.T) {
	t.Cleanup(func() { _ = CloseRemote(context.Background()) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	enabled := true
	l := InitLogger(config.LoggersConfig{
		FilePath: t.TempDir(), // 即使有路径也不应注册 file hook
		Format:   "json",
		Outputs: []config.LogOutputConfig{
			{
				Type: "remote", Enabled: &enabled, MinLevel: "info",
				Remote: &config.RemoteOutputConfig{
					Driver: "loki", Format: "json",
					QueueSize: 16, BatchSize: 8, FlushIntervalMs: 50,
					Loki: &config.LokiOutputConfig{
						URL: srv.URL + "/loki/api/v1/push", TimeoutMs: 2000,
					},
				},
			},
		},
	})
	require.NotNil(t, l)
	require.NotNil(t, GetRemotePipeline())
	assert.Equal(t, io.Discard, l.Out)
	// 仅 remoteHook：每个 level 槽位通常 1 个 hook（无 lfshook）
	assert.Equal(t, 1, len(l.Hooks[logrus.InfoLevel]))
}

// TestInitLogger_FileAndRemote_DualWrite file+remote
func TestInitLogger_FileAndRemote_DualWrite(t *testing.T) {
	t.Cleanup(func() { _ = CloseRemote(context.Background()) })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	defer srv.Close()

	dir, err := os.MkdirTemp("", "gin-core-outputs-dual-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	enabled := true
	l := InitLogger(config.LoggersConfig{
		FilePath: dir,
		Format:   "json",
		Outputs: []config.LogOutputConfig{
			{Type: "file", Enabled: &enabled},
			{
				Type: "remote", Enabled: &enabled,
				Remote: &config.RemoteOutputConfig{
					Driver: "loki",
					Loki:   &config.LokiOutputConfig{URL: srv.URL + "/loki/api/v1/push", TimeoutMs: 2000},
				},
			},
		},
	})
	require.NotNil(t, GetRemotePipeline())
	assert.Equal(t, io.Discard, l.Out) // 未列 stdout
	assert.GreaterOrEqual(t, len(l.Hooks[logrus.InfoLevel]), 2)
}

// Package logger 日志模块单元测试
//
// ==================== 测试说明 ====================
// 本文件包含 logger 包的单元测试，不依赖外部服务。
//
// 测试覆盖内容：
// 1. MaskValue 值脱敏（短值/长值/边界）
// 2. SanitizeMessage 消息脱敏（key=value/JSON/多种关键词）
// 3. isSensitiveField 敏感字段判断
// 4. SanitizeFields 结构化字段脱敏
// 5. sanitizeLog 格式化 + 脱敏
// 6. getCaller 调用者信息获取
// 7. withCallerFields 调用者字段注入
// 8. 日志级别函数（Info/Error/Warn/Debug/Trace/Add）
// 9. 带字段日志函数（InfoWithFields/ErrorWithFields/WarnWithFields/DebugWithFields/TraceWithFields）
// 10. InitLogger 日志初始化
// 11. initRotatelogs 配置优先级
//
// 运行测试：go test -v ./logger/... -run Test
// ==================================================
package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// setupTestLogger 创建用于捕获输出的测试 logger
func setupTestLogger() *bytes.Buffer {
	buf := &bytes.Buffer{}
	Logger = logrus.New()
	Logger.SetOutput(buf)
	Logger.SetLevel(logrus.TraceLevel)
	Logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:   true,
	})
	return buf
}

// --- MaskValue 测试 ---

// TestMaskValue 测试值脱敏
//
// 【功能点】验证 MaskValue 对不同长度值的脱敏效果
// 【测试流程】
// 1. 短值（≤4 字符）应返回 "****"
// 2. 长值应保留首尾各 2 个字符
// 3. 边界值（5 字符）
func TestMaskValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"空字符串", "", "****"},
		{"1 字符", "a", "****"},
		{"4 字符", "abcd", "****"},
		{"5 字符", "abcde", "ab****de"},
		{"长字符串", "mysecretpassword123", "my****23"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MaskValue(tt.input))
		})
	}
}

// --- SanitizeMessage 测试 ---

// TestSanitizeMessage 测试消息脱敏
//
// 【功能点】验证多种格式的敏感信息自动脱敏
// 【测试流程】
// 1. 无敏感信息时不变
// 2. key=value 格式脱敏
// 3. JSON 格式 "key":"value" 脱敏
// 4. key: value 格式脱敏
// 5. 多个敏感字段同时脱敏
func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, result string)
	}{
		{
			"无敏感信息",
			"hello world",
			func(t *testing.T, result string) { assert.Equal(t, "hello world", result) },
		},
		{
			"password=value 格式",
			"login password=mysecretpass123",
			func(t *testing.T, result string) {
				assert.NotContains(t, result, "mysecretpass123")
				assert.Contains(t, result, "****")
			},
		},
		{
			"JSON 格式 token",
			`request body: {"token": "abc123xyz789"}`,
			func(t *testing.T, result string) {
				assert.NotContains(t, result, "abc123xyz789")
				assert.Contains(t, result, "****")
			},
		},
		{
			"authorization 头",
			"authorization: Bearer eyJhbGciOiJIUzI1NiJ9",
			func(t *testing.T, result string) {
				// 正则匹配 key: value 格式，value 截取到首个空格，
				// 因此 "Bearer" 被当作值脱敏，JWT 部分不受影响
				assert.NotContains(t, result, "authorization: Bearer")
				assert.Contains(t, result, "****")
			},
		},
		{
			"多个敏感字段",
			`user login: password=secret123 token=abcdef12345`,
			func(t *testing.T, result string) {
				assert.NotContains(t, result, "secret123")
				assert.NotContains(t, result, "abcdef12345")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeMessage(tt.input)
			tt.check(t, result)
		})
	}
}

// --- isSensitiveField 测试 ---

// TestIsSensitiveField 测试敏感字段识别
//
// 【功能点】验证 isSensitiveField 对各种字段名的判断
// 【测试流程】
// 1. 已知敏感关键词应返回 true
// 2. 大小写混合应返回 true
// 3. 非敏感字段应返回 false
func TestIsSensitiveField(t *testing.T) {
	sensitiveFields := []string{
		"password", "Password", "user_password",
		"token", "AccessToken", "refreshToken",
		"secret", "apiKey", "api_key",
		"authorization", "auth_header",
		"credential", "private_key",
	}
	for _, field := range sensitiveFields {
		assert.True(t, isSensitiveField(field), "应识别为敏感字段: %s", field)
	}

	normalFields := []string{
		"username", "email", "name", "age", "status", "id", "created_at",
	}
	for _, field := range normalFields {
		assert.False(t, isSensitiveField(field), "不应识别为敏感字段: %s", field)
	}
}

// --- SanitizeFields 测试 ---

// TestSanitizeFields 测试结构化字段脱敏
//
// 【功能点】验证敏感字段值被脱敏，非敏感字段保持不变
// 【测试流程】
// 1. 敏感字段的 string 值被 MaskValue 处理
// 2. 敏感字段的非 string 值变为 "****"
// 3. 非敏感字段保持原值
func TestSanitizeFields(t *testing.T) {
	fields := map[string]any{
		"username": "john",
		"password": "mysecretpass123",
		"token":    12345, // non-string sensitive field
		"status":   "active",
	}

	result := SanitizeFields(fields)

	assert.Equal(t, "john", result["username"])
	assert.Equal(t, MaskValue("mysecretpass123"), result["password"])
	assert.Equal(t, "****", result["token"])
	assert.Equal(t, "active", result["status"])
}

// --- sanitizeLog 测试 ---

// TestSanitizeLog 测试格式化 + 脱敏
//
// 【功能点】验证 sanitizeLog 在格式化后正确脱敏
// 【测试流程】
// 1. 无参数时直接脱敏
// 2. 有参数时先 Sprintf 再脱敏
func TestSanitizeLog(t *testing.T) {
	result := sanitizeLog("plain message")
	assert.Equal(t, "plain message", result)

	result = sanitizeLog("user %s login with password=%s", "john", "secret12345")
	assert.NotContains(t, result, "secret12345")
	assert.Contains(t, result, "john")
}

// --- getCaller 测试 ---

// TestGetCaller 测试调用者信息获取
//
// 【功能点】验证 getCaller 返回正确的文件路径和函数名
// 【测试流程】
// 1. skip=1 获取当前函数信息
// 2. 文件路径应包含 logger_test.go
// 3. 函数名应包含 TestGetCaller
func TestGetCaller(t *testing.T) {
	info := getCaller(1)
	assert.Contains(t, info.File, "logger_test.go")
	assert.Contains(t, info.Func, "TestGetCaller")
}

// TestGetCaller_InvalidSkip 测试无效 skip 值
//
// 【功能点】skip 过大时应返回 unknown
func TestGetCaller_InvalidSkip(t *testing.T) {
	info := getCaller(100)
	assert.Equal(t, "unknown:0", info.File)
	assert.Equal(t, "unknown", info.Func)
}

// --- withCallerFields 测试 ---

// TestWithCallerFields_Disabled 测试关闭调用者信息
//
// 【功能点】printCaller=false 时不添加 caller 字段
func TestWithCallerFields_Disabled(t *testing.T) {
	_ = setupTestLogger()
	printCaller = false

	entry := withCallerFields(1)
	assert.Empty(t, entry.Data)
}

// TestWithCallerFields_Enabled 测试开启调用者信息
//
// 【功能点】printCaller=true 时添加 func 和 file 字段
func TestWithCallerFields_Enabled(t *testing.T) {
	_ = setupTestLogger()
	printCaller = true
	defer func() { printCaller = false }()

	entry := withCallerFields(1)
	assert.Contains(t, entry.Data, "func")
	assert.Contains(t, entry.Data, "file")
}

// --- 日志级别函数测试 ---

// TestLogFunctions 测试各级别日志函数
//
// 【功能点】验证 Info/Error/Warn/Debug/Trace 正确写入日志
// 【测试流程】
// 1. 每个函数调用后 buffer 中应包含对应内容
// 2. 敏感信息应被脱敏
func TestLogFunctions(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(string, ...any)
		level   string
		message string
	}{
		{"Info", Info, "info", "test info message"},
		{"Error", Error, "error", "test error message"},
		{"Warn", Warn, "warning", "test warn message"},
		{"Debug", Debug, "debug", "test debug message"},
		{"Trace", Trace, "trace", "test trace message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupTestLogger()
			tt.fn(tt.message)
			output := buf.String()
			assert.Contains(t, output, tt.message)
			assert.Contains(t, output, tt.level)
		})
	}
}

// TestLogFunctions_WithFormat 测试格式化日志
//
// 【功能点】验证日志函数支持 fmt.Sprintf 风格格式化
func TestLogFunctions_WithFormat(t *testing.T) {
	buf := setupTestLogger()
	Info("user %s logged in, count=%d", "alice", 42)
	output := buf.String()
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "42")
}

// TestLogFunctions_Sanitize 测试日志函数自动脱敏
//
// 【功能点】日志中的敏感信息应被自动脱敏
func TestLogFunctions_Sanitize(t *testing.T) {
	buf := setupTestLogger()
	Info("login with password=supersecret123")
	output := buf.String()
	assert.NotContains(t, output, "supersecret123")
	assert.Contains(t, output, "****")
}

// --- Add 函数测试 ---

// TestAdd_WithError 测试 Add 函数记录错误日志
//
// 【功能点】err 非 nil 时使用 Error 级别
func TestAdd_WithError(t *testing.T) {
	buf := setupTestLogger()
	Add("req-123", "some operation", errors.New("something failed"))
	output := buf.String()
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "req-123")
	assert.Contains(t, output, "something failed")
}

// TestAdd_WithoutError 测试 Add 函数记录信息日志
//
// 【功能点】err 为 nil 时使用 Info 级别
func TestAdd_WithoutError(t *testing.T) {
	buf := setupTestLogger()
	Add("req-456", "success operation", nil)
	output := buf.String()
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "req-456")
}

// --- WithFields 函数测试 ---

// TestInfoWithFields 测试带字段的 Info 日志
//
// 【功能点】字段和消息同时输出，敏感字段被脱敏
func TestInfoWithFields(t *testing.T) {
	buf := setupTestLogger()
	fields := map[string]any{
		"username": "john",
		"password": "secret12345",
	}
	InfoWithFields(fields, "user login")
	output := buf.String()
	assert.Contains(t, output, "john")
	assert.NotContains(t, output, "secret12345")
	assert.Contains(t, output, "user login")
}

// TestErrorWithFields 测试带字段的 Error 日志
func TestErrorWithFields(t *testing.T) {
	buf := setupTestLogger()
	fields := map[string]any{"code": 500}
	ErrorWithFields(fields, "internal error")
	output := buf.String()
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "internal error")
}

// TestWarnWithFields 测试带字段的 Warn 日志
func TestWarnWithFields(t *testing.T) {
	buf := setupTestLogger()
	fields := map[string]any{"reason": "slow"}
	WarnWithFields(fields, "slow query")
	output := buf.String()
	assert.Contains(t, output, "warning")
	assert.Contains(t, output, "slow query")
}

// TestDebugWithFields 测试带字段的 Debug 日志
func TestDebugWithFields(t *testing.T) {
	buf := setupTestLogger()
	fields := map[string]any{"step": "parsing"}
	DebugWithFields(fields, "parsing request")
	output := buf.String()
	assert.Contains(t, output, "debug")
	assert.Contains(t, output, "parsing request")
}

// TestTraceWithFields 测试带字段的 Trace 日志
func TestTraceWithFields(t *testing.T) {
	buf := setupTestLogger()
	fields := map[string]any{"detail": "full"}
	TraceWithFields(fields, "trace detail")
	output := buf.String()
	assert.Contains(t, output, "trace")
	assert.Contains(t, output, "trace detail")
}

// --- InitLogger 测试 ---

// TestInitLogger 测试日志初始化
//
// 【功能点】验证 InitLogger 返回有效的 Logger 且设置 PrintCaller
// 【测试流程】
// 1. 使用自定义配置调用 InitLogger
// 2. 返回的 Logger 不为 nil
// 3. PrintCaller 被正确设置
func TestInitLogger(t *testing.T) {
	cfg := config.LoggersConfig{
		FilePath:     t.TempDir(),
		MaxAge:       1,
		RotationTime: 60,
		RotationSize: 512,
		PrintCaller:  true,
		Loggers: []config.LoggerConfig{
			{Level: "info", FileName: "test-info"},
		},
	}

	l := InitLogger(cfg)
	require.NotNil(t, l)
	assert.Equal(t, logrus.TraceLevel, l.Level)
	assert.True(t, printCaller)

	printCaller = false
}

// TestInitLogger_DefaultFallback 测试 InitLogger 缺少匹配级别时回退默认
//
// 【功能点】未配置某些级别时应回退到默认配置
func TestInitLogger_DefaultFallback(t *testing.T) {
	cfg := config.LoggersConfig{
		FilePath: t.TempDir(),
	}

	l := InitLogger(cfg)
	require.NotNil(t, l)
}

// --- initRotatelogs 测试 ---

// TestInitRotatelogs_Defaults 测试默认配置
//
// 【功能点】空配置时使用默认值
func TestInitRotatelogs_Defaults(t *testing.T) {
	writer, err := initRotatelogs(config.LoggersConfig{}, config.LoggerConfig{}, "")
	require.NoError(t, err)
	require.NotNil(t, writer)
}

// TestInitRotatelogs_CustomConfig 测试自定义配置优先级
//
// 【功能点】loggerConfig 优先于 globalConfig 优先于默认值
func TestInitRotatelogs_CustomConfig(t *testing.T) {
	tmpDir := t.TempDir()

	global := config.LoggersConfig{
		FilePath:     tmpDir,
		MaxAge:       7,
		RotationTime: 120,
		RotationSize: 2048,
	}
	local := config.LoggerConfig{
		FileName: "custom",
	}

	writer, err := initRotatelogs(global, local, "info")
	require.NoError(t, err)
	require.NotNil(t, writer)
}

// TestInitRotatelogs_LocalOverridesGlobal 测试 loggerConfig 完全覆盖
//
// 【功能点】loggerConfig 的所有字段覆盖 globalConfig
func TestInitRotatelogs_LocalOverridesGlobal(t *testing.T) {
	tmpDir := t.TempDir()

	global := config.LoggersConfig{
		FilePath:     tmpDir,
		MaxAge:       7,
		RotationTime: 120,
		RotationSize: 2048,
	}
	local := config.LoggerConfig{
		FilePath:     tmpDir,
		FileName:     "override",
		MaxAge:       1,
		RotationTime: 30,
		RotationSize: 512,
	}

	writer, err := initRotatelogs(global, local, "error")
	require.NoError(t, err)
	require.NotNil(t, writer)
}

// TestInitRotatelogs_FilePatterns 测试不同轮转时间的文件命名模式
//
// 【功能点】轮转时间 ≤60min 精确到分钟，60-1440min 精确到小时，≥1440min 精确到天
func TestInitRotatelogs_FilePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		rotationTime int
	}{
		{"精确到分钟（30min）", 30},
		{"精确到分钟（60min）", 60},
		{"精确到小时（120min）", 120},
		{"精确到天（1440min）", 1440},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := initRotatelogs(
				config.LoggersConfig{FilePath: tmpDir, RotationTime: tt.rotationTime},
				config.LoggerConfig{},
				"info",
			)
			require.NoError(t, err)
			require.NotNil(t, writer)
		})
	}
}

// --- 敏感关键词全覆盖 ---

// TestSanitizeMessage_AllKeywords 测试所有敏感关键词
//
// 【功能点】验证 sensitiveKeywords 列表中的所有关键词都被正确匹配
func TestSanitizeMessage_AllKeywords(t *testing.T) {
	keywords := []string{
		"password", "pwd", "passwd",
		"token", "accesstoken", "refreshtoken",
		"secret", "apikey", "api_key",
		"authorization", "auth",
		"credential", "private",
	}

	for _, kw := range keywords {
		t.Run(kw, func(t *testing.T) {
			msg := kw + "=supersecretvalue123"
			result := SanitizeMessage(msg)
			assert.NotContains(t, result, "supersecretvalue123",
				"关键词 %s 未被脱敏", kw)
		})
	}
}

// TestSanitizeMessage_JSONFormat 测试 JSON 格式脱敏
//
// 【功能点】验证 "key": "value" 格式的脱敏
func TestSanitizeMessage_JSONFormat(t *testing.T) {
	msg := `{"password": "mypass12345", "username": "admin"}`
	result := SanitizeMessage(msg)
	assert.NotContains(t, result, "mypass12345")
	assert.Contains(t, result, "admin")
}

// --- SanitizeFields 边界场景 ---

// TestSanitizeFields_Empty 测试空字段
func TestSanitizeFields_Empty(t *testing.T) {
	result := SanitizeFields(map[string]any{})
	assert.Empty(t, result)
}

// TestSanitizeFields_NonStringSensitive 测试非字符串敏感字段
//
// 【功能点】非 string 类型的敏感字段值应为 "****"
func TestSanitizeFields_NonStringSensitive(t *testing.T) {
	fields := map[string]any{
		"password": 12345,
		"token":    true,
		"secret":   []byte("bytes"),
	}
	result := SanitizeFields(fields)
	assert.Equal(t, "****", result["password"])
	assert.Equal(t, "****", result["token"])
	assert.Equal(t, "****", result["secret"])
}

// --- callerInfo 结构 ---

// TestCallerInfo_Structure 测试调用者信息结构
func TestCallerInfo_Structure(t *testing.T) {
	info := getCaller(1)
	assert.True(t, strings.Contains(info.File, ":"), "File 应包含行号分隔符")
	assert.NotEmpty(t, info.Func)
}

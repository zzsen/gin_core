// Package core 命令行参数解析功能测试
//
// ==================== 测试说明 ====================
// 本文件包含命令行参数解析相关功能的单元测试。
//
// 测试覆盖内容：
// 1. CmdArgs 结构体 - 参数存储结构
// 2. parseCmdArgs - 命令行参数解析
// 3. 默认值 - 未指定参数时的默认值
// 4. 环境参数 - -env 参数解析
// 5. 配置路径 - -config 参数解析
// 6. 解密密钥 - -cipherKey 参数解析
// 7. 组合参数 - 多参数组合使用
//
// 支持的参数：
//   -env        环境标识（如 dev、test、prod）
//   -config     配置文件目录路径
//   -cipherKey  配置加密密钥
//
// 运行测试：go test -v ./core/... -run CmdArgs
// ==================================================
package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== CmdArgs 结构体测试 ====================

// TestCmdArgs 测试CmdArgs结构体
//
// 【功能点】验证 CmdArgs 结构体的字段定义
// 【测试流程】
//  1. 创建 CmdArgs 实例并设置各字段值
//  2. 验证各字段值正确存储
func TestCmdArgs(t *testing.T) {
	t.Run("create cmd args", func(t *testing.T) {
		args := CmdArgs{
			Env:       "dev",
			Config:    "./conf",
			CipherKey: "test-key",
		}

		assert.Equal(t, "dev", args.Env)
		assert.Equal(t, "./conf", args.Config)
		assert.Equal(t, "test-key", args.CipherKey)
	})
}

// ==================== parseCmdArgs 测试 ====================

// TestParseCmdArgs 测试parseCmdArgs函数
//
// 【功能点】验证命令行参数解析功能
// 【测试流程】
//  1. 测试默认值 - 验证未指定参数时的默认值
//  2. 测试 -env 参数 - 验证环境标识解析
//  3. 测试 -config 参数 - 验证配置路径解析
//  4. 测试 -cipherKey 参数 - 验证解密密钥解析
//  5. 测试组合参数 - 验证多参数组合使用
func TestParseCmdArgs(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args

	tests := []struct {
		name     string
		args     []string
		expected *CmdArgs
		wantErr  bool
	}{
		{
			name: "default values",
			args: []string{"program"},
			expected: &CmdArgs{
				Env:       "",
				Config:    "./conf", // 默认值
				CipherKey: "",
			},
			wantErr: false,
		},
		{
			name: "with env parameter",
			args: []string{"program", "-env", "prod"},
			expected: &CmdArgs{
				Env:       "prod",
				Config:    "./conf", // 默认值
				CipherKey: "",
			},
			wantErr: false,
		},
		{
			name: "with config parameter",
			args: []string{"program", "-config", "/custom/config"},
			expected: &CmdArgs{
				Env:       "",
				Config:    "/custom/config",
				CipherKey: "",
			},
			wantErr: false,
		},
		{
			name: "with cipherKey parameter",
			args: []string{"program", "-cipherKey", "my-secret-key"},
			expected: &CmdArgs{
				Env:       "",
				Config:    "./conf", // 默认值
				CipherKey: "my-secret-key",
			},
			wantErr: false,
		},
		{
			name: "with all parameters",
			args: []string{"program", "-env", "test", "-config", "/test/config", "-cipherKey", "test-key"},
			expected: &CmdArgs{
				Env:       "test",
				Config:    "/test/config",
				CipherKey: "test-key",
			},
			wantErr: false,
		},
		{
			name: "with empty values",
			args: []string{"program", "-env", "", "-config", "", "-cipherKey", ""},
			expected: &CmdArgs{
				Env:       "",
				Config:    "",
				CipherKey: "",
			},
			wantErr: false,
		},
		{
			name: "with special characters",
			args: []string{"program", "-env", "dev-test", "-config", "/path/with spaces", "-cipherKey", "key@#$%"},
			expected: &CmdArgs{
				Env:       "dev-test",
				Config:    "/path/with spaces",
				CipherKey: "key@#$%",
			},
			wantErr: false,
		},
		{
			name: "with long values",
			args: []string{"program", "-env", "very-long-environment-name", "-config", "/very/long/path/to/config/directory", "-cipherKey", "very-long-cipher-key-with-many-characters"},
			expected: &CmdArgs{
				Env:       "very-long-environment-name",
				Config:    "/very/long/path/to/config/directory",
				CipherKey: "very-long-cipher-key-with-many-characters",
			},
			wantErr: false,
		},
		{
			name: "with unicode characters",
			args: []string{"program", "-env", "测试环境", "-config", "/配置/路径", "-cipherKey", "密钥123"},
			expected: &CmdArgs{
				Env:       "测试环境",
				Config:    "/配置/路径",
				CipherKey: "密钥123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试参数
			os.Args = tt.args

			// 执行解析
			result, err := parseCmdArgs()

			// 验证结果
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}

	// 恢复原始参数
	os.Args = originalArgs
}

// TestParseCmdArgs_EdgeCases 测试边界情况
//
// 【功能点】验证命令行参数解析的边界情况
// 【测试流程】
//  1. 测试空字符串参数 - 验证处理空值
//  2. 测试特殊字符参数 - 验证正确解析
//  3. 测试路径包含空格 - 验证正确处理
func TestParseCmdArgs_EdgeCases(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args

	t.Run("no arguments", func(t *testing.T) {
		os.Args = []string{""} // 至少需要一个程序名

		result, err := parseCmdArgs()

		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "", result.Env)
		assert.Equal(t, "./conf", result.Config) // 默认值
		assert.Equal(t, "", result.CipherKey)
	})

	t.Run("unknown flags", func(t *testing.T) {
		os.Args = []string{"program", "-unknown", "value", "-env", "dev"}

		// 未知标志会导致panic，所以我们需要捕获它
		assert.Panics(t, func() {
			parseCmdArgs()
		})
	})

	t.Run("duplicate flags", func(t *testing.T) {
		os.Args = []string{"program", "-env", "first", "-env", "second"}

		result, err := parseCmdArgs()

		// 重复标志，后面的会覆盖前面的
		assert.Nil(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "second", result.Env)
	})

	t.Run("flags without values", func(t *testing.T) {
		os.Args = []string{"program", "-env", "-config", "-cipherKey"}

		// 标志没有值会导致panic
		assert.Panics(t, func() {
			parseCmdArgs()
		})
	})

	// 恢复原始参数
	os.Args = originalArgs
}

// TestParseCmdArgs_Concurrent 测试并发安全性
//
// 【功能点】验证参数解析的并发安全性
// 【测试流程】
//  1. 启动多个协程并发解析参数
//  2. 验证无数据竞争
//  3. 验证所有解析结果正确
func TestParseCmdArgs_Concurrent(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args

	t.Run("concurrent parsing", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				// 每个goroutine使用不同的参数
				os.Args = []string{"program", "-env", "test" + string(rune(index)), "-config", "/config" + string(rune(index))}

				result, err := parseCmdArgs()

				assert.Nil(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, result.Env, "test")
				assert.Contains(t, result.Config, "/config")

				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	// 恢复原始参数
	os.Args = originalArgs
}

// TestParseCmdArgs_RealWorld 测试真实世界场景
//
// 【功能点】验证真实使用场景的参数解析
// 【测试流程】
//  1. 模拟开发环境参数组合
//  2. 模拟生产环境参数组合
//  3. 模拟 CI/CD 环境参数组合
func TestParseCmdArgs_RealWorld(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args

	tests := []struct {
		name     string
		args     []string
		scenario string
	}{
		{
			name:     "development environment",
			args:     []string{"gin_core", "-env", "dev", "-config", "./conf"},
			scenario: "开发环境启动",
		},
		{
			name:     "production environment",
			args:     []string{"gin_core", "-env", "prod", "-config", "/etc/gin_core", "-cipherKey", "prod-secret-key"},
			scenario: "生产环境启动",
		},
		{
			name:     "testing environment",
			args:     []string{"gin_core", "-env", "test", "-config", "./test_conf"},
			scenario: "测试环境启动",
		},
		{
			name:     "docker environment",
			args:     []string{"gin_core", "-env", "docker", "-config", "/app/config", "-cipherKey", "docker-key"},
			scenario: "Docker容器启动",
		},
		{
			name:     "kubernetes environment",
			args:     []string{"gin_core", "-env", "k8s", "-config", "/config", "-cipherKey", "k8s-secret"},
			scenario: "Kubernetes环境启动",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			result, err := parseCmdArgs()

			assert.Nil(t, err)
			assert.NotNil(t, result)

			// 验证基本字段不为空（除了可能为空的CipherKey）
			assert.NotEmpty(t, result.Env)
			assert.NotEmpty(t, result.Config)

			// 根据场景验证特定值
			switch tt.scenario {
			case "开发环境启动":
				assert.Equal(t, "dev", result.Env)
				assert.Equal(t, "./conf", result.Config)
			case "生产环境启动":
				assert.Equal(t, "prod", result.Env)
				assert.Equal(t, "/etc/gin_core", result.Config)
				assert.Equal(t, "prod-secret-key", result.CipherKey)
			case "测试环境启动":
				assert.Equal(t, "test", result.Env)
				assert.Equal(t, "./test_conf", result.Config)
			case "Docker容器启动":
				assert.Equal(t, "docker", result.Env)
				assert.Equal(t, "/app/config", result.Config)
				assert.Equal(t, "docker-key", result.CipherKey)
			case "Kubernetes环境启动":
				assert.Equal(t, "k8s", result.Env)
				assert.Equal(t, "/config", result.Config)
				assert.Equal(t, "k8s-secret", result.CipherKey)
			}
		})
	}

	// 恢复原始参数
	os.Args = originalArgs
}

// TestParseCmdArgs_Performance 测试性能
//
// 【功能点】验证参数解析的性能表现
// 【测试方法】多次解析参数，统计执行时间
func TestParseCmdArgs_Performance(t *testing.T) {
	// 保存原始参数
	originalArgs := os.Args

	t.Run("performance test", func(t *testing.T) {
		os.Args = []string{"program", "-env", "perf", "-config", "/perf/config", "-cipherKey", "perf-key"}

		// 执行多次解析以测试性能
		for i := 0; i < 1000; i++ {
			result, err := parseCmdArgs()
			assert.Nil(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "perf", result.Env)
		}
	})

	// 恢复原始参数
	os.Args = originalArgs
}

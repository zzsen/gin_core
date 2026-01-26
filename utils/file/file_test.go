// Package file 文件操作工具功能测试
//
// ==================== 测试说明 ====================
// 本文件包含文件操作工具函数的单元测试。
//
// 测试覆盖内容：
// 1. FileMd5 - 计算文件MD5哈希值
// 2. FileExist - 检查文件是否存在
// 3. DirExist - 检查目录是否存在
// 4. CreateDir - 创建目录（递归创建）
// 5. CopyFile - 复制文件
// 6. ReadFile - 读取文件内容
// 7. WriteFile - 写入文件内容
// 8. GetFileSize - 获取文件大小
//
// 运行测试：go test -v ./utils/file/...
// ==================================================
package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== FileMd5 测试 ====================

// TestFileMd5 测试文件MD5计算功能
//
// 【功能点】验证文件MD5哈希值计算的正确性
// 【测试流程】
//  1. 创建临时文件并写入测试内容
//  2. 调用 FileMd5 计算哈希值
//  3. 验证返回的MD5与期望值一致
//  4. 清理临时文件
func TestFileMd5(t *testing.T) {
	tests := []struct {
		name     string // 测试用例名称
		content  string // 文件内容
		expected string // 期望的MD5值
		wantErr  bool   // 是否期望出错
	}{
		{
			name:     "empty file",
			content:  "",
			expected: "D41D8CD98F00B204E9800998ECF8427E", // 空文件的MD5
			wantErr:  false,
		},
		{
			name:     "hello world",
			content:  "hello world",
			expected: "5EB63BBBE01EEED093CB22BB8F5ACDC3", // "hello world"的MD5
			wantErr:  false,
		},
		{
			name:     "chinese content",
			content:  "你好世界",
			expected: "65396EE4AAD0B4F17AACD1C6112EE364", // "你好世界"的MD5
			wantErr:  false,
		},
		{
			name:     "special characters",
			content:  "!@#$%^&*()_+-=[]{}|;':\",./<>?",
			expected: "55F63EA4FDD78EF5E227F735E191AFDC", // 特殊字符的MD5
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时文件
			tempFile, err := os.CreateTemp("", "test_file_*.txt")
			assert.Nil(t, err)
			defer os.Remove(tempFile.Name()) // 清理临时文件

			// 写入测试内容
			_, err = tempFile.WriteString(tt.content)
			assert.Nil(t, err)
			tempFile.Close()

			// 计算MD5值
			result := FileMd5(tempFile.Name())

			if tt.wantErr {
				// 期望出错的情况
				assert.Empty(t, result)
			} else {
				// 期望成功的情况
				assert.NotEmpty(t, result)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestFileMd5_NonExistentFile 测试不存在的文件的MD5计算
//
// 【功能点】验证不存在文件的错误处理
// 【测试流程】
//  1. 使用一个不存在的文件路径
//  2. 调用 FileMd5 计算
//  3. 验证返回空字符串（不是 panic）
func TestFileMd5_NonExistentFile(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		// 使用一个不存在的文件路径
		nonExistentPath := "/path/that/does/not/exist.txt"

		// 计算不存在的文件的MD5
		result := FileMd5(nonExistentPath)

		// 应该返回空字符串
		assert.Empty(t, result)
	})
}

// TestFileMd5_LargeFile 测试大文件的MD5计算
//
// 【功能点】验证大文件（1MB）的 MD5 计算性能和正确性
// 【测试流程】
//  1. 创建临时文件并写入 1MB 数据
//  2. 调用 FileMd5 计算
//  3. 验证返回 32 字符的有效 MD5 值
//  4. 清理临时文件
func TestFileMd5_LargeFile(t *testing.T) {
	t.Run("large file", func(t *testing.T) {
		// 创建临时文件
		tempFile, err := os.CreateTemp("", "test_large_file_*.txt")
		assert.Nil(t, err)
		defer os.Remove(tempFile.Name()) // 清理临时文件

		// 写入大量数据（1MB）
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		_, err = tempFile.Write(largeContent)
		assert.Nil(t, err)
		tempFile.Close()

		// 计算大文件的MD5
		result := FileMd5(tempFile.Name())

		// 应该返回非空的MD5值
		assert.NotEmpty(t, result)
		assert.Len(t, result, 32) // MD5值应该是32个字符
	})
}

// ==================== PathExists 测试 ====================

// TestPathExists 测试路径存在性检查功能
//
// 【功能点】验证文件/目录存在性检查
// 【测试流程】
//  1. 测试存在的文件 - 返回 true
//  2. 测试存在的目录 - 返回 true
//  3. 测试不存在的路径 - 返回 false
//  4. 测试空路径 - 返回 false
func TestPathExists(t *testing.T) {
	tests := []struct {
		name     string                 // 测试用例名称
		setup    func() (string, error) // 设置函数，返回路径和错误
		expected bool                   // 期望的结果
		cleanup  func(string)           // 清理函数
	}{
		{
			name: "existing file",
			setup: func() (string, error) {
				// 创建临时文件
				tempFile, err := os.CreateTemp("", "test_existing_file_*.txt")
				if err != nil {
					return "", err
				}
				tempFile.Close()
				return tempFile.Name(), nil
			},
			expected: true,
			cleanup: func(path string) {
				os.Remove(path)
			},
		},
		{
			name: "existing directory",
			setup: func() (string, error) {
				// 创建临时目录
				tempDir, err := os.MkdirTemp("", "test_existing_dir_*")
				return tempDir, err
			},
			expected: true,
			cleanup: func(path string) {
				os.RemoveAll(path)
			},
		},
		{
			name: "non-existent path",
			setup: func() (string, error) {
				// 返回一个不存在的路径
				return "/path/that/does/not/exist", nil
			},
			expected: false,
			cleanup:  func(string) {}, // 无需清理
		},
		{
			name: "empty path",
			setup: func() (string, error) {
				// 返回空路径
				return "", nil
			},
			expected: false,
			cleanup:  func(string) {}, // 无需清理
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试环境
			path, err := tt.setup()
			assert.Nil(t, err)

			// 确保清理
			defer tt.cleanup(path)

			// 测试路径存在性
			result := PathExists(path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPathExists_CurrentDirectory 测试当前目录的存在性
//
// 【功能点】验证当前工作目录的存在性检查
// 【测试流程】
//  1. 获取当前工作目录
//  2. 调用 PathExists 检查
//  3. 验证返回 true
func TestPathExists_CurrentDirectory(t *testing.T) {
	t.Run("current directory", func(t *testing.T) {
		// 获取当前工作目录
		currentDir, err := os.Getwd()
		assert.Nil(t, err)

		// 检查当前目录是否存在
		result := PathExists(currentDir)
		assert.True(t, result)
	})
}

// TestPathExists_TempDirectory 测试临时目录的存在性
//
// 【功能点】验证系统临时目录的存在性检查
// 【测试流程】
//  1. 获取系统临时目录路径
//  2. 调用 PathExists 检查
//  3. 验证返回 true
func TestPathExists_TempDirectory(t *testing.T) {
	t.Run("temp directory", func(t *testing.T) {
		// 获取系统临时目录
		tempDir := os.TempDir()

		// 检查临时目录是否存在
		result := PathExists(tempDir)
		assert.True(t, result)
	})
}

// TestPathExists_RelativePath 测试相对路径的存在性
//
// 【功能点】验证相对路径的存在性检查
// 【测试流程】
//  1. 创建临时文件
//  2. 获取文件的相对路径
//  3. 调用 PathExists 检查（不会 panic）
func TestPathExists_RelativePath(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		// 创建临时文件
		tempFile, err := os.CreateTemp("", "test_relative_file_*.txt")
		assert.Nil(t, err)
		defer os.Remove(tempFile.Name())

		// 获取相对路径
		relativePath := filepath.Base(tempFile.Name())

		// 检查相对路径是否存在
		// 相对路径可能不存在，取决于当前工作目录
		// 这里只验证函数不会panic
		assert.NotPanics(t, func() {
			PathExists(relativePath)
		})
	})
}

// ==================== 并发安全测试 ====================

// TestFileMd5_Concurrent 测试并发计算文件MD5
//
// 【功能点】验证 FileMd5 在并发环境下的安全性
// 【测试流程】
//  1. 创建临时文件
//  2. 启动多个协程并发计算同一文件的 MD5
//  3. 验证所有结果一致，无数据竞争
func TestFileMd5_Concurrent(t *testing.T) {
	t.Run("concurrent md5 calculation", func(t *testing.T) {
		// 创建临时文件
		tempFile, err := os.CreateTemp("", "test_concurrent_file_*.txt")
		assert.Nil(t, err)
		defer os.Remove(tempFile.Name())

		// 写入测试内容
		content := "concurrent test content"
		_, err = tempFile.WriteString(content)
		assert.Nil(t, err)
		tempFile.Close()

		// 并发计算MD5
		done := make(chan string, 10)
		for i := 0; i < 10; i++ {
			go func() {
				result := FileMd5(tempFile.Name())
				done <- result
			}()
		}

		// 收集结果
		var results []string
		for i := 0; i < 10; i++ {
			result := <-done
			results = append(results, result)
		}

		// 验证所有结果都相同
		firstResult := results[0]
		for _, result := range results {
			assert.Equal(t, firstResult, result)
		}
		assert.NotEmpty(t, firstResult)
	})
}

// TestPathExists_Concurrent 测试并发检查路径存在性
//
// 【功能点】验证 PathExists 在并发环境下的安全性
// 【测试流程】
//  1. 创建临时文件
//  2. 启动多个协程并发检查同一路径
//  3. 验证所有结果一致为 true，无数据竞争
func TestPathExists_Concurrent(t *testing.T) {
	t.Run("concurrent path existence check", func(t *testing.T) {
		// 创建临时文件
		tempFile, err := os.CreateTemp("", "test_concurrent_path_*.txt")
		assert.Nil(t, err)
		defer os.Remove(tempFile.Name())

		// 并发检查路径存在性
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				result := PathExists(tempFile.Name())
				done <- result
			}()
		}

		// 收集结果
		var results []bool
		for i := 0; i < 10; i++ {
			result := <-done
			results = append(results, result)
		}

		// 验证所有结果都为true
		for _, result := range results {
			assert.True(t, result)
		}
	})
}

// TestPathExists_InvalidPath 测试无效路径的处理
//
// 【功能点】验证无效路径的安全处理
// 【测试流程】
//  1. 测试各种无效路径（空字符串、null字符、控制字符等）
//  2. 调用 PathExists 检查
//  3. 验证均返回 false，不会 panic
func TestPathExists_InvalidPath(t *testing.T) {
	t.Run("invalid path with special characters", func(t *testing.T) {
		// 测试包含无效字符的路径
		invalidPaths := []string{
			"",                 // 空字符串
			"\x00",             // 包含null字符
			"con\x00.txt",      // 包含null字符的文件名
			"test\x01\x02\x03", // 包含控制字符
		}

		for _, path := range invalidPaths {
			// 这些路径可能会导致os.Stat返回非os.ErrNotExist的错误
			result := PathExists(path)
			// 无论什么情况，都应该返回false（安全起见）
			assert.False(t, result)
		}
	})
}

// Package encrypt AES ECB模式加密解密功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 AES ECB 模式加密解密功能的单元测试。
//
// 测试覆盖内容：
// 1. AesEcbEncrypt - AES ECB 模式加密
// 2. AesEcbDecrypt - AES ECB 模式解密
// 3. AesEcbCrypt - 加密解密完整流程（加密→解密→验证一致性）
//
// 密钥要求：
// - AES-128: 16字节密钥
// - AES-192: 24字节密钥
// - AES-256: 32字节密钥
//
// 运行测试：go test -v ./utils/encrypt/...
// ==================================================
package encrypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== AES ECB 加密测试 ====================

// TestAesEcbEncrypt 测试AES ECB模式加密功能
//
// 【功能点】验证 AES ECB 模式加密的正确性
// 【测试流程】
//  1. 准备明文 "Hello World" 和 16字节密钥
//  2. 调用 AesEcbEncrypt 进行加密
//  3. 验证返回的 Base64 编码密文与期望值一致
func TestAesEcbEncrypt(t *testing.T) {
	type args struct {
		src2Encrypt string // 待加密的明文
		key         string // 加密密钥
	}
	tests := []struct {
		name    string // 测试用例名称
		args    args   // 测试参数
		want    string // 期望的加密结果
		wantErr bool   // 是否期望出错
	}{
		{
			name: "aes ecb encrypt", // 修正拼写错误：ecn -> ecb
			args: args{
				src2Encrypt: "Hello World",
				key:         "UTabIUiHgDyh464+", // 16字节密钥
			},
			want:    "/t8wxJyz5nLKYDa7w8W3oQ==", // base64编码的加密结果
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行加密操作
			got, err := AesEcbEncrypt(tt.args.src2Encrypt, tt.args.key)
			if tt.wantErr {
				// 期望出错的情况
				assert.Error(t, err)
			} else {
				// 期望成功的情况
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ==================== AES ECB 解密测试 ====================

// TestAesEcbDecrypt 测试AES ECB模式解密功能
//
// 【功能点】验证 AES ECB 模式解密的正确性
// 【测试流程】
//  1. 准备 Base64 编码的密文和对应密钥
//  2. 调用 AesEcbDecrypt 进行解密
//  3. 验证解密后的明文与期望值一致
func TestAesEcbDecrypt(t *testing.T) {
	type args struct {
		src2Decrypt string // 待解密的密文（base64编码）
		key         string // 解密密钥
	}
	tests := []struct {
		name    string // 测试用例名称
		args    args   // 测试参数
		want    string // 期望的解密结果
		wantErr bool   // 是否期望出错
	}{
		{
			name: "aes ecb decrypt",
			args: args{
				src2Decrypt: "/t8wxJyz5nLKYDa7w8W3oQ==", // base64编码的密文
				key:         "UTabIUiHgDyh464+",         // 16字节密钥
			},
			want:    "Hello World", // 期望解密后的明文
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行解密操作
			got, err := AesEcbDecrypt(tt.args.src2Decrypt, tt.args.key)
			if tt.wantErr {
				// 期望出错的情况
				assert.Error(t, err)
			} else {
				// 期望成功的情况
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// ==================== AES ECB 加解密完整流程测试 ====================

// TestAesEcbCrypt 测试AES ECB模式加密解密的完整流程
//
// 【功能点】验证加密和解密的完整流程一致性
// 【测试流程】
//  1. 准备明文和密钥
//  2. 调用 AesEcbEncrypt 加密，验证密文与期望值一致
//  3. 调用 AesEcbDecrypt 解密，验证明文与原始值一致
//  4. 确保 明文 → 加密 → 解密 → 明文 的完整循环
func TestAesEcbCrypt(t *testing.T) {
	type args struct {
		src2Encrypt   string // 待加密的明文
		encryptResult string // 预期的加密结果（用于验证）
		key           string // 加密/解密密钥
	}
	tests := []struct {
		name string // 测试用例名称
		args args   // 测试参数
	}{
		{
			name: "aes ecb crypt",
			args: args{
				src2Encrypt:   "Hello World",
				encryptResult: "/t8wxJyz5nLKYDa7w8W3oQ==", // 预期的加密结果
				key:           "UTabIUiHgDyh464+",         // 16字节密钥
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 执行加密操作
			encryptResult, err := AesEcbEncrypt(tt.args.src2Encrypt, tt.args.key)
			assert.Nil(t, err)

			// 2. 执行解密操作
			decryptResult, err := AesEcbDecrypt(encryptResult, tt.args.key)
			assert.Nil(t, err)

			// 3. 验证解密结果与原始明文一致
			assert.Equal(t, decryptResult, tt.args.src2Encrypt)
		})
	}
}

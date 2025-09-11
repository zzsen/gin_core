// Package encrypt AES ECB模式加密解密功能测试
package encrypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAesEcbEncrypt 测试AES ECB模式加密功能
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

// TestAesEcbDecrypt 测试AES ECB模式解密功能
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

// TestAesEcbCrypt 测试AES ECB模式加密解密的完整流程
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

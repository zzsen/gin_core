// Package encrypt 提供AES ECB模式的加密解密功能
package encrypt

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"fmt"
)

// AesEcbEncrypt AES ECB模式加密
// plainText: 待加密的明文
// key: 加密密钥，长度必须为16、24或32字节
// isPad: 是否使用padding填充，默认为true
// 返回: base64编码的加密结果和错误信息
func AesEcbEncrypt(plainText string, key string, isPad ...bool) (string, error) {
	// 将明文转换为字节数组
	plainBytes := []byte(plainText)
	if len(plainBytes) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	// 创建AES密码块
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// 根据参数决定是否使用padding填充
	if len(isPad) > 0 && !isPad[0] {
		plainBytes = noPadding(plainBytes)
	} else {
		plainBytes = padding(plainBytes)
	}

	// 分块加密
	buf := make([]byte, aes.BlockSize)
	encrypted := make([]byte, 0)
	for i := 0; i < len(plainBytes); i += aes.BlockSize {
		block.Encrypt(buf, plainBytes[i:i+aes.BlockSize])
		encrypted = append(encrypted, buf...)
	}
	// 返回base64编码的加密结果
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// AesEcbDecrypt AES ECB模式解密
// cryptText: base64编码的密文
// key: 解密密钥，长度必须为16、24或32字节
// isPad: 是否使用padding填充，默认为true
// 返回: 解密后的明文和错误信息
func AesEcbDecrypt(cryptText string, key string, isPad ...bool) (string, error) {
	// 将base64编码的密文解码为字节数组
	cryptBytes, err := base64.StdEncoding.DecodeString(cryptText)
	if err != nil {
		return "", err
	}
	if len(cryptBytes) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	// 创建AES密码块
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// 分块解密
	buf := make([]byte, aes.BlockSize)
	decrypted := make([]byte, 0)
	for i := 0; i < len(cryptBytes); i += aes.BlockSize {
		block.Decrypt(buf, cryptBytes[i:i+aes.BlockSize])
		decrypted = append(decrypted, buf...)
	}

	// 根据参数决定是否去除padding填充
	if len(isPad) > 0 && !isPad[0] {
		decrypted = unNoPadding(decrypted)
	} else {
		decrypted = unPadding(decrypted)
	}

	return string(decrypted), nil
}

// noPadding 零填充模式
// 将数据填充到AES块大小的整数倍，不足部分用0填充
// src: 待填充的字节数组
// 返回: 填充后的字节数组
func noPadding(src []byte) []byte {
	count := aes.BlockSize - len(src)%aes.BlockSize
	if len(src)%aes.BlockSize == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, count)...)
	}
}

// unNoPadding 去除零填充
// 从数据末尾开始去除所有的0填充
// src: 待去除填充的字节数组
// 返回: 去除填充后的字节数组
func unNoPadding(src []byte) []byte {
	for i := len(src) - 1; i >= 0; i-- {
		if src[i] != 0 {
			return src[:i+1]
		}
	}
	return src
}

// padding PKCS7填充模式
// 将数据填充到AES块大小的整数倍，填充内容为填充长度的值
// src: 待填充的字节数组
// 返回: 填充后的字节数组
func padding(src []byte) []byte {
	count := aes.BlockSize - len(src)%aes.BlockSize
	padding := bytes.Repeat([]byte{byte(0)}, count)
	padding[count-1] = byte(count)
	return append(src, padding...)
}

// unPadding 去除PKCS7填充
// 根据最后一个字节的值确定填充长度，去除填充内容
// src: 待去除填充的字节数组
// 返回: 去除填充后的字节数组
func unPadding(src []byte) []byte {
	l := len(src)
	p := int(src[l-1])
	return src[:l-p]
}

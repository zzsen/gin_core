package encrypt

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"fmt"
)

func AesEcbEncrypt(plainText string, key string, isPad ...bool) (string, error) {
	plainBytes := []byte(plainText)
	if len(plainBytes) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	if len(isPad) > 0 && !isPad[0] {
		plainBytes = noPadding(plainBytes)
	} else {
		plainBytes = padding(plainBytes)
	}

	buf := make([]byte, aes.BlockSize)
	encrypted := make([]byte, 0)
	for i := 0; i < len(plainBytes); i += aes.BlockSize {
		block.Encrypt(buf, plainBytes[i:i+aes.BlockSize])
		encrypted = append(encrypted, buf...)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func AesEcbDecrypt(cryptText string, key string, isPad ...bool) (string, error) {
	cryptBytes, err := base64.StdEncoding.DecodeString(cryptText)
	if err != nil {
		return "", err
	}
	if len(cryptBytes) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	buf := make([]byte, aes.BlockSize)
	decrypted := make([]byte, 0)
	for i := 0; i < len(cryptBytes); i += aes.BlockSize {
		block.Decrypt(buf, cryptBytes[i:i+aes.BlockSize])
		decrypted = append(decrypted, buf...)
	}

	if len(isPad) > 0 && !isPad[0] {
		decrypted = unNoPadding(decrypted)
	} else {
		decrypted = unPadding(decrypted)
	}

	return string(decrypted), nil
}

// nopadding模式
func noPadding(src []byte) []byte {
	count := aes.BlockSize - len(src)%aes.BlockSize
	if len(src)%aes.BlockSize == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, count)...)
	}
}

// nopadding模式
func unNoPadding(src []byte) []byte {
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1]
		}
	}
	return nil
}

// padding模式
func padding(src []byte) []byte {
	count := aes.BlockSize - len(src)%aes.BlockSize
	padding := bytes.Repeat([]byte{byte(0)}, count)
	padding[count-1] = byte(count)
	return append(src, padding...)
}

// padding模式
func unPadding(src []byte) []byte {
	l := len(src)
	p := int(src[l-1])
	return src[:l-p]
}

package aes

import (
	"bytes"
	"crypto/aes"
	"fmt"
)

type Ecb struct {
	Key []byte
}

func (a *Ecb) Encrypt(src []byte, isPad ...bool) ([]byte, error) {
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		return nil, err
	}

	if len(src) == 0 {
		return nil, fmt.Errorf("content is empty")
	}

	if len(isPad) > 0 && isPad[0] == false {
		src = a.noPadding(src)
	} else {
		src = a.padding(src)
	}

	buf := make([]byte, aes.BlockSize)
	encrypted := make([]byte, 0)
	for i := 0; i < len(src); i += aes.BlockSize {
		block.Encrypt(buf, src[i:i+aes.BlockSize])
		encrypted = append(encrypted, buf...)
	}
	return encrypted, nil
}

func (a *Ecb) Decrypt(src []byte, isPad ...bool) ([]byte, error) {
	block, err := aes.NewCipher(a.Key)
	if err != nil {
		return nil, err
	}
	if len(src) == 0 {
		return nil, fmt.Errorf("content is empty")
	}
	buf := make([]byte, aes.BlockSize)
	decrypted := make([]byte, 0)
	for i := 0; i < len(src); i += aes.BlockSize {
		block.Decrypt(buf, src[i:i+aes.BlockSize])
		decrypted = append(decrypted, buf...)
	}

	if len(isPad) > 0 && isPad[0] == false {
		decrypted = a.unNoPadding(decrypted)
	} else {
		decrypted = a.unPadding(decrypted)
	}

	return decrypted, nil
}

//nopadding模式
func (a *Ecb) noPadding(src []byte) []byte {
	count := aes.BlockSize - len(src)%aes.BlockSize
	if len(src)%aes.BlockSize == 0 {
		return src
	} else {
		return append(src, bytes.Repeat([]byte{byte(0)}, count)...)
	}
}

//nopadding模式
func (a *Ecb) unNoPadding(src []byte) []byte {
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1]
		}
	}
	return nil
}

//padding模式
func (a *Ecb) padding(src []byte) []byte {
	count := aes.BlockSize - len(src)%aes.BlockSize
	padding := bytes.Repeat([]byte{byte(0)}, count)
	padding[count-1] = byte(count)
	return append(src, padding...)
}

//padding模式
func (a *Ecb) unPadding(src []byte) []byte {
	l := len(src)
	p := int(src[l-1])
	return src[:l-p]
}

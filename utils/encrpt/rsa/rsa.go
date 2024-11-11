package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
)

// Encrypt RSA加密
func Encrypt(plainText []byte, publicKey []byte) ([]byte, error) {
	//x509解码
	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	//类型断言
	key := publicKeyInterface.(*rsa.PublicKey)
	//对明文进行加密
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, key, plainText)
	if err != nil {
		return nil, err
	}
	//返回密文
	return cipherText, err
}

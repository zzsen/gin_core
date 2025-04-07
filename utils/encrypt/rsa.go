package encrypt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// 生成指定长度的RSA私钥
func RsaGeneratePrivateKey(bits int) (*rsa.PrivateKey, error) {
	// 使用加密安全的随机数生成器生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// 保存RSA私钥或公钥到PEM格式的文件
func savePem(key any, filePath string, isPrivateKey bool) error {
	var keyBytes []byte
	if isPrivateKey {
		// 按照PKCS#1标准对私钥进行ASN.1 DER编码
		keyBytes = x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey))
	} else {
		keyBytes = x509.MarshalPKCS1PublicKey(key.(*rsa.PublicKey))
	}
	keyType := "RSA PRIVATE KEY"
	if !isPrivateKey {
		keyType = "RSA PUBLIC KEY"
	}
	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}
	pemBytes := pem.EncodeToMemory(block)
	err := os.WriteFile(filePath, pemBytes, 0600)
	if err != nil {
		return fmt.Errorf("save %s failed: %v", filePath, err)
	}
	return nil
}

// 将RSA私钥转换为PEM格式的字节切片, 并保存到文件
func RsaSavePrivatePem(privateKey *rsa.PrivateKey, filePath string) error {
	return savePem(privateKey, filePath, true)
}

// 将RSA私钥转换为PEM格式的字节切片, 并保存到文件
func RsaSavePublicPem(publicKey *rsa.PublicKey, filePath string) error {
	return savePem(publicKey, filePath, false)
}

// 转换字符串为私钥
func convertStrToPrivateKey(privatekeyStr string) (*rsa.PrivateKey, error) {
	privatekeyBytes := []byte(privatekeyStr)
	block, _ := pem.Decode(privatekeyBytes)
	if block == nil {
		return nil, errors.New("get private key error")
	}
	priCs1, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priCs1, nil
	}
	pri2, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pri2.(*rsa.PrivateKey), nil
}

// 从文件中读取PEM格式的RSA私钥
func RsaReadPrivatePem(filePath string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read private key failed: %v", err)
	}
	return convertStrToPrivateKey(string(pemBytes))
}

// 转换字符串为公钥
func convertStrToPublicKey(publickeyStr string) (*rsa.PublicKey, error) {
	publickeyBytes := []byte(publickeyStr)
	// decode public key
	block, _ := pem.Decode(publickeyBytes)
	if block == nil {
		return nil, errors.New("get public key error")
	}
	// x509 parse public key
	pubCs1, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err == nil {
		return pubCs1, err
	}
	// x509 parse public key
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*rsa.PublicKey), err
}

// 从文件中读取PEM格式的RSA公钥
func RsaReadPublicPem(filePath string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read public key failed: %v", err)
	}
	return convertStrToPublicKey(string(pemBytes))
}

// 公钥加密，返回密文
func RsaEncrypt(publicKey *rsa.PublicKey, plainText string) ([]byte, error) {
	plainBytes := []byte(plainText)
	if publicKey == nil {
		return nil, errors.New(`rsa public key is empty`)
	}

	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, plainBytes)
}

// 公钥加密，返回base64编码的密文
func RsaEncrypt2Base64(publicKey *rsa.PublicKey, plainText string) (string, error) {
	encryptedData, err := RsaEncrypt(publicKey, plainText)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// 私钥解密
func RsaDecrypt(privateKey *rsa.PrivateKey, cipherBytes []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New(`rsa private key is empty`)
	}

	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, cipherBytes)
}

// 私钥解密
func RsaDecryptFromBase64(privateKey *rsa.PrivateKey, cipherText string) (string, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	decryptedData, err := RsaDecrypt(privateKey, cipherBytes)
	if err != nil {
		return "", err
	}
	return string(decryptedData), nil
}

// 使用RSA私钥加密数据（数字签名场景常用）
func RsaSign(privateKey *rsa.PrivateKey, plaintext string) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New(`rsa private key is empty`)
	}
	plainBytes := []byte(plaintext)
	// 1、选择hash算法，对需要签名的数据进行hash运算
	myhash := crypto.SHA256
	hashInstance := myhash.New()
	hashInstance.Write(plainBytes)
	hashed := hashInstance.Sum(nil)

	// 2、RSA数字签名（参数是随机数、私钥对象、哈希类型、签名文件的哈希串）
	return rsa.SignPKCS1v15(rand.Reader, privateKey, myhash, hashed)
}

// 使用RSA私钥加密数据（数字签名场景常用）
func RsaSign2Base64(privateKey *rsa.PrivateKey, plaintext string) (string, error) {
	bytes, err := RsaSign(privateKey, plaintext)
	if err != nil {
		return "", err
	}
	// 3、base64编码签名字符串
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// 使用RSA公钥解密数据（验证数字签名场景常用）
func RsaValidSign(publicKey *rsa.PublicKey, ciphertext string, signBytes []byte) error {
	cipherBytes := []byte(ciphertext)
	// 1、选择hash算法，对需要签名的数据进行hash运算
	myhash := crypto.SHA256
	hashInstance := myhash.New()
	hashInstance.Write(cipherBytes)
	hashed := hashInstance.Sum(nil)

	// 2、RSA验证数字签名（参数是公钥对象、哈希类型、签名文件的哈希串、签名后的字节）
	return rsa.VerifyPKCS1v15(publicKey, myhash, hashed, signBytes)
}

// 使用RSA公钥解密数据（验证数字签名场景常用）
func RsaValidSignFromBase64(publicKey *rsa.PublicKey, ciphertext string, sign string) error {
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}

	return RsaValidSign(publicKey, ciphertext, signBytes)
}

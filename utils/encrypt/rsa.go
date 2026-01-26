// Package encrypt 提供RSA非对称加密解密和数字签名功能
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

// RsaGeneratePrivateKey 生成指定长度的RSA私钥
// bits: 密钥长度，建议使用2048或4096位
// 返回: RSA私钥对象和错误信息
func RsaGeneratePrivateKey(bits int) (*rsa.PrivateKey, error) {
	// 使用加密安全的随机数生成器生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// savePem 保存RSA私钥或公钥到PEM格式的文件
// key: RSA私钥或公钥对象
// filePath: 保存路径
// isPrivateKey: 是否为私钥
// 返回: 错误信息
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
	// 创建PEM块
	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}
	// 编码为PEM格式
	pemBytes := pem.EncodeToMemory(block)
	// 写入文件，设置权限为600（仅所有者可读写）
	err := os.WriteFile(filePath, pemBytes, 0600)
	if err != nil {
		return fmt.Errorf("save %s failed: %w", filePath, err)
	}
	return nil
}

// RsaSavePrivatePem 将RSA私钥保存为PEM格式文件
// privateKey: RSA私钥对象
// filePath: 保存路径
// 返回: 错误信息
func RsaSavePrivatePem(privateKey *rsa.PrivateKey, filePath string) error {
	return savePem(privateKey, filePath, true)
}

// RsaSavePublicPem 将RSA公钥保存为PEM格式文件
// publicKey: RSA公钥对象
// filePath: 保存路径
// 返回: 错误信息
func RsaSavePublicPem(publicKey *rsa.PublicKey, filePath string) error {
	return savePem(publicKey, filePath, false)
}

// convertStrToPrivateKey 将PEM格式字符串转换为RSA私钥
// privatekeyStr: PEM格式的私钥字符串
// 返回: RSA私钥对象和错误信息
func convertStrToPrivateKey(privatekeyStr string) (*rsa.PrivateKey, error) {
	privatekeyBytes := []byte(privatekeyStr)
	// 解码PEM格式
	block, _ := pem.Decode(privatekeyBytes)
	if block == nil {
		return nil, errors.New("get private key error")
	}
	// 尝试PKCS#1格式解析
	priCs1, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priCs1, nil
	}
	// 尝试PKCS#8格式解析
	pri2, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pri2.(*rsa.PrivateKey), nil
}

// RsaReadPrivatePem 从文件中读取PEM格式的RSA私钥
// filePath: 私钥文件路径
// 返回: RSA私钥对象和错误信息
func RsaReadPrivatePem(filePath string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read private key failed: %w", err)
	}
	return convertStrToPrivateKey(string(pemBytes))
}

// convertStrToPublicKey 将PEM格式字符串转换为RSA公钥
// publickeyStr: PEM格式的公钥字符串
// 返回: RSA公钥对象和错误信息
func convertStrToPublicKey(publickeyStr string) (*rsa.PublicKey, error) {
	publickeyBytes := []byte(publickeyStr)
	// 解码PEM格式
	block, _ := pem.Decode(publickeyBytes)
	if block == nil {
		return nil, errors.New("get public key error")
	}
	// 尝试PKCS#1格式解析
	pubCs1, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err == nil {
		return pubCs1, err
	}
	// 尝试PKIX格式解析
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub.(*rsa.PublicKey), err
}

// RsaReadPublicPem 从文件中读取PEM格式的RSA公钥
// filePath: 公钥文件路径
// 返回: RSA公钥对象和错误信息
func RsaReadPublicPem(filePath string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read public key failed: %w", err)
	}
	return convertStrToPublicKey(string(pemBytes))
}

// RsaEncrypt 使用RSA公钥加密数据
// publicKey: RSA公钥对象
// plainText: 待加密的明文
// 返回: 加密后的字节数组和错误信息
func RsaEncrypt(publicKey *rsa.PublicKey, plainText string) ([]byte, error) {
	plainBytes := []byte(plainText)
	if publicKey == nil {
		return nil, errors.New(`rsa public key is empty`)
	}

	// 使用PKCS1v15填充模式进行加密
	return rsa.EncryptPKCS1v15(rand.Reader, publicKey, plainBytes)
}

// RsaEncrypt2Base64 使用RSA公钥加密数据并返回base64编码的密文
// publicKey: RSA公钥对象
// plainText: 待加密的明文
// 返回: base64编码的加密结果和错误信息
func RsaEncrypt2Base64(publicKey *rsa.PublicKey, plainText string) (string, error) {
	encryptedData, err := RsaEncrypt(publicKey, plainText)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// RsaDecrypt 使用RSA私钥解密数据
// privateKey: RSA私钥对象
// cipherBytes: 待解密的密文字节数组
// 返回: 解密后的明文字节数组和错误信息
func RsaDecrypt(privateKey *rsa.PrivateKey, cipherBytes []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New(`rsa private key is empty`)
	}

	// 使用PKCS1v15填充模式进行解密
	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, cipherBytes)
}

// RsaDecryptFromBase64 使用RSA私钥解密base64编码的密文
// privateKey: RSA私钥对象
// cipherText: base64编码的密文
// 返回: 解密后的明文和错误信息
func RsaDecryptFromBase64(privateKey *rsa.PrivateKey, cipherText string) (string, error) {
	// 解码base64密文
	cipherBytes, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	// 解密数据
	decryptedData, err := RsaDecrypt(privateKey, cipherBytes)
	if err != nil {
		return "", err
	}
	return string(decryptedData), nil
}

// RsaSign 使用RSA私钥对数据进行数字签名
// privateKey: RSA私钥对象
// plaintext: 待签名的明文
// 返回: 签名字节数组和错误信息
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

// RsaSign2Base64 使用RSA私钥对数据进行数字签名并返回base64编码
// privateKey: RSA私钥对象
// plaintext: 待签名的明文
// 返回: base64编码的签名和错误信息
func RsaSign2Base64(privateKey *rsa.PrivateKey, plaintext string) (string, error) {
	bytes, err := RsaSign(privateKey, plaintext)
	if err != nil {
		return "", err
	}
	// 3、base64编码签名字符串
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// RsaValidSign 使用RSA公钥验证数字签名
// publicKey: RSA公钥对象
// ciphertext: 原始明文
// signBytes: 签名字节数组
// 返回: 验证结果错误信息
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

// RsaValidSignFromBase64 使用RSA公钥验证base64编码的数字签名
// publicKey: RSA公钥对象
// ciphertext: 原始明文
// sign: base64编码的签名
// 返回: 验证结果错误信息
func RsaValidSignFromBase64(publicKey *rsa.PublicKey, ciphertext string, sign string) error {
	// 解码base64签名
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}

	return RsaValidSign(publicKey, ciphertext, signBytes)
}

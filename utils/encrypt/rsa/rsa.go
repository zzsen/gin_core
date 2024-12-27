package rsaUtil

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
func GenerateRsaPrivateKey(bits int) (*rsa.PrivateKey, error) {
	// 使用加密安全的随机数生成器生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// 保存RSA私钥或公钥到PEM格式的文件
func savePem(key interface{}, filePath string, isPrivateKey bool) error {
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
func SavePrivateKeyPem(privateKey *rsa.PrivateKey, filePath string) error {
	return savePem(privateKey, filePath, true)
}

// 将RSA私钥转换为PEM格式的字节切片, 并保存到文件
func SavePublicKeyPem(publicKey *rsa.PublicKey, filePath string) error {
	return savePem(publicKey, filePath, false)
}

// 从文件中读取PEM格式的RSA私钥
func ReadPrivateKeyPem(filePath string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read private key failed: %v", err)
	}
	return convertStrToPrivateKey(string(pemBytes))
}

// 从文件中读取PEM格式的RSA公钥
func ReadPublicKeyPem(filePath string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read public key failed: %v", err)
	}
	return convertStrToPublicKey(string(pemBytes))
}

type RSAUtil struct {
	//公钥字符串
	publicKeyStr string
	//私钥字符串
	privateKeyStr string
	//公钥
	PublicKey *rsa.PublicKey
	//私钥
	PrivateKey *rsa.PrivateKey
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

// 设置公钥
func (r *RSAUtil) SetPublicKey(publicKeyStr string) (err error) {
	r.publicKeyStr = publicKeyStr
	r.PublicKey, err = convertStrToPublicKey(r.publicKeyStr)
	return err
}

// 设置私钥
func (r *RSAUtil) SetPrivateKey(privateKeyStr string) (err error) {
	r.privateKeyStr = privateKeyStr
	r.PrivateKey, err = convertStrToPrivateKey(r.privateKeyStr)
	return err
}

// 公钥加密
func (r *RSAUtil) EncryptWithPrivateKey(plainTest string) (string, error) {
	plainBytes := []byte(plainTest)
	if r.PublicKey == nil {
		return "", errors.New(`rsa public key is empty`)
	}

	var encryptedData []byte
	encryptedData, err := rsa.EncryptPKCS1v15(rand.Reader, r.PublicKey, plainBytes)
	if err != nil {
		return "", err
	}
	// // 计算每次加密的最大数据长度，考虑PKCS#1 v1.5填充
	// maxLen := rsas.PublicKey.Size() - 11
	// var encryptedData []byte
	// // 对明文进行分块加密，以适应RSA算法对加密数据长度的限制
	// for start := 0; start < len(input); start += maxLen {
	// 	end := start + maxLen
	// 	if end > len(input) {
	// 		end = len(input)
	// 	}
	// 	block := input[start:end]
	// 	encryptedBlock, err := rsa.EncryptPKCS1v15(rand.Reader, rsas.PublicKey, block)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	encryptedData = append(encryptedData, encryptedBlock...)
	// }
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

// 私钥解密
func (r *RSAUtil) DecryptWithPrivateKey(crypeText string) (string, error) {
	if r.PrivateKey == nil {
		return "", errors.New(`rsa private key is empty`)
	}
	input, err := base64.StdEncoding.DecodeString(crypeText)
	if err != nil {
		return "", err
	}

	decryptedData, err := rsa.DecryptPKCS1v15(rand.Reader, r.PrivateKey, input)
	if err != nil {
		return "", err
	}
	// // 计算每次解密的最大数据长度（基于私钥长度）
	// maxLen := rsas.PrivateKey.Size()
	// var decryptedData []byte
	// // 对密文进行分块解密
	// for start := 0; start < len(input); start += maxLen {
	// 	end := start + maxLen
	// 	if end > len(input) {
	// 		end = len(input)
	// 	}
	// 	block := input[start:end]
	// 	decryptedBlock, err := rsa.DecryptPKCS1v15(rand.Reader, rsas.PrivateKey, block)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	decryptedData = append(decryptedData, decryptedBlock...)
	// }
	return string(decryptedData), nil
}

// 使用RSA私钥加密数据（数字签名场景常用）
func (r *RSAUtil) SignWithPrivateKey(plaintext string) (string, error) {
	if r.PrivateKey == nil {
		return "", errors.New(`rsa private key is empty`)
	}
	plainBytes := []byte(plaintext)
	// 1、选择hash算法，对需要签名的数据进行hash运算
	myhash := crypto.SHA256
	hashInstance := myhash.New()
	hashInstance.Write(plainBytes)
	hashed := hashInstance.Sum(nil)

	// 2、RSA数字签名（参数是随机数、私钥对象、哈希类型、签名文件的哈希串）
	bytes, err := rsa.SignPKCS1v15(rand.Reader, r.PrivateKey, myhash, hashed)
	if err != nil {
		return "", err
	}

	// 3、base64编码签名字符串
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// 使用RSA公钥解密数据（验证数字签名场景常用）
func (r *RSAUtil) ValidSignWithPublicKey(ciphertext string, sign string) error {
	cipherBytes := []byte(ciphertext)
	// 1、选择hash算法，对需要签名的数据进行hash运算
	myhash := crypto.SHA256
	hashInstance := myhash.New()
	hashInstance.Write(cipherBytes)
	hashed := hashInstance.Sum(nil)
	// 2、base64解码签名字符串
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}
	// 3、RSA验证数字签名（参数是公钥对象、哈希类型、签名文件的哈希串、签名后的字节）
	return rsa.VerifyPKCS1v15(r.PublicKey, myhash, hashed, signBytes)
}

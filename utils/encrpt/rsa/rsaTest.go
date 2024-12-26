package rsaUtil

import (
	"errors"
	"fmt"
)

// 公钥加密私钥解密
func encryptAndDecrypt(r RSAUtil) error {
	content := "hello world"
	pubEnctypt, err := r.EncryptWithPrivateKey(content)
	if err != nil {
		fmt.Printf("\033[31mrsa encrypt failed: %v\n\033[0m", err)
		return err
	}
	fmt.Printf("rsa encrypt result: %s\n", pubEnctypt)

	pridecrypt, err := r.DecryptWithPrivateKey(pubEnctypt)
	if err != nil {
		return err
	}
	if string(pridecrypt) != content {
		return errors.New(`rsa encrypt failed, content is not equal`)
	}
	fmt.Println("\033[32mrsa encrypt successed\033[0m")
	return nil
}

func signAndValidSign(r RSAUtil) error {
	content := "hello world"
	sign, err := r.SignWithPrivateKey(content)
	if err != nil {
		fmt.Printf("\033[31mrsa sign failed: %v\n\033[0m", err)
		return err
	}
	fmt.Printf("rsa sign result: %s\n", sign)

	err = r.ValidSignWithPublicKey(content, sign)
	if err != nil {
		fmt.Printf("\033[31mrsa valid sign failed: %v\n\033[0m", err)
		return err
	}
	fmt.Println("\033[32mrsa valid sign successed\033[0m")
	return nil
}

func Test() {
	// 生成2048位的RSA私钥
	privateKey, err := GenerateRsaPrivateKey(2048)
	if err != nil {
		fmt.Printf("\033[31mgenerate rsa key error: %v\n\033[0m", err)
		return
	}

	// 保存到文件
	privateKeyPem := "private_key.pem"
	publicKeyPem := "public_key.pem"
	SavePrivateKeyPem(privateKey, privateKeyPem)
	SavePublicKeyPem(&privateKey.PublicKey, publicKeyPem)

	// 从文件中读取
	privateKey, err = ReadPrivateKeyPem(privateKeyPem)
	if err != nil {
		// 带颜色的输出
		fmt.Printf("\033[31mread private key error: %v\n\033[0m", err)
		return
	}
	publicKey, err := ReadPublicKeyPem(publicKeyPem)
	if err != nil {
		fmt.Printf("\033[31mread public key error: %v\n\033[0m", err)
		return
	}

	r := RSAUtil{}
	r.PrivateKey = privateKey
	r.PublicKey = publicKey

	// 公钥加密私钥解密
	if err := encryptAndDecrypt(r); err != nil {
		fmt.Println(err)
	}

	// 签名和验证签名
	if err := signAndValidSign(r); err != nil {
		fmt.Println(err)
	}
}

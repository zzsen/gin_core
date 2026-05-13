# 加密工具 (Encrypt)

## 一、概述

`utils/encrypt` 包提供了常用的加密解密功能，包括 AES 对称加密和 RSA 非对称加密。

支持的算法：

| 算法 | 文件 | 说明 |
|------|------|------|
| AES ECB | `aes_ecb.go` | AES 对称加密（ECB 模式），支持 PKCS7 和零填充 |
| RSA | `rsa.go` | RSA 非对称加密、解密、数字签名和签名验证 |

## 二、AES ECB 加解密

### 2.1 加密

调用链：[AesEcbEncrypt](../utils/encrypt/aes_ecb.go) → `padding` / `noPadding` → `aes.NewCipher` → 分块加密 → Base64 编码

```go
import "github.com/zzsen/gin_core/utils/encrypt"

// 默认 PKCS7 填充
cipherText, err := encrypt.AesEcbEncrypt("hello world", "1234567890123456")

// 指定零填充（不使用 PKCS7）
cipherText, err := encrypt.AesEcbEncrypt("hello world", "1234567890123456", false)
```

**参数说明：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `plainText` | `string` | 待加密的明文 |
| `key` | `string` | 加密密钥，长度必须为 16、24 或 32 字节 |
| `isPad` | `...bool` | 可选，是否使用 PKCS7 填充，默认 `true` |

**返回值：** Base64 编码的密文字符串。

### 2.2 解密

调用链：[AesEcbDecrypt](../utils/encrypt/aes_ecb.go) → Base64 解码 → `aes.NewCipher` → 分块解密 → `unPadding` / `unNoPadding`

```go
plainText, err := encrypt.AesEcbDecrypt(cipherText, "1234567890123456")
```

### 2.3 填充模式

| 模式 | 函数 | 说明 |
|------|------|------|
| PKCS7 | `padding` / `unPadding` | 标准 PKCS7 填充，填充值为填充字节数 |
| 零填充 | `noPadding` / `unNoPadding` | 不足块大小的部分用 `0x00` 填充 |

## 三、RSA 加解密

### 3.1 密钥生成与管理

调用链：[RsaGeneratePrivateKey](../utils/encrypt/rsa.go) → `rsa.GenerateKey`

```go
import "github.com/zzsen/gin_core/utils/encrypt"

// 生成 2048 位 RSA 密钥对
privateKey, err := encrypt.RsaGeneratePrivateKey(2048)
publicKey := &privateKey.PublicKey

// 保存密钥到 PEM 文件
err = encrypt.RsaSavePrivatePem(privateKey, "private.pem")
err = encrypt.RsaSavePublicPem(publicKey, "public.pem")

// 从 PEM 文件读取密钥
privateKey, err = encrypt.RsaReadPrivatePem("private.pem")
publicKey, err = encrypt.RsaReadPublicPem("public.pem")
```

**密钥管理函数一览：**

| 函数 | 说明 |
|------|------|
| [RsaGeneratePrivateKey](../utils/encrypt/rsa.go) | 生成指定位数的 RSA 私钥 |
| [RsaSavePrivatePem](../utils/encrypt/rsa.go) | 保存私钥为 PEM 文件（PKCS#1） |
| [RsaSavePublicPem](../utils/encrypt/rsa.go) | 保存公钥为 PEM 文件（PKCS#1） |
| [RsaReadPrivatePem](../utils/encrypt/rsa.go) | 从 PEM 文件读取私钥（兼容 PKCS#1 和 PKCS#8） |
| [RsaReadPublicPem](../utils/encrypt/rsa.go) | 从 PEM 文件读取公钥（兼容 PKCS#1 和 PKIX） |

### 3.2 加密与解密

调用链：[RsaEncrypt2Base64](../utils/encrypt/rsa.go) → [RsaEncrypt](../utils/encrypt/rsa.go) → `rsa.EncryptPKCS1v15`

```go
// 加密（返回 Base64 密文）
cipherText, err := encrypt.RsaEncrypt2Base64(publicKey, "sensitive data")

// 解密
plainText, err := encrypt.RsaDecryptFromBase64(privateKey, cipherText)
```

| 函数 | 说明 |
|------|------|
| [RsaEncrypt](../utils/encrypt/rsa.go) | 公钥加密，返回 `[]byte` |
| [RsaEncrypt2Base64](../utils/encrypt/rsa.go) | 公钥加密，返回 Base64 字符串 |
| [RsaDecrypt](../utils/encrypt/rsa.go) | 私钥解密，接收 `[]byte` |
| [RsaDecryptFromBase64](../utils/encrypt/rsa.go) | 私钥解密，接收 Base64 字符串 |

### 3.3 数字签名

调用链：[RsaSign2Base64](../utils/encrypt/rsa.go) → [RsaSign](../utils/encrypt/rsa.go) → SHA256 哈希 → `rsa.SignPKCS1v15`

```go
// 签名
signature, err := encrypt.RsaSign2Base64(privateKey, "data to sign")

// 验签
err = encrypt.RsaValidSignFromBase64(publicKey, "data to sign", signature)
if err != nil {
    // 签名验证失败
}
```

| 函数 | 说明 |
|------|------|
| [RsaSign](../utils/encrypt/rsa.go) | 私钥签名，返回 `[]byte` |
| [RsaSign2Base64](../utils/encrypt/rsa.go) | 私钥签名，返回 Base64 字符串 |
| [RsaValidSign](../utils/encrypt/rsa.go) | 公钥验签，接收 `[]byte` |
| [RsaValidSignFromBase64](../utils/encrypt/rsa.go) | 公钥验签，接收 Base64 字符串 |

## 四、注意事项

1. **AES 密钥长度**：必须为 16（AES-128）、24（AES-192）或 32（AES-256）字节
2. **ECB 模式安全性**：ECB 模式不推荐用于加密大量数据，相同的明文块会产生相同的密文块；适用于加密短数据（如配置字段）
3. **RSA 密钥长度**：建议使用 2048 位以上
4. **密钥文件权限**：`RsaSavePrivatePem` 生成的文件权限为 `0600`（仅所有者可读写），请确保密钥文件安全存储

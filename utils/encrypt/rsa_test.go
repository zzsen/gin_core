// Package encrypt RSA非对称加密解密和数字签名功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 RSA 非对称加密解密和数字签名功能的单元测试。
//
// 测试覆盖内容：
// 1. 密钥生成 - 生成指定位数的RSA私钥
// 2. 密钥保存 - 保存私钥/公钥到PEM文件
// 3. 密钥读取 - 从PEM文件读取私钥/公钥
// 4. 数字签名 - 使用私钥签名数据
// 5. 签名验证 - 使用公钥验证签名
// 6. 加密解密 - 公钥加密、私钥解密
// 7. 完整流程 - 签名验证、加解密的完整循环
//
// 密钥长度支持：1024、2048、4096 位
// 签名算法：SHA256withRSA
// 填充方式：PKCS1v15
//
// 运行测试：go test -v ./utils/encrypt/... -run RSA
// ==================================================
package encrypt

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== RSA 密钥生成测试 ====================

// TestRsaGeneratePrivateKey 测试RSA私钥生成功能
//
// 【功能点】验证 RSA 私钥的正确生成
// 【测试流程】
//  1. 调用 RsaGeneratePrivateKey 生成 2048 位私钥
//  2. 验证不返回错误
func TestRsaGeneratePrivateKey(t *testing.T) {
	t.Run("rsa generate private key", func(t *testing.T) {
		// 生成2048位RSA私钥
		_, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)
	})
}

// TestRsaSavePrivatePem 测试RSA私钥保存为PEM格式文件功能
//
// 【功能点】验证 RSA 私钥保存到 PEM 文件
// 【测试流程】
//  1. 生成 2048 位 RSA 私钥
//  2. 调用 RsaSavePrivatePem 保存到文件
//  3. 验证文件存在
//  4. 清理测试文件
func TestRsaSavePrivatePem(t *testing.T) {
	t.Run("rsa save private pem", func(t *testing.T) {
		// 生成带时间戳的文件名，避免测试冲突
		fileName := fmt.Sprintf("private_key_%s.pem", time.Now().Format("20060102150405"))

		// 生成2048位RSA私钥
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		// 保存私钥到PEM文件
		err = RsaSavePrivatePem(privateKey, fileName)
		assert.Nil(t, err)

		// 验证文件是否存在
		exists := assert.FileExists(t, fileName)
		if exists {
			// 清理测试文件
			os.Remove(fileName)
		}
	})
}

// TestRsaSavePublicPem 测试RSA公钥保存为PEM格式文件功能
//
// 【功能点】验证 RSA 公钥保存到 PEM 文件
// 【测试流程】
//  1. 生成 2048 位 RSA 私钥
//  2. 从私钥提取公钥并保存到文件
//  3. 验证文件存在
//  4. 清理测试文件
func TestRsaSavePublicPem(t *testing.T) {
	t.Run("rsa save public pem", func(t *testing.T) {
		// 生成带时间戳的文件名，避免测试冲突
		fileName := fmt.Sprintf("public_key_%s.pem", time.Now().Format("20060102150405"))

		// 生成2048位RSA私钥
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		// 保存公钥到PEM文件
		err = RsaSavePublicPem(&privateKey.PublicKey, fileName)
		assert.Nil(t, err)

		// 验证文件是否存在
		exists := assert.FileExists(t, fileName)
		if exists {
			// 清理测试文件
			os.Remove(fileName)
		}
	})
}

// TestRsaReadPrivatePem 测试从PEM文件读取RSA私钥功能
//
// 【功能点】验证从 PEM 文件读取 RSA 私钥
// 【测试流程】
//  1. 生成私钥并保存到 PEM 文件
//  2. 调用 RsaReadPrivatePem 读取私钥
//  3. 验证读取成功
//  4. 清理测试文件
func TestRsaReadPrivatePem(t *testing.T) {
	t.Run("rsa read private pem", func(t *testing.T) {
		// 生成带时间戳的文件名，避免测试冲突
		fileName := fmt.Sprintf("private_key_%s.pem", time.Now().Format("20060102150405"))

		// 生成2048位RSA私钥
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		// 保存私钥到PEM文件
		err = RsaSavePrivatePem(privateKey, fileName)
		assert.Nil(t, err)

		// 验证文件是否存在
		exists := assert.FileExists(t, fileName)
		if exists {
			// 从PEM文件读取私钥
			_, err = RsaReadPrivatePem(fileName)
			assert.Nil(t, err)
			// 清理测试文件
			os.Remove(fileName)
		}
	})
}

// TestRsaReadPublicPem 测试从PEM文件读取RSA公钥功能
//
// 【功能点】验证从 PEM 文件读取 RSA 公钥
// 【测试流程】
//  1. 生成公钥并保存到 PEM 文件
//  2. 调用 RsaReadPublicPem 读取公钥
//  3. 验证读取成功
//  4. 清理测试文件
func TestRsaReadPublicPem(t *testing.T) {
	t.Run("rsa read public pem", func(t *testing.T) {
		// 生成带时间戳的文件名，避免测试冲突
		fileName := fmt.Sprintf("public_key_%s.pem", time.Now().Format("20060102150405"))

		// 生成2048位RSA私钥
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		// 保存公钥到PEM文件
		err = RsaSavePublicPem(&privateKey.PublicKey, fileName)
		assert.Nil(t, err)

		// 验证文件是否存在
		exists := assert.FileExists(t, fileName)
		if exists {
			// 从PEM文件读取公钥
			_, err = RsaReadPublicPem(fileName)
			assert.Nil(t, err)
			// 清理测试文件
			os.Remove(fileName)
		}
	})
}

// 测试用的RSA私钥（PEM格式）
const privateKeyStr = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA+GKTixVsQR79E1BAs/IaxJgFS0eELrJvAWkeQ+tdEabYGpWX
YoiueiT8gpyU2TkLQGDvR+Zr7/TRUtCYwzF4zsLDRW9gokCKsIJDqflFP2q6zPX2
yuwfLUE8g0U3NIpqD34kQJguyt2Fn5IsaWE2tVFXczyyEHETugCbthK/oNfgn3TH
DYqwTMGwPtQz5ifi8VncwdaWJdnbbDPmKi1iSeNVHTZv4rKdv7oPDep7eaA2qR4F
LMyCQyPtfDImBypLbnLjDdaBSw8hIGEe9VbHHrCWwta34bBYnSi86Dm7LsyAsjOh
zR2ROsXiaTs5/GXBnIdCuKVawLjegKS6Xhd5EwIDAQABAoIBAHQWbmrkulG9T/7E
1VjE4Kndeyvx4t+IWcVJAfIwgLENT5ctLzHIO/Ouca4BzLexp4aRR4RNN0lRHLwd
7ifcaWAJOwaqxXlPvQI9/63jaO/4zBGbK5svvGqEQOoBYYnW3zcad4sRFV2PJzKr
OMKPwuf/emXLilWQ4+1c92mjXZioIQejSyy5s3N3z9dRVn2GcXkeMvSIH+MZCpWJ
LfYSWl5Te5rpIgEJOmAo685U5DMz06dKURXh/pSf5gGFDosqyINbTJAYOtLolxHp
onVBR67lHeMkuWTDCt2d9kWd1wM5qaaT1uhN1Zb/1btQRz52QRsVvLnrDpIMTV6I
zILJtXECgYEA/BcO9fz8FxRawClPVk60G3jZ8SujRylFYR8+1ey2zBNw8AgBqqmK
C64OJepDKwWJff6DTYL8u5VIovH5jH2q5DLAUI5QLC7GZAzz8CnNbc+dEL7vUrRX
kZK5Nh5ugZgo77dau94SXdWk8eipW/FnMkTaJSzc/S8p3pusZi2uBg0CgYEA/DzO
kRgBP+Yd/x5imaKk2RNwjtedzOU0KjYPD3o40Z1IX1AJ/MjehnTvUz1bPKlPpc1F
2Vf6w8yCU0U1xjdUR5KWRfqg+DIJPG2Z+kRgRUx+0IxQqnN3GlAEFE5udvFQzMUL
O7ebxnPGwKLA0SHdlWYpYYxRw5MsNwyNH2vZ058CgYEAgqms/oGHZKsPMsT2s2SN
5CNqy59zvSG+LU4VsqpEQVjeU/vCaWQBAnbQLITVFcqD7oNqKVX4i34gLR1A3LoS
Rr+rgNWS5qPD/v3bvqLcMMvIvHJK99I0BWdIiq2RV6i3pzChXfkICgz/tseCaP6i
H6MictxjGvREPnbwD/IjXk0CgYAh1apDzjuErcKCUToar7V7JN9pWcTiEjDAJMY6
ZkOu4nEtz9e3H96xnIfp24Ycif2UGQfwkpuhnhIxR0xiTVOx0hj0RB1JjbStdWo9
JuTfBtbP9LJxWtG0Jt2VN7wbml0jSp8qIIP1x9v2RR6mLuvBOZX9bswc9uXscHOR
rm7mswKBgQCFQTXNmnVVCSOEnOKG1YihOjJ0DcwQqtTVDdV28CLoYvK5o0ryvUvd
OSyOnAxET/AlMZyQCQERREPkKRRjirzYovVby63E+CnDpJtR3D8KQr2hrpklQkg0
iFVA1o+tKnvG8GokwoYBWeS6DWySRjeYyt9A4K5Co892J7RHJZLiQg==
-----END RSA PRIVATE KEY-----
`

// 测试用的RSA公钥（PEM格式）
const publicKeyStr = `-----BEGIN RSA PUBLIC KEY-----
MIIBCgKCAQEA+GKTixVsQR79E1BAs/IaxJgFS0eELrJvAWkeQ+tdEabYGpWXYoiu
eiT8gpyU2TkLQGDvR+Zr7/TRUtCYwzF4zsLDRW9gokCKsIJDqflFP2q6zPX2yuwf
LUE8g0U3NIpqD34kQJguyt2Fn5IsaWE2tVFXczyyEHETugCbthK/oNfgn3THDYqw
TMGwPtQz5ifi8VncwdaWJdnbbDPmKi1iSeNVHTZv4rKdv7oPDep7eaA2qR4FLMyC
QyPtfDImBypLbnLjDdaBSw8hIGEe9VbHHrCWwta34bBYnSi86Dm7LsyAsjOhzR2R
OsXiaTs5/GXBnIdCuKVawLjegKS6Xhd5EwIDAQAB
-----END RSA PUBLIC KEY-----
`

// 测试用的明文
const plainText = "hello world"

// 测试用的密文（base64编码）
const cipherText = "qu8QSBqJiYuJ84z3cDLJ4VfOUeaVw2XjRp4jkrbguAy7X/xKh+o1MWaQmiGlcOQyoRsD5oab+JJ4e3Tw/eYry5OWB1H8N2wxpRYaoZu66zNB65huAoktZTE+RqMaBSvDWYRwCOuV8X2w36zFOjLJTQgz45eWr78oRxOD+8xO30jiR/rCecIHq7EQha8E1QXApoggCatUkj9gRs4TdkIAqPlxUItWp+348IZTJh8U7zy2tf2vy8x3VX7lQ7sqd5Jpm2akIxRcSU5MNJS9gO8o4l0qUpv7BT3DzIgYdqJy3NV/Df7NH270KfuBr2uLVXApKlZBhgNEhCoAozZ9LRDzJw=="

// 测试用的签名（base64编码）
const signText = "t5shPK2KELo8cB49Gtz8IMLkgBi903sVNBFVPGYRaoFXkk4dlx0zbi3qsJ6LDd4qJSwBIx8FpLu1u2zJXFk/iNIhQmG7KSMX5WlzJwgbl956jD2VP8b0lSqLZzO2CPL5WV0JbEcXe08EIlQwdPQRs10tyj7WhmQXmxBxT4CkD0q+1vRLHYAE9yU7K51TezjHdbIqrZC3MpkW8m/ptr7aO4AC/q8x5puqeVEC4OMJKZGxyhXiS+OYkT5cCIpCTo5oHnMZNutfZFM3bmvR64054h6DJk9b2ZJ0KbUFp9HfjzmEDsL74iloTSCe7DBqMik8cSlSf/+DOCShSliXwlJe2Q=="

// ==================== RSA 加密解密测试 ====================

// TestRsaEncrypt 测试RSA公钥加密功能
//
// 【功能点】验证使用公钥加密明文
// 【测试流程】
//  1. 从 PEM 字符串解析公钥
//  2. 使用公钥加密明文
//  3. 验证返回非空密文
func TestRsaEncrypt(t *testing.T) {
	t.Run("rsa encrypt", func(t *testing.T) {
		// 从PEM字符串解析公钥
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		// 使用公钥加密明文
		encryptResult, err := RsaEncrypt(publicKey, plainText)
		assert.Nil(t, err)
		assert.NotNil(t, encryptResult)
	})
}

// TestRsaEncrypt2Base64 测试RSA公钥加密并返回base64编码功能
//
// 【功能点】验证公钥加密并返回 Base64 编码结果
// 【测试流程】
//  1. 从 PEM 字符串解析公钥
//  2. 调用 RsaEncrypt2Base64 加密明文
//  3. 验证返回非空 Base64 字符串
func TestRsaEncrypt2Base64(t *testing.T) {
	t.Run("rsa encrypt", func(t *testing.T) {
		// 从PEM字符串解析公钥
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		// 使用公钥加密明文并返回base64编码
		result, err := RsaEncrypt2Base64(publicKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, result, "")
	})
}

// TestRsaDecrypt 测试RSA私钥解密功能
//
// 【功能点】验证使用私钥解密密文
// 【测试流程】
//  1. 从 PEM 字符串解析私钥
//  2. 解码 Base64 密文
//  3. 使用私钥解密
//  4. 验证解密成功
func TestRsaDecrypt(t *testing.T) {
	t.Run("rsa decrypt", func(t *testing.T) {
		// 从PEM字符串解析私钥
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		// 解码base64密文
		cipherBytes, err := base64.StdEncoding.DecodeString(cipherText)
		assert.Nil(t, err)

		// 使用私钥解密密文
		result, err := RsaDecrypt(privateKey, cipherBytes)
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})
}

// TestRsaDecryptFromBase64 测试RSA私钥解密base64编码密文功能
//
// 【功能点】验证私钥解密 Base64 编码的密文
// 【测试流程】
//  1. 从 PEM 字符串解析私钥
//  2. 调用 RsaDecryptFromBase64 解密
//  3. 验证解密结果与原始明文一致
func TestRsaDecryptFromBase64(t *testing.T) {
	t.Run("rsa decrypt", func(t *testing.T) {
		// 从PEM字符串解析私钥
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		// 使用私钥解密base64编码的密文
		result, err := RsaDecryptFromBase64(privateKey, cipherText)
		assert.Nil(t, err)
		assert.Equal(t, result, plainText)
	})
}

// TestRsaCrypt 测试RSA加密解密的完整流程
//
// 【功能点】验证加密→解密的完整循环
// 【测试流程】
//  1. 解析公钥和私钥
//  2. 使用公钥加密明文
//  3. 使用私钥解密密文
//  4. 验证解密结果与原始明文一致
func TestRsaCrypt(t *testing.T) {
	t.Run("rsa crypt", func(t *testing.T) {
		// 从PEM字符串解析公钥
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		// 从PEM字符串解析私钥
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		// 使用公钥加密明文
		encryptResult, err := RsaEncrypt2Base64(publicKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, encryptResult, "")

		// 使用私钥解密密文
		result, err := RsaDecryptFromBase64(privateKey, encryptResult)
		assert.Nil(t, err)
		assert.Equal(t, result, plainText)
	})
}

// ==================== RSA 数字签名测试 ====================

// TestRsaSign 测试RSA私钥数字签名功能
//
// 【功能点】验证使用私钥对数据进行签名
// 【测试流程】
//  1. 从 PEM 字符串解析私钥
//  2. 使用私钥对明文签名
//  3. 验证返回非空签名
func TestRsaSign(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		// 从PEM字符串解析私钥
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		// 使用私钥对明文进行数字签名
		result, err := RsaSign(privateKey, plainText)
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})
}

// TestRsaSign2Base64 测试RSA私钥数字签名并返回base64编码功能
//
// 【功能点】验证私钥签名并返回 Base64 编码结果
// 【测试流程】
//  1. 从 PEM 字符串解析私钥
//  2. 调用 RsaSign2Base64 签名
//  3. 验证返回非空 Base64 字符串
func TestRsaSign2Base64(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		// 从PEM字符串解析私钥
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		// 使用私钥对明文进行数字签名并返回base64编码
		result, err := RsaSign2Base64(privateKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, result, "")
	})
}

// TestRsaValidSign 测试RSA公钥验证数字签名功能
//
// 【功能点】验证使用公钥验证签名的正确性
// 【测试流程】
//  1. 从 PEM 字符串解析公钥
//  2. 解码 Base64 签名
//  3. 使用公钥验证签名
//  4. 验证签名有效（无错误返回）
func TestRsaValidSign(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		// 从PEM字符串解析公钥
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		// 解码base64签名
		cipherBytes, err := base64.StdEncoding.DecodeString(signText)
		assert.Nil(t, err)

		// 使用公钥验证数字签名
		err = RsaValidSign(publicKey, plainText, cipherBytes)
		assert.Nil(t, err)
	})
}

// TestRsaValidSignFromBase64 测试RSA公钥验证base64编码数字签名功能
//
// 【功能点】验证公钥验证 Base64 编码的签名
// 【测试流程】
//  1. 从 PEM 字符串解析公钥
//  2. 调用 RsaValidSignFromBase64 验证签名
//  3. 验证签名有效（无错误返回）
func TestRsaValidSignFromBase64(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		// 从PEM字符串解析公钥
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		// 使用公钥验证base64编码的数字签名
		err = RsaValidSignFromBase64(publicKey, plainText, signText)
		assert.Nil(t, err)
	})
}

// TestRsaSignAndValid 测试RSA数字签名和验证的完整流程
//
// 【功能点】验证签名→验签的完整循环
// 【测试流程】
//  1. 解析公钥和私钥
//  2. 使用私钥签名明文
//  3. 使用公钥验证签名
//  4. 验证签名有效（无错误返回）
func TestRsaSignAndValid(t *testing.T) {
	t.Run("rsa sign and valid", func(t *testing.T) {
		// 从PEM字符串解析公钥
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		// 从PEM字符串解析私钥
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		// 使用私钥对明文进行数字签名
		result, err := RsaSign2Base64(privateKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, result, "")

		// 使用公钥验证数字签名
		err = RsaValidSignFromBase64(publicKey, plainText, result)
		assert.Nil(t, err)
	})
}

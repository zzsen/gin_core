// Package encrypt AES/RSA 扩展边界与异常路径测试
//
// ==================== 测试说明 ====================
// 本文件补充 AES ECB 和 RSA 工具函数的边界与异常场景，用于提升覆盖率。
// 在临时目录下进行 RSA 密钥文件读写测试。
//
// 测试覆盖内容：
// 1. AesEcbEncrypt 扩展（非标准密钥长度错误）
// 2. AesEcbDecrypt 扩展（密文长度非对齐错误）
// 3. AES ECB NoPadding 模式完整流程
// 4. UnPadding 边界用例（PKCS7 不一致、空切片、全零尾部）
// 5. RSA ReadPem 文件错误（无效 PEM、空文件、缺失文件）
// 6. ConvertStrToPrivateKey / ConvertStrToPublicKey 多格式解析与失败
// 7. RSA SavePem 写入失败
// 8. RSA Encrypt/Decrypt nil 输入与无效输入
// 9. RSA Sign/Verify 失败场景
// 10. RSA GeneratePrivateKey 非法位数
// 11. RSA ReadWritePem 完整往返测试
// 12. RSA 明文过长加密失败
//
// 运行测试：go test -v ./utils/encrypt/... -run "Extended|Variants|Failure|EdgeCase|RoundTrip"
// ==================================================
package encrypt

import (
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const aesKey16 = "UTabIUiHgDyh464+" // 16 字节 AES-128 密钥

// TestAesEcbEncryptExtended AES ECB 加密边界与异常
//
// 【功能点】空明文、非法密钥长度导致的加密失败路径
// 【测试流程】
//  1. 明文为空字符串时调用 AesEcbEncrypt，期望返回错误
//  2. 密钥长度不符合 AES 要求时调用 AesEcbEncrypt，期望 aes.NewCipher 报错
func TestAesEcbEncryptExtended(t *testing.T) {
	t.Run("empty_plaintext", func(t *testing.T) {
		got, err := AesEcbEncrypt("", aesKey16)
		assert.Empty(t, got)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("invalid_key_length", func(t *testing.T) {
		got, err := AesEcbEncrypt("hello", "short")
		assert.Empty(t, got)
		assert.Error(t, err)
	})
}

// TestAesEcbDecryptExtended AES ECB 解密密钥错误与填充异常
//
// 【功能点】非法密钥长度；错误密钥解密后 PKCS7 校验失败时明文不可用（仍为合法解密调用）
// 【测试流程】
//  1. 使用非法长度密钥解密，期望报错
//  2. 使用合法密文但错误密钥解密，期望无 Cipher 错误（解密结果与原文不一致）
func TestAesEcbDecryptExtended(t *testing.T) {
	cipherB64, err := AesEcbEncrypt("payload", aesKey16)
	require.NoError(t, err)

	t.Run("invalid_key_length", func(t *testing.T) {
		got, err := AesEcbDecrypt(cipherB64, "bad")
		assert.Empty(t, got)
		assert.Error(t, err)
	})

	wrongKey := "AAAAAAAAAAAAAAAA"
	require.Len(t, wrongKey, 16)

	t.Run("wrong_key_decrypt_succeeds_cipher_but_plaintext_differs", func(t *testing.T) {
		got, err := AesEcbDecrypt(cipherB64, wrongKey)
		require.NoError(t, err)
		assert.NotEqual(t, "payload", got)
	})
}

// TestAesEcbNoPaddingRoundTrip AES ECB 关闭 PKCS7 填充时的往返
//
// 【功能点】isPad=false 时走 noPadding / unNoPadding 分支，覆盖非整块与整块明文
// 【测试流程】
//  1. 对长度非 16 倍数的明文使用 isPad=false 加密再解密，验证还原一致
//  2. 对长度恰为 16 倍数的明文使用 isPad=false 加密再解密，验证 noPadding 整块分支与还原一致
func TestAesEcbNoPaddingRoundTrip(t *testing.T) {
	for _, plain := range []string{"abc", strings.Repeat("x", 16), strings.Repeat("y", 32)} {
		t.Run(fmt.Sprintf("len_%d", len(plain)), func(t *testing.T) {
			enc, err := AesEcbEncrypt(plain, aesKey16, false)
			require.NoError(t, err)
			dec, err := AesEcbDecrypt(enc, aesKey16, false)
			require.NoError(t, err)
			assert.Equal(t, plain, dec)
		})
	}
}

// TestAesEcbUnPaddingEdgeCases PKCS7 去填充边界与损坏密文
//
// 【功能点】触发 unPadding 对非法填充字节保留原文的路径；Base64 解码成功但块倍数非法
// 【测试流程】
//  1. 构造解密后为无效 PKCS7 的密文（手工篡改最后一个块），解密应返回非预期明文但不报错
//  2. Base64 表示长度为 AES 块整数倍但内容为随机字节，解密调用成功但明文乱码
func TestAesEcbUnPaddingEdgeCases(t *testing.T) {
	block, err := aes.NewCipher([]byte(aesKey16))
	require.NoError(t, err)

	raw := make([]byte, aes.BlockSize)
	copy(raw, []byte("fixed-block-text"))
	block.Encrypt(raw, raw)
	// 最后一个字节改为非法 PKCS7 长度值
	raw[aes.BlockSize-1] = byte(aes.BlockSize + 1)
	tampered := base64.StdEncoding.EncodeToString(raw)

	out, err := AesEcbDecrypt(tampered, aesKey16)
	require.NoError(t, err)
	assert.NotEqual(t, "fixed-block-text", out)

	garbage := make([]byte, aes.BlockSize*2)
	_, _ = rand.Read(garbage)
	garbageB64 := base64.StdEncoding.EncodeToString(garbage)
	out2, err := AesEcbDecrypt(garbageB64, aesKey16)
	require.NoError(t, err)
	assert.Len(t, out2, aes.BlockSize*2)
}

// TestRsaReadPemFileErrors RSA 密钥文件读取边界
//
// 【功能点】文件不存在、空文件、非法 PEM 时的 RsaReadPrivatePem / RsaReadPublicPem 错误路径
// 【测试流程】
//  1. 在临时目录下引用不存在的路径读取私钥/公钥，期望包装后的 read 错误
//  2. 写入空文件并读取，期望 convert 阶段报错
//  3. 写入非 PEM 文本并读取，期望 get private/public key error
func TestRsaReadPemFileErrors(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.pem")

	t.Run("private_file_not_exist", func(t *testing.T) {
		_, err := RsaReadPrivatePem(missing)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read private key failed")
	})

	t.Run("public_file_not_exist", func(t *testing.T) {
		_, err := RsaReadPublicPem(missing)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read public key failed")
	})

	emptyPriv := filepath.Join(dir, "empty_private.pem")
	require.NoError(t, os.WriteFile(emptyPriv, nil, 0600))
	t.Run("private_empty_file", func(t *testing.T) {
		_, err := RsaReadPrivatePem(emptyPriv)
		assert.Error(t, err)
	})

	emptyPub := filepath.Join(dir, "empty_public.pem")
	require.NoError(t, os.WriteFile(emptyPub, []byte("   \n"), 0600))
	t.Run("public_whitespace_only", func(t *testing.T) {
		_, err := RsaReadPublicPem(emptyPub)
		assert.Error(t, err)
	})

	gibberish := filepath.Join(dir, "gibberish.pem")
	require.NoError(t, os.WriteFile(gibberish, []byte("not a pem"), 0600))
	t.Run("private_invalid_pem", func(t *testing.T) {
		_, err := RsaReadPrivatePem(gibberish)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "get private key error")
	})
	t.Run("public_invalid_pem", func(t *testing.T) {
		_, err := RsaReadPublicPem(gibberish)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "get public key error")
	})
}

// TestConvertStrToPrivateKeyVariants 私钥 PEM 多种编码与类型校验
//
// 【功能点】PKCS#8 RSA 私钥解析成功路径；PKCS#8 非 RSA 解析失败；PEM 解码失败
// 【测试流程】
//  1. 生成 RSA 私钥，MarshalPKCS8PrivateKey 写入 PEM，调用 convertStrToPrivateKey 成功
//  2. 生成 ECDSA 私钥 PKCS8 PEM，调用 convertStrToPrivateKey 得到「非 RSA」错误
//  3. 非法 PEM 字符串，期望 get private key error
func TestConvertStrToPrivateKeyVariants(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pkcs8DER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)
	pkcs8PEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8DER}))

	t.Run("pkcs8_rsa_ok", func(t *testing.T) {
		got, err := convertStrToPrivateKey(pkcs8PEM)
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, priv.N, got.N)
	})

	ecPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ecDER, err := x509.MarshalPKCS8PrivateKey(ecPriv)
	require.NoError(t, err)
	ecPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecDER}))

	t.Run("pkcs8_non_rsa", func(t *testing.T) {
		_, err := convertStrToPrivateKey(ecPEM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an RSA private key")
	})

	t.Run("pem_decode_nil", func(t *testing.T) {
		_, err := convertStrToPrivateKey("-----BEGIN RSA PRIVATE KEY-----\n!!!\n-----END RSA PRIVATE KEY-----")
		assert.Error(t, err)
	})
}

// TestConvertStrToPublicKeyVariants 公钥 PEM PKIX 与非 RSA 分支
//
// 【功能点】PKIX SubjectPublicKeyInfo 公钥解析；解析出的非 RSA 公钥拒绝
// 【测试流程】
//  1. 由 RSA 私钥得到 PKIX 公钥 PEM（TYPE PUBLIC KEY），convertStrToPublicKey 成功
//  2. ECDSA PKIX 公钥 PEM，期望 parsed key is not an RSA public key
//  3. 无效 PEM，期望 get public key error
func TestConvertStrToPublicKeyVariants(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pixDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pixPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pixDER}))

	t.Run("pkix_rsa_ok", func(t *testing.T) {
		got, err := convertStrToPublicKey(pixPEM)
		assert.NoError(t, err)
		assert.NotNil(t, got)
	})

	ecPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ecPubDER, err := x509.MarshalPKIXPublicKey(&ecPriv.PublicKey)
	require.NoError(t, err)
	ecPubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: ecPubDER}))

	t.Run("pkix_non_rsa", func(t *testing.T) {
		_, err := convertStrToPublicKey(ecPubPEM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an RSA public key")
	})

	t.Run("pem_decode_nil", func(t *testing.T) {
		_, err := convertStrToPublicKey("not pem at all")
		assert.Error(t, err)
	})
}

// TestRsaSavePemWriteFailure savePem 写入失败错误包装
//
// 【功能点】目标路径父目录不存在时 os.WriteFile 失败，savePem 返回包装错误
// 【测试流程】
//  1. 生成 RSA 密钥对
//  2. 使用层级路径 middle/not_exist/key.pem（中间目录未创建）调用 RsaSavePrivatePem / RsaSavePublicPem
//  3. 断言错误非空且包含路径片段
func TestRsaSavePemWriteFailure(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "no_parent_dir_yet", "key.pem")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("private_save_fails", func(t *testing.T) {
		err := RsaSavePrivatePem(priv, badPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save")
	})

	t.Run("public_save_fails", func(t *testing.T) {
		err := RsaSavePublicPem(&priv.PublicKey, badPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save")
	})
}

// TestRsaEncryptDecryptNilAnd_invalidInputs RSA 加解密空密钥与非法输入
//
// 【功能点】RsaEncrypt/RsaDecrypt 空密钥；RsaDecryptFromBase64 非法 Base64 与损坏密文
// 【测试流程】
//  1. publicKey 为 nil 调用 RsaEncrypt / RsaEncrypt2Base64，期望 rsa public key is empty
//  2. privateKey 为 nil 调用 RsaDecrypt / RsaDecryptFromBase64，期望 rsa private key is empty
//  3. Base64 非法字符串解密，期望解码错误
//  4. 随机字节密文解密，期望 DecryptPKCS1v15 报错
func TestRsaEncryptDecryptNilAnd_invalidInputs(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("encrypt_nil_public", func(t *testing.T) {
		_, err := RsaEncrypt(nil, "x")
		assert.EqualError(t, err, `rsa public key is empty`)
	})
	t.Run("encrypt2base64_nil_public", func(t *testing.T) {
		_, err := RsaEncrypt2Base64(nil, "x")
		assert.Error(t, err)
	})

	t.Run("decrypt_nil_private", func(t *testing.T) {
		_, err := RsaDecrypt(nil, []byte{1})
		assert.EqualError(t, err, `rsa private key is empty`)
	})

	t.Run("decrypt_from_base64_nil_private", func(t *testing.T) {
		_, err := RsaDecryptFromBase64(nil, "abcd")
		assert.Error(t, err)
	})

	t.Run("decrypt_from_base64_bad_b64", func(t *testing.T) {
		_, err := RsaDecryptFromBase64(priv, "%%%invalid%%%")
		assert.Error(t, err)
	})

	garbage := make([]byte, priv.Size())
	_, _ = rand.Read(garbage)
	t.Run("decrypt_corrupted_cipher", func(t *testing.T) {
		_, err := RsaDecrypt(priv, garbage)
		assert.Error(t, err)
	})
}

// TestRsaSignVerifyFailures RSA 签名与验签失败路径
//
// 【功能点】私钥为空；Base64 签名解码失败；验签使用错误签名或篡改明文
// 【测试流程】
//  1. RsaSign / RsaSign2Base64 私钥为 nil 报错
//  2. RsaValidSignFromBase64 传入非法 Base64，期望解码错误
//  3. 合法签名对应错误明文，RsaValidSign 返回验证失败
func TestRsaSignVerifyFailures(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pub := &priv.PublicKey

	t.Run("sign_nil_private", func(t *testing.T) {
		_, err := RsaSign(nil, "msg")
		assert.Error(t, err)
	})
	t.Run("sign2base64_nil_private", func(t *testing.T) {
		_, err := RsaSign2Base64(nil, "msg")
		assert.Error(t, err)
	})

	sig, err := RsaSign2Base64(priv, "original")
	require.NoError(t, err)

	t.Run("valid_sign_from_base64_bad_b64", func(t *testing.T) {
		err := RsaValidSignFromBase64(pub, "original", "###")
		assert.Error(t, err)
	})

	t.Run("valid_sign_wrong_plaintext", func(t *testing.T) {
		err := RsaValidSign(pub, "tampered", decodeB64(t, sig))
		assert.Error(t, err)
	})

	wrongSig := base64.StdEncoding.EncodeToString(make([]byte, 64))
	t.Run("valid_sign_wrong_signature_bytes", func(t *testing.T) {
		err := RsaValidSign(pub, "original", decodeB64(t, wrongSig))
		assert.Error(t, err)
	})
}

func decodeB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return b
}

// TestRsaGeneratePrivateKeyInvalidBits RSA 密钥位数过小错误路径
//
// 【功能点】bits < 1024 时 rsa.GenerateKey 返回错误，覆盖 RsaGeneratePrivateKey 的错误分支
// 【测试流程】
//  1. 调用 RsaGeneratePrivateKey(512)
//  2. 断言返回错误且私钥为 nil
func TestRsaGeneratePrivateKeyInvalidBits(t *testing.T) {
	key, err := RsaGeneratePrivateKey(512)
	assert.Nil(t, key)
	assert.Error(t, err)
}

// TestAesEcbUnNoPaddingAllZerosTrail AES ECB 无填充解密全零尾块
//
// 【功能点】isPad=false 且明文为整块零字节时，unNoPadding 扫描到底返回整块（覆盖末尾 return src）
// 【测试流程】
//  1. 构造 16 字节全零明文，使用 AesEcbEncrypt(..., false) 加密
//  2. AesEcbDecrypt(..., false) 解密
//  3. 断言解密结果为 16 个 \\x00
func TestAesEcbUnNoPaddingAllZerosTrail(t *testing.T) {
	zeros := string(make([]byte, aes.BlockSize))
	enc, err := AesEcbEncrypt(zeros, aesKey16, false)
	require.NoError(t, err)
	dec, err := AesEcbDecrypt(enc, aesKey16, false)
	require.NoError(t, err)
	assert.Equal(t, zeros, dec)
	assert.Equal(t, aes.BlockSize, len(dec))
}

// TestAesEcbUnPaddingInconsistentPKCS7 PKCS7 填充字节不一致
//
// 【功能点】解密后最后一字节为合法填充长度，但填充区内存在不等于该值的字节，触发 unPadding 循环提前返回原文
// 【测试流程】
//  1. 使用密钥加密已知明文得到合法密文并 Base64 解码为字节
//  2. 篡改解密结果倒数第二个字节（仍保持最后一个字节为原 PKCS7 长度）
//  3. 将篡改后的缓冲区再用密钥 ECB 加密回「伪造」密文块并 Base64 编码后调用 AesEcbDecrypt
//  4. 断言解密得到的字符串仍包含乱码而不等于原始明文（未触发 panic）
func TestAesEcbUnPaddingInconsistentPKCS7(t *testing.T) {
	plain := "pkcs7-edge"
	cipherB64, err := AesEcbEncrypt(plain, aesKey16)
	require.NoError(t, err)
	raw, err := base64.StdEncoding.DecodeString(cipherB64)
	require.NoError(t, err)
	require.Len(t, raw, aes.BlockSize)

	block, err := aes.NewCipher([]byte(aesKey16))
	require.NoError(t, err)
	decBuf := make([]byte, aes.BlockSize)
	block.Decrypt(decBuf, raw)
	padLen := int(decBuf[aes.BlockSize-1])
	require.GreaterOrEqual(t, padLen, 1)
	require.LessOrEqual(t, padLen, aes.BlockSize)
	if padLen >= 2 {
		decBuf[aes.BlockSize-2] ^= 0xFF
	}
	tamperedCipher := make([]byte, aes.BlockSize)
	block.Encrypt(tamperedCipher, decBuf)
	tamperedB64 := base64.StdEncoding.EncodeToString(tamperedCipher)

	got, err := AesEcbDecrypt(tamperedB64, aesKey16)
	require.NoError(t, err)
	assert.NotEqual(t, plain, got)
}

// TestConvertStrToPrivateKeyBothParsesFail PKCS1 与 PKCS8 解析均失败
//
// 【功能点】合法 PEM 块但 DER 既非 PKCS1 私钥也非 PKCS8，覆盖 convertStrToPrivateKey 末尾返回解析错误
// 【测试流程】
//  1. 构造 TYPE 为 RSA PRIVATE KEY、Bytes 为截断 ASN.1 的 PEM 字符串
//  2. 调用 convertStrToPrivateKey，断言返回错误
func TestConvertStrToPrivateKeyBothParsesFail(t *testing.T) {
	invalidDER := []byte{0x30, 0x03, 0x01, 0x01, 0xff}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: invalidDER}))
	_, err := convertStrToPrivateKey(pemStr)
	assert.Error(t, err)
}

// TestConvertStrToPublicKeyBothParsesFail PKCS1 与 PKIX 公钥解析均失败
//
// 【功能点】RSA PUBLIC KEY PEM 但 DER 无效，PKCS1 与 PKIX 均解析失败
// 【测试流程】
//  1. 构造 TYPE 为 RSA PUBLIC KEY、Bytes 为无效 DER 的 PEM
//  2. convertStrToPublicKey 应返回错误
func TestConvertStrToPublicKeyBothParsesFail(t *testing.T) {
	invalidDER := []byte{0x30, 0x03, 0x02, 0x01, 0x00}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: invalidDER}))
	_, err := convertStrToPublicKey(pemStr)
	assert.Error(t, err)
}

// TestRsaReadWritePemRoundTripInTempDir 临时目录中公钥私钥 PEM 读写往返
//
// 【功能点】在 t.TempDir() 下验证 RsaSave* / RsaRead* 成功路径与密钥材质一致
// 【测试流程】
//  1. 生成 2048 位密钥并写入临时目录下的 private.pem、public.pem
//  2. RsaReadPrivatePem / RsaReadPublicPem 读回并与原始密钥比对模数 N
func TestRsaReadWritePemRoundTripInTempDir(t *testing.T) {
	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	require.NoError(t, RsaSavePrivatePem(priv, privPath))
	require.NoError(t, RsaSavePublicPem(&priv.PublicKey, pubPath))

	readPriv, err := RsaReadPrivatePem(privPath)
	require.NoError(t, err)
	readPub, err := RsaReadPublicPem(pubPath)
	require.NoError(t, err)

	assert.Equal(t, priv.N, readPriv.N)
	assert.Equal(t, priv.PublicKey.N, readPub.N)
}

// TestUnPaddingEmptySlice PKCS7 unPadding 空切片短路
//
// 【功能点】unPadding 在长度为 0 时直接返回，覆盖 AES 公开接口难以触发的防御分支
// 【测试流程】
//  1. 调用 unPadding(nil 或空切片)
//  2. 断言返回与原切片等价（长度为零）
func TestUnPaddingEmptySlice(t *testing.T) {
	assert.Len(t, unPadding(nil), 0)
	assert.Len(t, unPadding([]byte{}), 0)
}

// TestRsaEncryptPlaintextTooLarge RSA 明文超过公钥允许长度
//
// 【功能点】明文长度超过 PKCS1v15 加密上限时 EncryptPKCS1v15 返回错误
// 【测试流程】
//  1. 生成 2048 位密钥对
//  2. 构造长度大于密钥可容纳上限的字节串调用 RsaEncrypt
//  3. 断言返回错误
func TestRsaEncryptPlaintextTooLarge(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pub := &priv.PublicKey
	maxLen := (pub.N.BitLen()+7)/8 - 11 // PKCS1 v1.5 overhead
	huge := strings.Repeat("x", maxLen+1)

	_, err = RsaEncrypt(pub, huge)
	assert.Error(t, err)
}

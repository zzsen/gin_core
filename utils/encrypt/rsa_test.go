package encrypt

import (
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRsaGeneratePrivateKey(t *testing.T) {
	t.Run("rsa generate private key", func(t *testing.T) {
		_, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)
	})
}

func TestRsaSavePrivatePem(t *testing.T) {
	t.Run("rsa save private pem", func(t *testing.T) {
		fileName := fmt.Sprintf("private_key_%s.pem", time.Now().Format("20060102150405"))
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		RsaSavePrivatePem(privateKey, fileName)
		assert.Nil(t, err)

		exists := assert.FileExists(t, fileName)
		if exists {
			os.Remove(fileName)
		}
	})
}

func TestRsaSavePublicPem(t *testing.T) {
	t.Run("rsa save public pem", func(t *testing.T) {
		fileName := fmt.Sprintf("public_key_%s.pem", time.Now().Format("20060102150405"))
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		err = RsaSavePublicPem(&privateKey.PublicKey, fileName)
		assert.Nil(t, err)

		exists := assert.FileExists(t, fileName)
		if exists {
			os.Remove(fileName)
		}
	})
}

func TestRsaReadPrivatePem(t *testing.T) {
	t.Run("rsa read private pem", func(t *testing.T) {
		fileName := fmt.Sprintf("private_key_%s.pem", time.Now().Format("20060102150405"))
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		err = RsaSavePrivatePem(privateKey, fileName)
		assert.Nil(t, err)

		exists := assert.FileExists(t, fileName)
		if exists {
			_, err = RsaReadPrivatePem(fileName)
			assert.Nil(t, err)
			os.Remove(fileName)
		}
	})
}

func TestRsaReadPublicPem(t *testing.T) {
	t.Run("rsa read public pem", func(t *testing.T) {
		fileName := fmt.Sprintf("public_key_%s.pem", time.Now().Format("20060102150405"))
		privateKey, err := RsaGeneratePrivateKey(2048)
		assert.Nil(t, err)

		err = RsaSavePublicPem(&privateKey.PublicKey, fileName)
		assert.Nil(t, err)

		exists := assert.FileExists(t, fileName)
		if exists {
			_, err = RsaReadPublicPem(fileName)
			assert.Nil(t, err)
			os.Remove(fileName)
		}
	})
}

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

const publicKeyStr = `-----BEGIN RSA PUBLIC KEY-----
MIIBCgKCAQEA+GKTixVsQR79E1BAs/IaxJgFS0eELrJvAWkeQ+tdEabYGpWXYoiu
eiT8gpyU2TkLQGDvR+Zr7/TRUtCYwzF4zsLDRW9gokCKsIJDqflFP2q6zPX2yuwf
LUE8g0U3NIpqD34kQJguyt2Fn5IsaWE2tVFXczyyEHETugCbthK/oNfgn3THDYqw
TMGwPtQz5ifi8VncwdaWJdnbbDPmKi1iSeNVHTZv4rKdv7oPDep7eaA2qR4FLMyC
QyPtfDImBypLbnLjDdaBSw8hIGEe9VbHHrCWwta34bBYnSi86Dm7LsyAsjOhzR2R
OsXiaTs5/GXBnIdCuKVawLjegKS6Xhd5EwIDAQAB
-----END RSA PUBLIC KEY-----
`

const plainText = "hello world"

const cipherText = "qu8QSBqJiYuJ84z3cDLJ4VfOUeaVw2XjRp4jkrbguAy7X/xKh+o1MWaQmiGlcOQyoRsD5oab+JJ4e3Tw/eYry5OWB1H8N2wxpRYaoZu66zNB65huAoktZTE+RqMaBSvDWYRwCOuV8X2w36zFOjLJTQgz45eWr78oRxOD+8xO30jiR/rCecIHq7EQha8E1QXApoggCatUkj9gRs4TdkIAqPlxUItWp+348IZTJh8U7zy2tf2vy8x3VX7lQ7sqd5Jpm2akIxRcSU5MNJS9gO8o4l0qUpv7BT3DzIgYdqJy3NV/Df7NH270KfuBr2uLVXApKlZBhgNEhCoAozZ9LRDzJw=="

const signText = "t5shPK2KELo8cB49Gtz8IMLkgBi903sVNBFVPGYRaoFXkk4dlx0zbi3qsJ6LDd4qJSwBIx8FpLu1u2zJXFk/iNIhQmG7KSMX5WlzJwgbl956jD2VP8b0lSqLZzO2CPL5WV0JbEcXe08EIlQwdPQRs10tyj7WhmQXmxBxT4CkD0q+1vRLHYAE9yU7K51TezjHdbIqrZC3MpkW8m/ptr7aO4AC/q8x5puqeVEC4OMJKZGxyhXiS+OYkT5cCIpCTo5oHnMZNutfZFM3bmvR64054h6DJk9b2ZJ0KbUFp9HfjzmEDsL74iloTSCe7DBqMik8cSlSf/+DOCShSliXwlJe2Q=="

func TestRsaEncrypt(t *testing.T) {
	t.Run("rsa encrypt", func(t *testing.T) {
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		encryptResult, err := RsaEncrypt(publicKey, plainText)
		assert.Nil(t, err)
		assert.NotNil(t, encryptResult)
	})
}

func TestRsaEncrypt2Base64(t *testing.T) {
	t.Run("rsa encrypt", func(t *testing.T) {
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		result, err := RsaEncrypt2Base64(publicKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, result, "")
	})
}

func TestRsaDecrypt(t *testing.T) {
	t.Run("rsa decrypt", func(t *testing.T) {
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		cipherBytes, err := base64.StdEncoding.DecodeString(cipherText)
		assert.Nil(t, err)

		result, err := RsaDecrypt(privateKey, cipherBytes)
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})
}

func TestRsaDecryptFromBase64(t *testing.T) {
	t.Run("rsa decrypt", func(t *testing.T) {
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		result, err := RsaDecryptFromBase64(privateKey, cipherText)
		assert.Nil(t, err)
		assert.Equal(t, result, plainText)
	})
}

func TestRsaCrypt(t *testing.T) {
	t.Run("rsa crypt", func(t *testing.T) {
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		encryptResult, err := RsaEncrypt2Base64(publicKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, encryptResult, "")

		result, err := RsaDecryptFromBase64(privateKey, encryptResult)
		assert.Nil(t, err)
		assert.Equal(t, result, plainText)
	})
}

func TestRsaSign(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		result, err := RsaSign(privateKey, plainText)
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})
}

func TestRsaSign2Base64(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		result, err := RsaSign2Base64(privateKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, result, "")
	})
}

func TestRsaValidSign(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		cipherBytes, err := base64.StdEncoding.DecodeString(signText)
		assert.Nil(t, err)

		err = RsaValidSign(publicKey, plainText, cipherBytes)
		assert.Nil(t, err)
	})
}

func TestRsaValidSignFromBase64(t *testing.T) {
	t.Run("rsa sign", func(t *testing.T) {
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		err = RsaValidSignFromBase64(publicKey, plainText, signText)
		assert.Nil(t, err)
	})
}

func TestRsaSignAndValid(t *testing.T) {
	t.Run("rsa sign and valid", func(t *testing.T) {
		publicKey, err := convertStrToPublicKey(publicKeyStr)
		assert.Nil(t, err)

		privateKey, err := convertStrToPrivateKey(privateKeyStr)
		assert.Nil(t, err)

		result, err := RsaSign2Base64(privateKey, plainText)
		assert.Nil(t, err)
		assert.NotEqual(t, result, "")

		err = RsaValidSignFromBase64(publicKey, plainText, result)
		assert.Nil(t, err)
	})
}

// Package configcenter ENV/CIPHER 预处理测试
//
// ==================== 测试说明 ====================
// 验证 prepareYAML 对环境变量占位与 CIPHER 解密的处理。
//
// 运行测试：go test ./configcenter/ -count=1 -run PrepareYAML -v
// ==================================================

package configcenter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/utils/encrypt"
)

const testAESKey = "UTabIUiHgDyh464+" // 16 bytes

// TestPrepareYAML_EnvPlaceholder 替换 {{ENV}}
func TestPrepareYAML_EnvPlaceholder(t *testing.T) {
	t.Setenv("CC_TEST_RATE", "77")
	out, err := prepareYAML([]byte("rateLimit:\n  defaultRate: {{CC_TEST_RATE}}\n"), "")
	require.NoError(t, err)
	assert.Contains(t, string(out), "77")
}

// TestPrepareYAML_MissingEnv 缺失环境变量报错
func TestPrepareYAML_MissingEnv(t *testing.T) {
	_ = os.Unsetenv("CC_MISSING_ENV_XYZ")
	_, err := prepareYAML([]byte("x: {{CC_MISSING_ENV_XYZ}}\n"), "")
	require.Error(t, err)
}

// TestPrepareYAML_CIPHER 解密 CIPHER()
func TestPrepareYAML_CIPHER(t *testing.T) {
	cipher, err := encrypt.AesEcbEncrypt("secret", testAESKey)
	require.NoError(t, err)
	raw := []byte("password: CIPHER(" + cipher + ")\n")
	out, err := prepareYAML(raw, testAESKey)
	require.NoError(t, err)
	assert.Contains(t, string(out), "secret")
	assert.NotContains(t, string(out), "CIPHER(")
}

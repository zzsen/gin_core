// Package email SMTP 认证实现测试
//
// ==================== 测试说明 ====================
// 本文件包含 SMTP 认证机制的单元测试，不需要外部依赖。
//
// 测试覆盖内容：
// 1. plainAuth.Start 正确主机名
// 2. plainAuth.Start 错误主机名
// 3. plainAuth.Next 无后续挑战
// 4. plainAuth.Next 有后续挑战（返回错误）
// 5. CRAMMD5Auth 创建
// 6. cramMD5Auth.Start 返回协议名
// 7. cramMD5Auth.Next 挑战响应
// 8. cramMD5Auth.Next 无后续挑战
//
// 运行测试：go test -v ./utils/email/... -run "Test"
// ==================================================
package email

import (
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockServerInfo(name string, tls bool) *smtp.ServerInfo {
	return &smtp.ServerInfo{Name: name, TLS: tls}
}

// TestPlainAuth_Start_CorrectHost 测试 plainAuth 正确主机名
func TestPlainAuth_Start_CorrectHost(t *testing.T) {
	auth := &plainAuth{identity: "", username: "user", password: "pass", host: "mail.example.com"}
	proto, resp, err := auth.Start(mockServerInfo("mail.example.com", false))

	require.NoError(t, err)
	assert.Equal(t, "PLAIN", proto)
	assert.Equal(t, []byte("\x00user\x00pass"), resp)
}

// TestPlainAuth_Start_WrongHost 测试 plainAuth 错误主机名
func TestPlainAuth_Start_WrongHost(t *testing.T) {
	auth := &plainAuth{identity: "", username: "user", password: "pass", host: "mail.example.com"}
	_, _, err := auth.Start(mockServerInfo("evil.example.com", false))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrong host name")
}

// TestPlainAuth_Next_NoMore 测试 plainAuth 无后续挑战
func TestPlainAuth_Next_NoMore(t *testing.T) {
	auth := &plainAuth{}
	resp, err := auth.Next(nil, false)
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

// TestPlainAuth_Next_UnexpectedChallenge 测试 plainAuth 意外挑战
func TestPlainAuth_Next_UnexpectedChallenge(t *testing.T) {
	auth := &plainAuth{}
	_, err := auth.Next([]byte("challenge"), true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected server challenge")
}

// TestCRAMMD5Auth_Create 测试 CRAMMD5Auth 创建
func TestCRAMMD5Auth_Create(t *testing.T) {
	auth := CRAMMD5Auth("user", "secret")
	require.NotNil(t, auth)
}

// TestCRAMMD5Auth_Start 测试 CRAM-MD5 认证启动
func TestCRAMMD5Auth_Start(t *testing.T) {
	auth := CRAMMD5Auth("user", "secret")
	proto, resp, err := auth.Start(mockServerInfo("any.host", false))

	require.NoError(t, err)
	assert.Equal(t, "CRAM-MD5", proto)
	assert.Nil(t, resp)
}

// TestCRAMMD5Auth_Next_WithChallenge 测试 CRAM-MD5 挑战响应
func TestCRAMMD5Auth_Next_WithChallenge(t *testing.T) {
	auth := CRAMMD5Auth("user", "secret")
	resp, err := auth.Next([]byte("test-challenge"), true)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Contains(t, string(resp), "user ")
}

// TestCRAMMD5Auth_Next_NoMore 测试 CRAM-MD5 无后续挑战
func TestCRAMMD5Auth_Next_NoMore(t *testing.T) {
	auth := CRAMMD5Auth("user", "secret")
	resp, err := auth.Next(nil, false)

	assert.NoError(t, err)
	assert.Nil(t, resp)
}

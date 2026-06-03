// Package email HTML 邮件发送功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 HTML 邮件发送（SendHtml / SendHtmlByTLS）和邮件体构造逻辑的单元测试。
// 绝大多数用例无需真实 SMTP：通过连接失败桩、本地隐式 TLS SMTP 桩服务器覆盖成功与错误路径。
//
// 测试覆盖内容：
// 1. BuildHtmlEmail 字段赋值与空字符串处理
// 2. BuildHtmlEmail 字节往返测试
// 3. SendHtml 隐式 TLS 成功发送
// 4. SendHtml NewClient 失败（错误 Greeting）
// 5. SendHtml Auth 失败（服务器拒绝）
// 6. SendHtml Mail/Rcpt 无效发件人/收件人
// 7. SendHtml TLS 握手失败 / 连接被拒
// 8. SendHtml Data/Write/Close/Quit 各阶段失败
// 9. SendHtml 空 Host 处理
// 10. SendHtmlByTLS 连接失败 / 空 Host / 构建消息
//
// 运行测试：go test -short -count=1 -cover ./utils/email/... -run Test
// ==================================================
package email

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// implicitTLSMode 本地隐式 TLS SMTP 桩的行为模式。
type implicitTLSMode int

const (
	implicitTLSModeOK implicitTLSMode = iota
	implicitTLSModeBadGreeting
	implicitTLSModeRejectAuth
	implicitTLSModeRejectData
	implicitTLSModeResetDuringBody
	implicitTLSModeRejectAfterBody
	implicitTLSModeBadQuit
)

func mustLocalhostTLSCert(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"gin_core"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
	}
}

func handleImplicitTLSSMTP(conn net.Conn, mode implicitTLSMode) error {
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	write := func(s string) error {
		if _, err := bw.WriteString(s + "\r\n"); err != nil {
			return err
		}
		return bw.Flush()
	}
	read := func() (string, error) {
		line, err := br.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	switch mode {
	case implicitTLSModeBadGreeting:
		return write("500 bad greeting")
	default:
		if err := write("220 localhost ESMTP ready"); err != nil {
			return err
		}
	}

	for {
		line, err := read()
		if err != nil {
			return err
		}
		u := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(u, "EHLO"):
			if err := write("250-localhost Hello localhost"); err != nil {
				return err
			}
			if err := write("250-AUTH PLAIN"); err != nil {
				return err
			}
			if err := write("250 8BITMIME"); err != nil {
				return err
			}
		case strings.HasPrefix(u, "AUTH"):
			if mode == implicitTLSModeRejectAuth {
				return write("535 Authentication failed")
			}
			if err := write("235 OK"); err != nil {
				return err
			}
		case strings.HasPrefix(u, "MAIL FROM"):
			if err := write("250 OK"); err != nil {
				return err
			}
		case strings.HasPrefix(u, "RCPT TO"):
			if err := write("250 OK"); err != nil {
				return err
			}
		case strings.HasPrefix(u, "DATA"):
			switch mode {
			case implicitTLSModeRejectData:
				return write("554 DATA command rejected")
			case implicitTLSModeResetDuringBody:
				if err := write("354 End data with <CR><LF>.<CR><LF>"); err != nil {
					return err
				}
				if _, err := read(); err != nil {
					return err
				}
				_ = conn.Close()
				return nil
			case implicitTLSModeRejectAfterBody:
				if err := write("354 End data with <CR><LF>.<CR><LF>"); err != nil {
					return err
				}
				for {
					bl, err := read()
					if err != nil {
						return err
					}
					if bl == "." {
						break
					}
				}
				return write("554 Message rejected")
			default:
				if err := write("354 End data with <CR><LF>.<CR><LF>"); err != nil {
					return err
				}
				for {
					bl, err := read()
					if err != nil {
						return err
					}
					if bl == "." {
						break
					}
				}
				if err := write("250 OK"); err != nil {
					return err
				}
			}
		case strings.HasPrefix(u, "QUIT"):
			if mode == implicitTLSModeBadQuit {
				return write("500 bad quit response")
			}
			return write("221 Bye")
		default:
			return fmt.Errorf("unexpected SMTP command: %q", line)
		}
	}
}

func startImplicitTLSFakeSMTP(t *testing.T, mode implicitTLSMode) (host string, port int, cleanup func()) {
	t.Helper()
	cert := mustLocalhostTLSCert(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	portNum, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = handleImplicitTLSSMTP(conn, mode)
	}()

	return host, portNum, func() {
		_ = ln.Close()
		wg.Wait()
	}
}

func closedLocalTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return addr
}

// TestBuildHtmlEmail_FieldAssignment 构造邮件对象字段赋值
// 【功能点】验证 buildHtmlEmail 将 SmtpConfig.Sender、收件人、主题、HTML 正文写入 jordan-wright/email 模型。
// 【测试流程】给定配置与参数 → 调用 buildHtmlEmail → 断言 From / To / Subject / HTML。
func TestBuildHtmlEmail_FieldAssignment(t *testing.T) {
	conf := SmtpConfig{
		Sender:             "noreply@example.com",
		Username:           "user",
		Password:           "secret",
		Host:               "smtp.example.com",
		Port:               587,
		InsecureSkipVerify: false,
	}
	em := buildHtmlEmail(conf, "recv@example.com", "测试主题", "<p>你好</p>")
	require.NotNil(t, em)
	assert.Equal(t, "noreply@example.com", em.From)
	assert.Equal(t, []string{"recv@example.com"}, em.To)
	assert.Equal(t, "测试主题", em.Subject)
	assert.Equal(t, []byte("<p>你好</p>"), em.HTML)
}

// TestBuildHtmlEmail_EmptyStrings 空字符串参数
// 【功能点】确认在未做业务层校验时，构造逻辑仍一致写入空发件人/收件人等字段。
// 【测试流程】Sender/receiver/subject/text 均为空 → 断言各字段为零值或空切片。
func TestBuildHtmlEmail_EmptyStrings(t *testing.T) {
	em := buildHtmlEmail(SmtpConfig{}, "", "", "")
	require.NotNil(t, em)
	assert.Empty(t, em.From)
	assert.Equal(t, []string{""}, em.To)
	assert.Empty(t, em.Subject)
	assert.Empty(t, em.HTML)
}

// TestSendHtml_ImplicitTLS_Success 隐式 TLS 全流程成功
// 【功能点】覆盖 SendHtml 的 tls.Dial、NewClient、Auth、Mail、Rcpt、Data、写入正文、Quit。
// 【测试流程】启动本地 tls.Listen SMTP 桩 → InsecureSkipVerify=true 调用 SendHtml → 断言成功。
func TestSendHtml_ImplicitTLS_Success(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeOK)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "user",
		Password:           "pass",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "Subject", "<html><body>hi</body></html>")
	require.NoError(t, err)
}

// TestSendHtml_NewClientFails_BadGreeting 非 220 问候语
// 【功能点】smtp.NewClient 读取首行响应失败时的错误返回。
// 【测试流程】桩服务器 TLS 握手后发送 500 → SendHtml 在 NewClient 处返回错误。
func TestSendHtml_NewClientFails_BadGreeting(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeBadGreeting)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "t")
	require.Error(t, err)
}

// TestSendHtml_AuthFails_ServerRejects 认证被拒绝
// 【功能点】client.Auth 失败时的错误路径。
// 【测试流程】桩服务器对 AUTH 返回 535 → SendHtml 返回错误。
func TestSendHtml_AuthFails_ServerRejects(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeRejectAuth)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "t")
	require.Error(t, err)
}

// TestSendHtml_Mail_InvalidSender 发件人含非法换行
// 【功能点】net/smtp Client.Mail 在校验 From 行含 CR/LF 时失败。
// 【测试流程】桩服务器允许认证 → Sender 含 "\\n" → Mail 报错且不依赖 DATA。
func TestSendHtml_Mail_InvalidSender(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeOK)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "bad\nfrom@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp:")
}

// TestSendHtml_Rcpt_InvalidReceiver 收件人含非法换行
// 【功能点】Rcpt 对非法地址行的校验错误路径。
// 【测试流程】Mail 成功 → receiver 含 "\\r" → Rcpt 返回错误。
func TestSendHtml_Rcpt_InvalidReceiver(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeOK)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "bad\rto@example.com", "s", "t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp:")
}

// TestSendHtml_TLSHandshakeFails_VerifyCertificate TLS 校验证书失败
// 【功能点】InsecureSkipVerify=false 时自签名证书导致 tls.Dial 失败。
// 【测试流程】本地 TLS 桩使用自签名证书 → SendHtml 在校验阶段返回错误。
func TestSendHtml_TLSHandshakeFails_VerifyCertificate(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeOK)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: false,
	}
	err := SendHtml(conf, "to@example.com", "s", "t")
	require.Error(t, err)
}

// TestSendHtml_TLSConnectionFails_Refused 连接被拒绝
// 【功能点】tls.Dial 无法建立 TCP 连接时的错误返回。
// 【测试流程】使用已关闭的本地端口 → SendHtml 在 Dial 阶段失败。
func TestSendHtml_TLSConnectionFails_Refused(t *testing.T) {
	port := closedLocalTCPPort(t)
	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               "127.0.0.1",
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "t")
	require.Error(t, err)
}

// TestSendHtml_EmptyHost 空 Host 地址
// 【功能点】fmt.Sprintf 拼接出无效地址时连接失败的错误路径。
// 【测试流程】Host 为空、端口为已关闭本地端口 → 断言返回错误。
func TestSendHtml_EmptyHost(t *testing.T) {
	port := closedLocalTCPPort(t)
	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               "",
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "t")
	require.Error(t, err)
}

// TestSendHtmlByTLS_ConnectionFails_Refused STARTTLS 路径连接失败
// 【功能点】SendHtmlByTLS 在构造邮件后调用 em.Send，连接被拒绝时返回错误。
// 【测试流程】closedLocalTCPPort → SendHtmlByTLS 报错。
func TestSendHtmlByTLS_ConnectionFails_Refused(t *testing.T) {
	port := closedLocalTCPPort(t)
	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               "127.0.0.1",
		Port:               port,
		InsecureSkipVerify: false,
	}
	err := SendHtmlByTLS(conf, "to@example.com", "subj", "<p>x</p>")
	require.Error(t, err)
}

// TestSendHtmlByTLS_EmptyHost 空 Host（PlainAuth 与拨号地址）
// 【功能点】SendHtmlByTLS 使用 conf.Host 拼地址与 PlainAuth 域名；空 Host 时应快速连接失败。
// 【测试流程】Host 为空、端口为关闭端口 → 断言错误。
func TestSendHtmlByTLS_EmptyHost(t *testing.T) {
	port := closedLocalTCPPort(t)
	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               "",
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtmlByTLS(conf, "to@example.com", "subj", "<p>x</p>")
	require.Error(t, err)
}

// TestSendHtmlByTLS_BuildsMessageBeforeSend 发送前构造邮件体
// 【功能点】即便底层 Send 失败，仍会走 buildHtmlEmail 与拨号逻辑（覆盖 em.Send 调用链前置步骤）。
// 【测试流程】连接拒绝 → 错误非 nil（证明已尝试发送）。
func TestSendHtmlByTLS_BuildsMessageBeforeSend(t *testing.T) {
	port := closedLocalTCPPort(t)
	conf := SmtpConfig{
		Sender:             "sender@example.com",
		Username:           "user",
		Password:           "pwd",
		Host:               "127.0.0.1",
		Port:               port,
		InsecureSkipVerify: true,
	}
	subject := "题目"
	body := "<html/>"
	err := SendHtmlByTLS(conf, "dst@example.com", subject, body)
	require.Error(t, err)
}

// TestSendHtml_DataFails_ServerRefuses 服务端拒绝 DATA
// 【功能点】覆盖 SendHtml 中 client.Data() 返回错误的分支（SMTP 返回非 354）。
// 【测试流程】隐式 TLS 桩对 DATA 回复 554 → SendHtml 在开启 DATA 流前失败。
func TestSendHtml_DataFails_ServerRefuses(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeRejectData)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "<p>x</p>")
	require.Error(t, err)
}

// TestSendHtml_WriteFails_ServerClosesMidBody 正文写入中途连接断开
// 【功能点】覆盖 DATA 写入阶段 w.Write 因连接被关闭而失败的错误路径。
// 【测试流程】桩在 354 后读完首行正文即关闭连接 → 客户端后续写入报错。
func TestSendHtml_WriteFails_ServerClosesMidBody(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeResetDuringBody)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "<p>long-body-"+strings.Repeat("x", 4096)+"</p>")
	require.Error(t, err)
}

// TestSendHtml_CloseFails_ServerRejectsAfterDot DATA 结束后 SMTP 否定响应
// 【功能点】覆盖 w.Close() 读取服务器应答时发现事务失败的分支。
// 【测试流程】桩在收到完整正文并以 "." 结束后回复 554 → Close 返回错误。
func TestSendHtml_CloseFails_ServerRejectsAfterDot(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeRejectAfterBody)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "<p>ok</p>")
	require.Error(t, err)
}

// TestSendHtml_QuitFails_ServerBadResponse QUIT 应答异常
// 【功能点】覆盖 SendHtml 末尾 client.Quit() 收到非成功应答时的错误路径。
// 【测试流程】邮件投递成功后桩对 QUIT 返回 500 → Quit 报错。
func TestSendHtml_QuitFails_ServerBadResponse(t *testing.T) {
	host, port, cleanup := startImplicitTLSFakeSMTP(t, implicitTLSModeBadQuit)
	defer cleanup()

	conf := SmtpConfig{
		Sender:             "from@example.com",
		Username:           "u",
		Password:           "p",
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	}
	err := SendHtml(conf, "to@example.com", "s", "<p>x</p>")
	require.Error(t, err)
}

// TestBuildHtmlEmail_BytesRoundTrip 序列化邮件字节流
// 【功能点】验证 buildHtmlEmail 产物可被 jordan-wright/email 序列化为合法 MIME（间接覆盖 SendHtml 中 em.Bytes 成功路径的数据形状）。
// 【测试流程】构造 SmtpConfig 与正文 → buildHtmlEmail → Bytes → 断言包含 From、主题与 HTML 片段。
func TestBuildHtmlEmail_BytesRoundTrip(t *testing.T) {
	conf := SmtpConfig{
		Sender:   "me@example.com",
		Username: "u",
		Password: "p",
		Host:     "smtp.example.com",
		Port:     465,
	}
	em := buildHtmlEmail(conf, "you@example.com", "Hello", "<b>正文</b>")
	raw, err := em.Bytes()
	require.NoError(t, err)
	s := string(raw)
	assert.Contains(t, s, "From: <me@example.com>")
	assert.Contains(t, s, "To: <you@example.com>")
	assert.Contains(t, s, "Subject: Hello")
	// UTF-8 正文经 quoted-printable 编码
	assert.Contains(t, s, "=E6=AD=A3=E6=96=87")
}

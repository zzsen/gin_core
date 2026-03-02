// Package email 提供基于 SMTP 协议的邮件发送工具。
//
// 支持两种发送方式：
//   - SendHtmlByTLS: 使用 STARTTLS 方式发送（适用于 587 端口）
//   - SendHtml: 使用直接 TLS 连接发送（适用于 465 端口）
package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

// SmtpConfig SMTP 邮件服务器配置
type SmtpConfig struct {
	Sender             string // 发件人地址（如 "noreply@example.com"）
	Username           string // SMTP 认证用户名
	Password           string // SMTP 认证密码
	Host               string // SMTP 服务器地址（如 "smtp.example.com"）
	Port               int    // SMTP 服务器端口（TLS 通常为 465，STARTTLS 通常为 587）
	InsecureSkipVerify bool   // 是否跳过 TLS 证书验证，默认 false（启用验证）；仅在自签名证书等场景下设为 true
}

// SendHtmlByTLS 通过 STARTTLS 方式发送 HTML 邮件。
// 适用于支持 STARTTLS 升级的 SMTP 服务器（通常使用 587 端口）。
//
// 参数：
//   - conf: SMTP 服务器配置
//   - receiver: 收件人邮箱地址
//   - subject: 邮件主题
//   - text: HTML 格式的邮件正文
//
// 返回：
//   - error: 发送失败时返回错误
func SendHtmlByTLS(conf SmtpConfig, receiver string, subject string, text string) error {
	em := email.NewEmail()
	// 设置 sender 发送方 的邮箱 ， 此处可以填写自己的邮箱
	em.From = conf.Sender

	// 设置 receiver 接收方 的邮箱  此处也可以填写自己的邮箱， 就是自己发邮件给自己
	em.To = []string{receiver}

	// 设置主题
	em.Subject = subject

	// 简单设置文件发送的内容，暂时设置成纯文本
	em.HTML = []byte(text)
	//设置服务器相关的配置
	err := em.Send(fmt.Sprintf("%s:%d", conf.Host, conf.Port), smtp.PlainAuth("", conf.Username, conf.Password, conf.Host))
	return err
}

// SendHtml 通过直接 TLS 连接发送 HTML 邮件。
// 适用于要求直接建立 TLS 连接的 SMTP 服务器（通常使用 465 端口）。
//
// 参数：
//   - conf: SMTP 服务器配置
//   - receiver: 收件人邮箱地址
//   - subject: 邮件主题
//   - text: HTML 格式的邮件正文
//
// 返回：
//   - error: 发送失败时返回错误
//
// 该方法手动建立 TLS 连接并完成 SMTP 会话，按以下顺序执行：
// 1. 建立 TLS 连接
// 2. 创建 SMTP 客户端并认证
// 3. 设置发件人、收件人
// 4. 构造邮件内容并写入
// 5. 发送 QUIT 关闭会话
func SendHtml(conf SmtpConfig, receiver string, subject string, text string) error {
	auth := &plainAuth{"", conf.Username, conf.Password, conf.Host}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: conf.InsecureSkipVerify,
		ServerName:         conf.Host,
	}

	// 1. 建立 TLS 连接
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", conf.Host, conf.Port), tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 2. 创建 SMTP 客户端并认证
	client, err := smtp.NewClient(conn, conf.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return err
	}

	// 3. 设置发件人、收件人
	if err = client.Mail(conf.Sender); err != nil {
		return err
	}

	if err = client.Rcpt(receiver); err != nil {
		return err
	}

	// 4. 构造邮件内容并写入
	w, err := client.Data()
	if err != nil {
		return err
	}

	em := email.NewEmail()
	// 设置 sender 发送方 的邮箱
	em.From = conf.Sender
	// 设置 receiver 接收方的邮箱
	em.To = []string{receiver}
	// 设置主题
	em.Subject = subject
	// 简单设置文件发送的内容，暂时设置成纯文本
	em.HTML = []byte(text)

	raw, err := em.Bytes()
	if err != nil {
		return fmt.Errorf("serialize email failed: %w", err)
	}

	if _, err = w.Write(raw); err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}

	// 5. 发送 QUIT 关闭会话
	return client.Quit()
}

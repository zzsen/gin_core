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
	Sender   string // 发件人地址（如 "noreply@example.com"）
	Username string // SMTP 认证用户名
	Password string // SMTP 认证密码
	Host     string // SMTP 服务器地址（如 "smtp.example.com"）
	Port     int    // SMTP 服务器端口（TLS 通常为 465，STARTTLS 通常为 587）
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
func SendHtml(conf SmtpConfig, receiver string, subject string, text string) error {
	auth := &plainAuth{"", conf.Username, conf.Password, conf.Host}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         conf.Host,
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", conf.Host, conf.Port), tlsConfig)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, conf.Host)
	if err != nil {
		return err
	}

	if err = client.Auth(auth); err != nil {
		if err != nil {
			return err
		}
	}

	if err = client.Mail(conf.Sender); err != nil {
		if err != nil {
			return err
		}
	}

	if err = client.Rcpt(receiver); err != nil {
		if err != nil {
			return err
		}
	}

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
	raw, _ := em.Bytes()
	_, err = w.Write(raw)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

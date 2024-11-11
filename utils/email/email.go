package email

import (
	"crypto/tls"
	"fmt"
	"github.com/jordan-wright/email"
	"net/smtp"
)

type SmtpConfig struct {
	Sender   string
	Username string
	Password string
	Host     string
	Port     int
}

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

// Package config 提供应用程序的配置结构定义
// 本文件定义了SMTP邮件服务的配置结构，用于发送邮件通知
package config

// SmtpInfo SMTP邮件服务配置信息
// 该结构体包含了连接SMTP服务器和发送邮件所需的基本配置参数
type SmtpInfo struct {
	Host     string `yaml:"host"`     // SMTP服务器地址
	Username string `yaml:"username"` // SMTP服务器用户名，通常是邮箱地址
	Password string `yaml:"password"` // SMTP服务器密码，可能是邮箱密码或应用专用密码
	Sender   string `yaml:"sender"`   // 发件人邮箱地址，用于标识邮件的来源
}

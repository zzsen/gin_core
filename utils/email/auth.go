// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package email

import (
	"crypto/hmac"
	"crypto/md5"
	"errors"
	"fmt"
	"net/smtp"
)

// plainAuth 实现 PLAIN 认证机制（RFC 4616）的自定义版本。
//
// 与标准库 smtp.PlainAuth 的区别：移除了强制 TLS 检查，允许在非 TLS 连接上发送凭据。
// 这是为了兼容部分不支持 STARTTLS 但已通过其他方式（如 VPN、内网）保障安全的 SMTP 服务器。
//
// ⚠️ 安全风险：在非 TLS 连接上使用 PLAIN 认证，用户名和密码将以明文方式传输，
// 存在被中间人攻击截获的风险。仅应在已确认网络链路安全的环境中使用。
type plainAuth struct {
	identity, username, password string
	host                         string
}

// Start 实现 smtp.Auth 接口，发起 PLAIN 认证。
// 与标准库不同，此实现跳过了 TLS 连接检查，允许在非加密连接上发送认证凭据。
// 仍会校验服务器主机名，防止凭据发送到错误的服务器。
func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

// Next 实现 smtp.Auth 接口，处理服务器的后续挑战。
// PLAIN 认证为单步认证，不期望收到后续挑战，收到时返回错误。
func (a *plainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

// cramMD5Auth 实现 CRAM-MD5 认证机制（RFC 2195）。
// 使用挑战-响应方式认证，密码不会以明文传输，安全性优于 PLAIN 认证。
type cramMD5Auth struct {
	username, secret string
}

// CRAMMD5Auth 创建 CRAM-MD5 认证实例（RFC 2195）。
// 使用给定的用户名和密钥，通过挑战-响应机制向服务器进行身份认证。
func CRAMMD5Auth(username, secret string) smtp.Auth {
	return &cramMD5Auth{username, secret}
}

// Start 实现 smtp.Auth 接口，声明使用 CRAM-MD5 认证方式。
// 首次握手不发送任何数据，等待服务器发送挑战字符串。
func (a *cramMD5Auth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "CRAM-MD5", nil, nil
}

// Next 实现 smtp.Auth 接口，使用 HMAC-MD5 对服务器挑战进行签名响应。
// 响应格式为 "username digest"，其中 digest 是以 secret 为密钥对挑战做 HMAC-MD5 的十六进制结果。
func (a *cramMD5Auth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		d := hmac.New(md5.New, []byte(a.secret))
		d.Write(fromServer)
		s := make([]byte, 0, d.Size())
		return []byte(fmt.Sprintf("%s %x", a.username, d.Sum(s))), nil
	}
	return nil, nil
}

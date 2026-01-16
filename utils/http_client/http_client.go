// Package http_client HTTP客户端工具包
// 提供常用的HTTP请求方法，支持GET、POST、PUT等请求类型
// 支持表单提交、文件上传、JSON请求等功能
// 已优化：连接池复用、链路追踪、请求重试
package http_client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// ResponseWrapper HTTP响应包装器
// 用于统一封装HTTP请求的响应结果
type ResponseWrapper struct {
	StatusCode int         // HTTP状态码
	Body       string      // 响应体内容
	Header     http.Header // 响应头信息
	Error      error       // 错误信息（新增）
}

// IsSuccess 判断请求是否成功
func (r *ResponseWrapper) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// HasError 判断是否有错误
func (r *ResponseWrapper) HasError() bool {
	return r.Error != nil || r.StatusCode == 0
}

// ==================== 基于 context.Context 的新 API ====================

// GetWithContext 发送GET请求（推荐使用）
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - url: 请求的URL地址
//   - timeout: 请求超时时间（秒），0 表示使用默认超时
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func GetWithContext(ctx context.Context, url string, timeout int, headers map[string]string) ResponseWrapper {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return createRequestError(err)
	}

	setHeaders(req, headers)
	return doRequest(ctx, req, timeout)
}

// PostJSONWithContext 发送POST请求（JSON格式，推荐使用）
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - url: 请求的URL地址
//   - body: JSON格式的请求体
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PostJSONWithContext(ctx context.Context, url string, body string, timeout int, headers map[string]string) ResponseWrapper {
	buf := bytes.NewBufferString(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return createRequestError(err)
	}

	req.Header.Set("Content-Type", "application/json")
	setHeaders(req, headers)
	return doRequest(ctx, req, timeout)
}

// PostParamsWithContext 发送POST请求（参数形式，推荐使用）
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - url: 请求的URL地址
//   - params: POST参数（字符串形式）
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PostParamsWithContext(ctx context.Context, url string, params string, timeout int, headers map[string]string) ResponseWrapper {
	buf := bytes.NewBufferString(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return createRequestError(err)
	}

	setHeaders(req, headers)
	return doRequest(ctx, req, timeout)
}

// PutJSONWithContext 发送PUT请求（JSON格式，推荐使用）
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - url: 请求的URL地址
//   - body: JSON格式的请求体
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PutJSONWithContext(ctx context.Context, url string, body string, timeout int, headers map[string]string) ResponseWrapper {
	buf := bytes.NewBufferString(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, buf)
	if err != nil {
		return createRequestError(err)
	}

	req.Header.Set("Content-Type", "application/json")
	setHeaders(req, headers)
	return doRequest(ctx, req, timeout)
}

// DeleteWithContext 发送DELETE请求（推荐使用）
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - url: 请求的URL地址
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func DeleteWithContext(ctx context.Context, url string, timeout int, headers map[string]string) ResponseWrapper {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return createRequestError(err)
	}

	setHeaders(req, headers)
	return doRequest(ctx, req, timeout)
}

// PostFormWithContext 发送POST请求（表单形式，支持文件上传，推荐使用）
// 参数:
//   - ctx: 上下文，用于超时控制和链路追踪
//   - httpUrl: 请求的URL地址
//   - dataMap: 普通表单字段（键值对）
//   - filePathMap: 文件路径映射（字段名 -> 文件路径）
//   - fileMap: 文件读取器映射（字段名 -> io.Reader）
//   - headers: 自定义请求头
//   - timeout: 请求超时时间（秒）
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PostFormWithContext(ctx context.Context, httpUrl string, dataMap map[string]string,
	filePathMap map[string]string, fileMap map[string]io.Reader,
	headers map[string]string, timeout int) ResponseWrapper {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 写入普通表单字段
	for key, value := range dataMap {
		if err := writer.WriteField(key, value); err != nil {
			return ResponseWrapper{0, fmt.Sprintf("写入表单字段 %s 失败: %s", key, err.Error()), make(http.Header), err}
		}
	}

	// 处理文件路径映射
	for fieldName, filePath := range filePathMap {
		file, err := os.Open(filePath)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("打开文件 %s 失败: %s", filePath, err.Error()), make(http.Header), err}
		}
		defer file.Close()

		part, err := writer.CreateFormFile(fieldName, filePath)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("创建表单文件字段失败: %s", err.Error()), make(http.Header), err}
		}

		if _, err = io.Copy(part, file); err != nil {
			return ResponseWrapper{0, fmt.Sprintf("复制文件内容失败: %s", err.Error()), make(http.Header), err}
		}
	}

	// 处理文件读取器映射
	for fieldName, fileReader := range fileMap {
		part, err := writer.CreateFormFile(fieldName, fieldName)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("创建表单文件字段失败: %s", err.Error()), make(http.Header), err}
		}

		if _, err = io.Copy(part, fileReader); err != nil {
			return ResponseWrapper{0, fmt.Sprintf("复制文件内容失败: %s", err.Error()), make(http.Header), err}
		}
	}

	if err := writer.Close(); err != nil {
		return ResponseWrapper{0, fmt.Sprintf("关闭写入器失败: %s", err.Error()), make(http.Header), err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, httpUrl, body)
	if err != nil {
		return createRequestError(err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	setHeaders(req, headers)
	return doRequest(ctx, req, timeout)
}

// ==================== 核心请求函数 ====================

// doRequest 执行HTTP请求的核心函数（使用全局客户端）
func doRequest(ctx context.Context, req *http.Request, timeout int) ResponseWrapper {
	wrapper := ResponseWrapper{StatusCode: 0, Body: "", Header: make(http.Header)}

	// 如果设置了超时，使用带超时的 context
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// 使用全局客户端执行请求
	client := GetDefaultClient()
	resp, err := client.Do(ctx, req)
	if err != nil {
		wrapper.Body = fmt.Sprintf("执行HTTP请求错误-%s", err.Error())
		wrapper.Error = err
		return wrapper
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		wrapper.Body = fmt.Sprintf("读取HTTP请求返回值失败-%s", err.Error())
		wrapper.Error = err
		return wrapper
	}

	wrapper.StatusCode = resp.StatusCode
	wrapper.Body = string(body)
	wrapper.Header = resp.Header
	return wrapper
}

// setHeaders 设置自定义请求头
func setHeaders(req *http.Request, headers map[string]string) {
	req.Header.Set("User-Agent", "gin-core/http-client")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

// createRequestError 创建请求错误响应
func createRequestError(err error) ResponseWrapper {
	return ResponseWrapper{
		StatusCode: 0,
		Body:       fmt.Sprintf("创建HTTP请求错误-%s", err.Error()),
		Header:     make(http.Header),
		Error:      err,
	}
}

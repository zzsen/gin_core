package http_client

// HTTP客户端工具包
// 提供常用的HTTP请求方法，支持GET、POST、PUT等请求类型
// 支持表单提交、文件上传、JSON请求等功能

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// ResponseWrapper HTTP响应包装器
// 用于统一封装HTTP请求的响应结果
type ResponseWrapper struct {
	StatusCode int         // HTTP状态码
	Body       string      // 响应体内容
	Header     http.Header // 响应头信息
}

// Get 发送GET请求
// 参数:
//   - ctx: Gin上下文对象
//   - url: 请求的URL地址
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func Get(ctx *gin.Context, url string, timeout int, headers map[string]string) ResponseWrapper {
	// 创建GET请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return createRequestError(err)
	}

	// 设置自定义请求头
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	return request(ctx, req, timeout)
}

// PostParams 发送POST请求（参数形式）
// 参数:
//   - ctx: Gin上下文对象
//   - url: 请求的URL地址
//   - params: POST参数（字符串形式）
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PostParams(ctx *gin.Context, url string, params string, timeout int, headers map[string]string) ResponseWrapper {
	// 将参数字符串转换为字节缓冲区
	buf := bytes.NewBufferString(params)
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return createRequestError(err)
	}

	// 设置自定义请求头
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	return request(ctx, req, timeout)
}

// PutParams 发送PUT请求（参数形式）
// 参数:
//   - ctx: Gin上下文对象
//   - url: 请求的URL地址
//   - params: PUT参数（字符串形式）
//   - timeout: 请求超时时间（秒）
//   - headers: 自定义请求头
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PutParams(ctx *gin.Context, url string, params string, timeout int, headers map[string]string) ResponseWrapper {
	// 将参数字符串转换为字节缓冲区
	buf := bytes.NewBufferString(params)
	req, err := http.NewRequest("PUT", url, buf)
	if err != nil {
		return createRequestError(err)
	}

	// 设置Content-Type为JSON格式
	req.Header.Set("Content-Type", "application/json")

	// 设置自定义请求头
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	return request(ctx, req, timeout)
}

// PostJson 发送POST请求（JSON格式）
// 参数:
//   - ctx: Gin上下文对象
//   - url: 请求的URL地址
//   - body: JSON格式的请求体
//   - headers: 自定义请求头
//   - timeout: 请求超时时间（秒）
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PostJson(ctx *gin.Context, url string, body string, headers map[string]string, timeout int) ResponseWrapper {
	// 将JSON字符串转换为字节缓冲区
	buf := bytes.NewBufferString(body)
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return createRequestError(err)
	}

	// 设置自定义请求头
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	// 设置Content-Type为JSON格式
	req.Header.Set("Content-type", "application/json")

	return request(ctx, req, timeout)
}

// PostForm 发送POST请求（表单形式，支持文件上传）
// 参数:
//   - ctx: Gin上下文对象
//   - httpUrl: 请求的URL地址
//   - dataMap: 普通表单字段（键值对）
//   - filePathMap: 文件路径映射（文件名 -> 文件路径）
//   - fileMap: 文件读取器映射（文件名 -> io.Reader）
//   - headers: 自定义请求头
//   - timeout: 请求超时时间（秒）
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func PostForm(ctx *gin.Context, httpUrl string, dataMap map[string]string,
	filePathMap map[string]string, fileMap map[string]io.Reader,
	headers map[string]string, timeout int) ResponseWrapper {

	// 设置TLS调试环境变量，解决某些TLS相关问题
	os.Setenv("GODEBUG", "tlsrsakex=1")

	// 创建一个字节缓冲区用于存储表单数据
	body := &bytes.Buffer{}

	// 创建一个multipart写入器，用于向缓冲区写入表单数据
	writer := multipart.NewWriter(body)

	// 遍历表单数据，将普通表单字段写入
	for key, value := range dataMap {
		err := writer.WriteField(key, value)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("写入表单字段 %s 失败: %s", key, err.Error()), make(http.Header)}
		}
	}

	// 处理文件路径映射（从本地文件路径读取文件）
	for fileName, filePath := range filePathMap {
		// 打开要上传的文件
		file, err := os.Open(filePath)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("打开文件 %s 失败: %s", filePath, err.Error()), make(http.Header)}
		}
		defer file.Close()

		// 创建一个表单文件字段
		part, err := writer.CreateFormFile(fileName, filePath)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("创建表单文件字段失败: %s", err.Error()), make(http.Header)}
		}

		// 将文件内容复制到表单文件字段中
		_, err = io.Copy(part, file)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("复制文件内容失败: %s", err.Error()), make(http.Header)}
		}
	}

	// 处理文件读取器映射（直接使用io.Reader）
	for fileName, fileReader := range fileMap {
		// 创建一个表单文件字段
		part, err := writer.CreateFormFile(fileName, fileName)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("创建表单文件字段失败: %s", err.Error()), make(http.Header)}
		}

		// 将文件内容复制到表单文件字段中
		_, err = io.Copy(part, fileReader)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("复制文件内容失败: %s", err.Error()), make(http.Header)}
		}
	}

	// 关闭写入器，完成表单数据的写入
	err := writer.Close()
	if err != nil {
		return ResponseWrapper{0, fmt.Sprintf("关闭写入器失败: %s", err.Error()), make(http.Header)}
	}

	// 创建一个POST请求
	req, err := http.NewRequest("POST", httpUrl, body)
	if err != nil {
		return ResponseWrapper{0, fmt.Sprintf("创建请求失败: %s", err.Error()), make(http.Header)}
	}

	// 设置请求头的Content-Type为multipart/form-data并包含边界信息
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 添加自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return request(ctx, req, timeout)
}

// request 执行HTTP请求的核心函数
// 参数:
//   - ctx: Gin上下文对象
//   - req: HTTP请求对象
//   - timeout: 请求超时时间（秒）
//
// 返回值: ResponseWrapper 包含响应状态码、响应体和响应头
func request(ctx *gin.Context, req *http.Request, timeout int) ResponseWrapper {
	// 初始化响应包装器
	wrapper := ResponseWrapper{StatusCode: 0, Body: "", Header: make(http.Header)}

	// 创建HTTP客户端
	client := &http.Client{}

	// 设置超时时间
	if timeout > 0 {
		client.Timeout = time.Duration(timeout) * time.Second
	}

	// 设置请求头（包括Trace ID等）
	setRequestHeader(ctx, req)

	// 执行HTTP请求
	resp, err := client.Do(req)
	if err != nil {
		wrapper.Body = fmt.Sprintf("执行HTTP请求错误-%s", err.Error())
		return wrapper
	}
	defer resp.Body.Close()

	// 读取响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		wrapper.Body = fmt.Sprintf("读取HTTP请求返回值失败-%s", err.Error())
		return wrapper
	}

	// 填充响应包装器
	wrapper.StatusCode = resp.StatusCode
	wrapper.Body = string(body)
	wrapper.Header = resp.Header

	return wrapper
}

// setRequestHeader 设置请求头信息
// 参数:
//   - ctx: Gin上下文对象
//   - req: HTTP请求对象
func setRequestHeader(ctx *gin.Context, req *http.Request) {
	// 设置User-Agent
	req.Header.Set("User-Agent", "golang/gocron")

	// 获取Trace ID并设置到请求头中
	traceId, exists := ctx.Get("traceId")
	if exists {
		req.Header.Set("X-Trace-ID", traceId.(string))
	}
}

// createRequestError 创建请求错误响应
// 参数:
//   - err: 错误信息
//
// 返回值: ResponseWrapper 包含错误信息的响应包装器
func createRequestError(err error) ResponseWrapper {
	errorMessage := fmt.Sprintf("创建HTTP请求错误-%s", err.Error())
	return ResponseWrapper{0, errorMessage, make(http.Header)}
}

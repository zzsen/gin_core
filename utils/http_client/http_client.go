package http_client

// http-client

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

type ResponseWrapper struct {
	StatusCode int
	Body       string
	Header     http.Header
}

func Get(ctx *gin.Context, url string, timeout int, headers map[string]string) ResponseWrapper {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return createRequestError(err)
	}
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	return request(ctx, req, timeout)
}

func PostParams(ctx *gin.Context, url string, params string, timeout int, headers map[string]string) ResponseWrapper {
	buf := bytes.NewBufferString(params)
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return createRequestError(err)
	}
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	return request(ctx, req, timeout)
}

func PutParams(ctx *gin.Context, url string, params string, timeout int, headers map[string]string) ResponseWrapper {
	buf := bytes.NewBufferString(params)
	req, err := http.NewRequest("PUT", url, buf)
	if err != nil {
		return createRequestError(err)
	}

	req.Header.Set("Content-Type", "application/json")
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}

	return request(ctx, req, timeout)
}

func PostJson(ctx *gin.Context, url string, body string, headers map[string]string, timeout int) ResponseWrapper {
	buf := bytes.NewBufferString(body)
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return createRequestError(err)
	}
	for headerKey, headerValue := range headers {
		req.Header.Set(headerKey, headerValue)
	}
	req.Header.Set("Content-type", "application/json")

	return request(ctx, req, timeout)
}

func PostForm(ctx *gin.Context, httpUrl string, dataMap map[string]string,
	filePathMap map[string]string, fileMap map[string]io.Reader,
	headers map[string]string, timeout int) ResponseWrapper {
	os.Setenv("GODEBUG", "tlsrsakex=1")
	// 创建一个字节缓冲区用于存储表单数据
	body := &bytes.Buffer{}
	// 创建一个 multipart 写入器，用于向缓冲区写入表单数据
	writer := multipart.NewWriter(body)

	// 遍历表单数据，将普通表单字段写入
	for key, value := range dataMap {
		err := writer.WriteField(key, value)
		if err != nil {
			return ResponseWrapper{0, fmt.Sprintf("写入表单字段 %s 失败: %s", key, err.Error()), make(http.Header)}
		}
	}

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

	// 创建一个 POST 请求
	req, err := http.NewRequest("POST", httpUrl, body)
	if err != nil {
		return ResponseWrapper{0, fmt.Sprintf("创建请求失败: %s", err.Error()), make(http.Header)}
	}

	// 设置请求头的 Content-Type 为 multipart/form-data 并包含边界信息
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 添加自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return request(ctx, req, timeout)
}

func request(ctx *gin.Context, req *http.Request, timeout int) ResponseWrapper {
	wrapper := ResponseWrapper{StatusCode: 0, Body: "", Header: make(http.Header)}
	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = time.Duration(timeout) * time.Second
	}
	setRequestHeader(ctx, req)
	resp, err := client.Do(req)
	if err != nil {
		wrapper.Body = fmt.Sprintf("执行HTTP请求错误-%s", err.Error())
		return wrapper
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		wrapper.Body = fmt.Sprintf("读取HTTP请求返回值失败-%s", err.Error())
		return wrapper
	}
	wrapper.StatusCode = resp.StatusCode
	wrapper.Body = string(body)
	wrapper.Header = resp.Header

	return wrapper
}

func setRequestHeader(ctx *gin.Context, req *http.Request) {
	req.Header.Set("User-Agent", "golang/gocron")
	// 获取 Trace ID
	traceId, exists := ctx.Get("traceId")
	if exists {
		req.Header.Set("X-Trace-ID", traceId.(string))
	}
}

func createRequestError(err error) ResponseWrapper {
	errorMessage := fmt.Sprintf("创建HTTP请求错误-%s", err.Error())
	return ResponseWrapper{0, errorMessage, make(http.Header)}
}

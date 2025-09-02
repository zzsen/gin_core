package file

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// FileMd5 计算指定文件的MD5哈希值
// 参数 filePath: 文件路径
// 返回值: 文件的MD5哈希值（大写字符串），如果文件不存在则返回空字符串
func FileMd5(filePath string) string {
	// 检查文件是否存在，不存在则返回空字符串
	if !PathExists(filePath) {
		return ""
	}

	// 打开文件，忽略错误处理（简化版本）
	file, _ := os.Open(filePath)
	defer file.Close() // 确保文件在函数结束时关闭

	// 创建MD5哈希计算器
	h := md5.New()

	// 将文件内容复制到哈希计算器中
	io.Copy(h, file)

	// 计算最终的MD5哈希值
	md5Code := h.Sum(nil)

	// 将字节数组转换为十六进制字符串并转为大写
	return strings.ToUpper(hex.EncodeToString(md5Code))
}

// PathExists 检查指定路径是否存在
// 参数 path: 要检查的路径（文件或文件夹）
// 返回值: true表示存在，false表示不存在或不确定
func PathExists(path string) bool {
	// 获取文件/文件夹的状态信息
	_, err := os.Stat(path)

	// 当err为nil时，表示文件或文件夹存在
	if err == nil {
		return true
	}

	// 当err为os.ErrNotExist时，表示文件或文件夹不存在
	if os.IsNotExist(err) {
		return false
	}

	// 其他类型的错误（如权限不足等），不确定是否存在
	// 为了安全起见，返回false
	return false
}

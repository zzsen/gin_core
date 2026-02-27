package file

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// FileMd5 计算指定文件的 MD5 哈希值
//
// 参数 filePath: 文件路径
// 返回值: 文件的 MD5 哈希值（大写字符串），如果文件不存在或读取失败则返回空字符串
func FileMd5(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return ""
	}

	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
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

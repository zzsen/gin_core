package file

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

func FileMd5(filePath string) string {
	if !PathExists(filePath) {
		return ""
	}
	file, _ := os.Open(filePath)
	defer file.Close()
	h := md5.New()
	io.Copy(h, file)
	md5Code := h.Sum(nil)
	return strings.ToUpper(hex.EncodeToString(md5Code))
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	//当为空文件或文件夹存在
	if err == nil {
		return true
	}
	//os.IsNotExist(err)为true，文件或文件夹不存在
	if os.IsNotExist(err) {
		return false
	}
	//其它类型，不确定是否存在
	return false
}

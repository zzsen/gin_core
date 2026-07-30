package configcenter

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/utils/encrypt"
)

// prepareYAML 对 Etcd 拉取的 YAML 做与本地 loadYamlConfig 同源的 ENV / CIPHER 处理。
//
// 【流程】
// 1. replaceWithEnv：替换 {{ENV_KEY}}
// 2. decryptConfig：解密 CIPHER(...)
func prepareYAML(raw []byte, cipherKey string) ([]byte, error) {
	// 步骤1：环境变量占位
	data, err := replaceWithEnv(raw)
	if err != nil {
		return nil, err
	}
	// 步骤2：CIPHER 解密
	return decryptConfig(data, cipherKey)
}

// replaceWithEnv 将 YAML 中的 {{ENV}} 占位符替换为环境变量值；缺失变量返回错误。
func replaceWithEnv(yamlData []byte) ([]byte, error) {
	yamlStr := string(yamlData)
	re := regexp.MustCompile(`\{\{.*?\}\}`)
	placeholders := re.FindAllString(yamlStr, -1)
	if len(placeholders) == 0 {
		return yamlData, nil
	}
	for _, ph := range placeholders {
		if len(ph) < 4 {
			return nil, errors.New("无效占位符:" + ph)
		}
		envKey := ph[2 : len(ph)-2]
		val, ok := os.LookupEnv(envKey)
		if !ok {
			return nil, errors.New("缺失环境变量:" + envKey)
		}
		yamlStr = strings.ReplaceAll(yamlStr, ph, val)
	}
	return []byte(yamlStr), nil
}

// decryptConfig 解密 YAML 中的 CIPHER(密文) 片段。
//
// cipherKey 为空且存在 CIPHER 时打 error 日志并原样返回（与本地配置加载策略对齐，不硬失败）。
func decryptConfig(yamlData []byte, cipherKey string) ([]byte, error) {
	yamlStr := string(yamlData)
	re := regexp.MustCompile(`CIPHER\((.*?)\)`)
	matches := re.FindAllStringSubmatch(yamlStr, -1)
	if len(matches) == 0 {
		return yamlData, nil
	}
	if cipherKey == "" {
		logger.Error("[configcenter] 配置中含加密内容, 但未提供 cipherKey")
		return yamlData, nil
	}
	for _, m := range matches {
		if len(m) != 2 {
			return nil, errors.New("无效占位符:" + m[0])
		}
		plain, err := encrypt.AesEcbDecrypt(m[1], cipherKey)
		if err != nil {
			return nil, err
		}
		yamlStr = strings.Replace(yamlStr, m[0], plain, 1)
	}
	return []byte(yamlStr), nil
}

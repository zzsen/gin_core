package core

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/utils/encrypt"
	"gopkg.in/yaml.v3"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
	fileUtil "github.com/zzsen/gin_core/utils/file"
)

// InitCustomConfig 初始化自定义配置
// 将用户自定义的配置结构体存储到全局变量中
// 参数 conf: 用户自定义的配置结构体指针
func InitCustomConfig(conf any) {
	app.Config = conf
}

// loadConfig 加载配置文件
// 这是配置加载的主入口函数，负责：
// 1. 解析命令行参数
// 2. 确定运行环境
// 3. 加载默认配置文件
// 4. 加载环境特定的配置文件
// 5. 设置Gin运行模式
// 参数 conf: 用户自定义的配置结构体指针
func loadConfig(conf any) {
	// 使用defer+recover确保配置加载失败时程序能优雅退出
	defer func() {
		if err := recover(); err != nil {
			logger.Error("%v", err)
			os.Exit(1)
		}
	}()

	// 解析命令行参数获取环境和配置路径
	cmdArgs, err := parseCmdArgs()

	// 如果命令行未指定环境，尝试从env文件读取
	if cmdArgs.Env == "" {
		env, err := getEnvFromFile()
		if err != nil {
			logger.Info("[配置解析] 获取环境变量失败, %s", err.Error())
			cmdArgs.Env = constant.DefaultEnv
		} else if env == "" {
			cmdArgs.Env = constant.DefaultEnv
			logger.Info("[配置解析] env文件内容无效, 当前运行环境: %s", cmdArgs.Env)
		} else {
			cmdArgs.Env = env
			logger.Info("[配置解析] 从env文件中加载运行环境, 当前运行环境: %s", cmdArgs.Env)
		}
	}

	// 如果是生产环境，设置Gin为Release模式以提高性能
	if cmdArgs.Env == constant.ProdEnv {
		gin.SetMode(gin.ReleaseMode)
	}

	// 检查命令行参数解析是否有错误
	if err != nil {
		logger.Error("[配置解析] 解析启动参数失败, %s, %s", getDateTime(), err.Error())
		os.Exit(1)
	}

	// 构建默认配置文件路径并加载
	defaultConfigFilePath := path.Join(cmdArgs.Config, constant.DefaultConfigFileName)
	if fileUtil.PathExists(defaultConfigFilePath) {
		// 如果有默认文件，则先加载默认配置文件
		// 默认配置提供基础配置，后续的环境配置会覆盖相同的配置项
		err = loadYamlConfig(defaultConfigFilePath, conf, cmdArgs.CipherKey)
		if err != nil {
			logger.Error("[配置解析] 加载默认配置失败: %s", err.Error())
			os.Exit(1)
		}
	}

	// 如果不是默认环境，加载环境特定的配置文件
	if cmdArgs.Env != constant.DefaultEnv {
		customConfigFileName := fmt.Sprintf("%s%s%s", constant.CustomConfigFileNamePrefix, cmdArgs.Env, constant.CustomConfigFileNameSuffix)
		customConfigFilePath := path.Join(cmdArgs.Config, customConfigFileName)
		if !fileUtil.PathExists(customConfigFilePath) {
			// 如果没有自定义配置文件，程序无法继续运行
			logger.Error("[配置解析] 加载自定义配置失败, 配置文件目录%s下, 不存在自定义配置文件%s", cmdArgs.Config, customConfigFilePath)
			os.Exit(1)
		}

		// 加载环境特定配置，会覆盖默认配置中的相同配置项
		err = loadYamlConfig(customConfigFilePath, conf, cmdArgs.CipherKey)
		if err != nil {
			logger.Error("[配置解析] 加载自定义配置%s失败: %s", customConfigFileName, err.Error())
			os.Exit(1)
		}
	}
	// 将确定的环境保存到全局变量
	app.Env = cmdArgs.Env
}

// getEnvFromFile 从env文件中获取环境变量
// 当命令行参数中未指定环境时，尝试从项目根目录的"env"文件中读取
// 文件格式：第一行为环境名称，支持字母、数字和下划线
// 返回值：环境名称字符串和可能的错误
func getEnvFromFile() (string, error) {
	// 命令行中未指定参数时, 先判断环境文件是否存在, 若存在则读取文件首行内容

	envFileName := "env"
	// 检查文件是否存在
	_, err := os.Stat(envFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("[getEnvFromFile] 环境文件不存在: %s", envFileName)
		}
		return "", fmt.Errorf("[getEnvFromFile] 检查环境文件时发生错误: %s", err.Error())
	}

	// 打开文件
	file, err := os.Open(envFileName)
	if err != nil {
		return "", fmt.Errorf("[getEnvFromFile] 打开环境文件时发生错误: %s", err.Error())
	}
	// 确保文件在函数结束时关闭，防止文件句柄泄露
	defer file.Close()

	// 创建一个扫描器来逐行读取文件
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		// 获取首行内容
		firstLine := scanner.Text()

		// 定义正则表达式，匹配字母、数字和下划线
		// 这确保环境名称符合标识符规范
		regex := regexp.MustCompile(`[a-zA-Z0-9_]+`)
		// 查找首个匹配项
		match := regex.FindString(firstLine)

		if match != "" {
			// 返回匹配到的内容
			return match, nil
		}
		return "", fmt.Errorf("[getEnvFromFile] 环境文件首行内容无效: %s", firstLine)
	}

	// 检查扫描过程中是否有错误
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("[getEnvFromFile] 扫描环境文件时发生错误: %s", err.Error())
	}

	return "", fmt.Errorf("[getEnvFromFile] 环境文件为空")
}

// getDateTime 获取当前时间的格式化字符串
// 返回格式：YYYY-MM-DD HH:MM:SS
// 主要用于日志记录和错误信息的时间戳
func getDateTime() string {
	return time.Unix(0, time.Now().UnixNano()).Format("2006-01-02 15:04:05")
}

// loadYamlConfig 加载YAML配置文件
// 这是配置文件处理的核心函数，支持：
// 1. 配置类型验证
// 2. 环境变量替换
// 3. 加密配置解密
// 4. 同时加载到基础配置和自定义配置
// 参数：
//   - path: 配置文件路径
//   - conf: 自定义配置结构体指针
//   - CipherKey: 解密密钥，用于解密配置中的敏感信息
func loadYamlConfig(path string, conf any, CipherKey string) error {
	// 验证配置结构体类型是否正确
	err := checkConfType(conf)
	if err != nil {
		return err
	}

	// 读取YAML文件内容
	fileData, err := loadYamlFile(path)
	if err != nil {
		return err
	}

	// 替换配置文件中的环境变量占位符
	// 支持 {{ENV_VAR_NAME}} 格式的占位符
	fileData, err = replaceWithEvn(fileData)
	if err != nil {
		return err
	}

	// 解密配置文件中的加密内容
	// 支持 CIPHER(encrypted_content) 格式的加密配置
	fileData, err = decryptConfig(fileData, CipherKey)
	if err != nil {
		return err
	}

	// 先将配置加载到基础配置结构体
	// 基础配置包含框架所需的所有配置项
	err = yaml.Unmarshal(fileData, &app.BaseConfig)
	if err != nil {
		logger.Error("[配置解析] 加载基础配置%s失败: %s", path, err.Error())
		return err
	}

	// 再将配置加载到用户自定义配置结构体
	// 用户配置可能包含业务特定的配置项
	err = yaml.Unmarshal(fileData, conf)
	return err
}

// checkConfType 检查配置结构体类型
// 确保传入的配置对象是结构体指针类型
// 这是为了能够正确地将YAML内容反序列化到配置对象中
// 参数 conf: 待检查的配置对象
// 返回值: 如果类型不正确则返回错误
func checkConfType(conf any) error {
	// 检查是否为指针类型
	if reflect.TypeOf(conf).Kind().String() != "ptr" {
		return errors.New("conf type is not ptr")
	}
	// 检查指针指向的类型是否为结构体
	if reflect.TypeOf(conf).Elem().Kind().String() != "struct" {
		return errors.New("*conf type is not struct")
	}
	return nil
}

// loadYamlFile 读取YAML文件内容
// 从指定路径读取文件并返回字节数组
// 参数 path: 文件路径
// 返回值: 文件内容字节数组和可能的错误
func loadYamlFile(path string) ([]byte, error) {
	// 首先检查文件是否存在
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// 确保文件正确关闭
	defer file.Close()

	// 读取文件全部内容
	var data []byte
	data, err = io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, err
}

// replaceWithEvn 使用环境变量的值替换YAML文件中的占位符
// 支持 {{ENV_VAR_NAME}} 格式的占位符替换
// 这允许在配置文件中使用环境变量，提高配置的灵活性
// 参数 yamlData: 原始YAML内容
// 返回值: 替换后的YAML内容和可能的错误
func replaceWithEvn(yamlData []byte) ([]byte, error) {
	yamlStr := string(yamlData)
	// 定义占位符的正则表达式：{{任意内容}}
	placeholderExpr := "{{.*?}}"
	regexpObj, _ := regexp.Compile(placeholderExpr)
	// 查找所有占位符
	placeholderList := regexpObj.FindAllString(yamlStr, -1)

	if len(placeholderList) > 0 {
		// 加载环境变量值
		evnData, err := loadEvnValue(placeholderList)
		if err != nil {
			return nil, err
		}
		// 逐个替换占位符
		for key, value := range evnData {
			yamlStr = strings.Replace(yamlStr, key, value, -1)
		}
		return []byte(yamlStr), nil
	}

	// 如果没有占位符，直接返回原内容
	return yamlData, nil
}

// loadEvnValue 从环境变量中加载占位符对应的值
// 解析占位符并从系统环境变量中获取对应的值
// 参数 keys: 占位符列表，格式为 {{ENV_VAR_NAME}}
// 返回值: 占位符到环境变量值的映射和可能的错误
func loadEvnValue(keys []string) (map[string]string, error) {
	valueMap := make(map[string]string)
	for _, key := range keys {
		// 检查占位符格式是否正确（至少需要4个字符：{{}}）
		if len(key) < 4 {
			return nil, errors.New("无效占位符:" + key)
		}
		// 提取环境变量名（去掉前后的{{}}）
		evnKey := key[2 : len(key)-2]
		// 从系统环境变量中查找
		data, exist := os.LookupEnv(evnKey)
		if !exist {
			return nil, errors.New("缺失环境变量:" + evnKey)
		}
		valueMap[key] = data
	}
	return valueMap, nil
}

// decryptConfig 解密配置文件中的加密内容
// 支持 CIPHER(encrypted_content) 格式的加密配置解密
// 这允许在配置文件中存储敏感信息（如密码、密钥等）
// 参数：
//   - yamlData: 原始YAML内容
//   - CipherKey: 解密密钥
//
// 返回值: 解密后的YAML内容和可能的错误
func decryptConfig(yamlData []byte, CipherKey string) ([]byte, error) {
	yamlStr := string(yamlData)
	// 定义加密内容的正则表达式：CIPHER(加密内容)
	placeholderExpr := `CIPHER\((.*?)\)`
	regexpObj := regexp.MustCompile(placeholderExpr)
	// 查找所有加密占位符，返回完整匹配和分组匹配
	placeholderList := regexpObj.FindAllStringSubmatch(yamlStr, -1)

	// 如果没有加密内容，直接返回
	if len(placeholderList) == 0 {
		return yamlData, nil
	}

	// 如果有加密内容但没有提供解密密钥
	if CipherKey == "" {
		// 仅输出警告日志，不中断服务
		// 这样可以避免在某些环境下确实不需要解密时导致的服务启动失败
		logger.Error("[配置解析] 配置中含加密内容, 但服务启动指令中不含解密key, 请检查配置或启动指令")
		return yamlData, nil
	}

	// 逐个解密加密内容
	for _, placeholder := range placeholderList {
		// placeholder[0] 是完整匹配 CIPHER(...)
		// placeholder[1] 是括号内的加密内容
		if len(placeholder) != 2 {
			return nil, errors.New("无效占位符:" + placeholder[0])
		}
		// 使用AES ECB模式解密
		data, err := encrypt.AesEcbDecrypt(placeholder[1], CipherKey)
		if err != nil {
			return nil, err
		}

		// 将加密占位符替换为解密后的明文
		yamlStr = strings.Replace(yamlStr, placeholder[0], data, -1)
	}

	return []byte(yamlStr), nil
}

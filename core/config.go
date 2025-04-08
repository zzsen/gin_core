package core

import (
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

func InitCustomConfig(conf any) {
	app.Config = conf
}

func loadConfig(conf any) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("%v", err)
			os.Exit(1)
		}
	}()

	cmdArgs, err := parseCmdArgs()

	if cmdArgs.Env == constant.ProdEnv {
		gin.SetMode(gin.ReleaseMode)
	}

	if err != nil {
		logger.Error("[配置解析] 解析启动参数失败, %s, %s", getDateTime(), err.Error())
		os.Exit(1)
	}

	defaultConfigFilePath := path.Join(cmdArgs.Config, constant.DefaultConfigFileName)
	if fileUtil.PathExists(defaultConfigFilePath) {
		//如果有默认文件，则先加载默认配置文件
		err = loadYamlConfig(defaultConfigFilePath, conf, cmdArgs.CipherKey)
		if err != nil {
			logger.Error("[配置解析] 加载默认配置失败: %s", err.Error())
			os.Exit(1)
		}
	}

	// 如果有自定义配置文件，则加载自定义配置文件
	if cmdArgs.Env != constant.DefaultEnv {
		customConfigFileName := fmt.Sprintf("%s%s%s", constant.CustomConfigFileNamePrefix, cmdArgs.Env, constant.CustomConfigFileNameSuffix)
		customConfigFilePath := path.Join(cmdArgs.Config, customConfigFileName)
		if !fileUtil.PathExists(customConfigFilePath) {
			// 如果没有自定义配置文件，则直接返回
			logger.Error("[配置解析] 加载自定义配置失败, 配置文件目录%s下, 不存在自定义配置文件%s", cmdArgs.Config, customConfigFilePath)
			os.Exit(1)
		}

		err = loadYamlConfig(customConfigFilePath, conf, cmdArgs.CipherKey)
		if err != nil {
			logger.Error("[配置解析] 加载自定义配置%s失败: %s", customConfigFileName, err.Error())
			os.Exit(1)
		}
	}
	app.Env = cmdArgs.Env
}

func getDateTime() string {
	return time.Unix(0, time.Now().UnixNano()).Format("2006-01-02 15:04:05")
}

func loadYamlConfig(path string, conf any, CipherKey string) error {
	err := checkConfType(conf)
	if err != nil {
		return err
	}

	fileData, err := loadYamlFile(path)
	if err != nil {
		return err
	}

	fileData, err = replaceWithEvn(fileData)
	if err != nil {
		return err
	}

	fileData, err = decryptConfig(fileData, CipherKey)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(fileData, &app.BaseConfig)
	if err != nil {
		logger.Error("[配置解析] 加载基础配置%s失败: %s", path, err.Error())
		return err
	}

	err = yaml.Unmarshal(fileData, conf)
	return err
}

func checkConfType(conf any) error {
	if reflect.TypeOf(conf).Kind().String() != "ptr" {
		return errors.New("conf type is not ptr")
	}
	if reflect.TypeOf(conf).Elem().Kind().String() != "struct" {
		return errors.New("*conf type is not struct")
	}
	return nil
}

func loadYamlFile(path string) ([]byte, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var data []byte
	data, err = io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return data, err
}

// 使用环境变量的值替换yaml文件中的占位符
func replaceWithEvn(yamlData []byte) ([]byte, error) {
	yamlStr := string(yamlData)
	placeholderExpr := "{{.*?}}"
	regexpObj, _ := regexp.Compile(placeholderExpr)
	placeholderList := regexpObj.FindAllString(yamlStr, -1)
	if len(placeholderList) > 0 {
		evnData, err := loadEvnValue(placeholderList)
		if err != nil {
			return nil, err
		}
		for key, value := range evnData {
			yamlStr = strings.Replace(yamlStr, key, value, -1)
		}
		return []byte(yamlStr), nil
	}

	return yamlData, nil
}

func loadEvnValue(keys []string) (map[string]string, error) {
	valueMap := make(map[string]string)
	for _, key := range keys {
		if len(key) < 4 {
			return nil, errors.New("无效占位符:" + key)
		}
		evnKey := key[2 : len(key)-2]
		data, exist := os.LookupEnv(evnKey)
		if !exist {
			return nil, errors.New("缺失环境变量:" + evnKey)
		}
		valueMap[key] = data
	}
	return valueMap, nil
}

// 使用环境变量的值替换yaml文件中的占位符
func decryptConfig(yamlData []byte, CipherKey string) ([]byte, error) {
	yamlStr := string(yamlData)
	placeholderExpr := `CIPHER\((.*?)\)`
	regexpObj := regexp.MustCompile(placeholderExpr)
	placeholderList := regexpObj.FindAllStringSubmatch(yamlStr, -1)
	if len(placeholderList) == 0 {
		return yamlData, nil
	}

	if CipherKey == "" {
		// 仅输出log, 不中断服务, 避免该加密内容确实不需要解密
		logger.Error("[配置解析] 配置中含加密内容, 但服务启动指令中不含解密key, 请检查配置或启动指令")
		return yamlData, nil
	}

	for _, placeholder := range placeholderList {
		if len(placeholder) != 2 {
			return nil, errors.New("无效占位符:" + placeholder[0])
		}
		data, err := encrypt.AesEcbDecrypt(placeholder[1], CipherKey)
		if err != nil {
			return nil, err
		}

		yamlStr = strings.Replace(yamlStr, placeholder[0], data, -1)
	}

	return []byte(yamlStr), nil
}

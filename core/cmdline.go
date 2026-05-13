package core

import (
	"flag"
	"os"

	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
)

// CmdArgs 命令行参数结构体
// 存储从命令行解析的运行时参数，用于控制应用的运行环境和配置加载方式
type CmdArgs struct {
	Env       string // 运行环境标识，如 dev、test、prod
	Config    string // 配置文件目录路径
	CipherKey string // 配置文件加密密钥，用于解密 CIPHER() 包裹的加密值
}

// parseCmdArgs 解析命令行参数
//
// 支持的参数：
//   - -env: 运行环境（dev/test/prod），默认空（走默认配置文件）
//   - -config: 配置文件目录，默认 ./conf
//   - -cipherKey: 加密密钥，用于解密配置文件中的敏感字段
//
// 返回：
//   - *CmdArgs: 解析后的命令行参数
//   - error: 解析错误
func parseCmdArgs() (*CmdArgs, error) {
	info := CmdArgs{}
	argv := flag.NewFlagSet(os.Args[0], flag.PanicOnError)
	argv.StringVar(&info.Env, "env", "", "运行环境，dev, test, prod等， 默认dev")
	argv.StringVar(&info.Config, "config", constant.DefaultConfigDirPath, "配置文件路径，默认./conf")
	argv.StringVar(&info.CipherKey, "cipherKey", "", "加密key, 配置文件加密时使用")
	if !argv.Parsed() {
		_ = argv.Parse(os.Args[1:])
	}

	logger.Info("[配置解析] 解析参数完成, 运行环境:%s, 配置文件路径: %s", info.Env, info.Config)
	return &info, nil
}

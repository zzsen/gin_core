package core

import (
	"flag"
	"os"

	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
)

type CmdArgs struct {
	Env       string
	Config    string
	CipherKey string
}

func ParseCmdArgs() (*CmdArgs, error) {
	info := CmdArgs{}
	argv := flag.NewFlagSet(os.Args[0], 2)
	argv.StringVar(&info.Env, "env", constant.DefaultEnv, "运行环境，dev, test, prod等， 默认dev")
	argv.StringVar(&info.Config, "config", constant.DefaultConfigDirPath, "配置文件路径，默认./conf")
	argv.StringVar(&info.CipherKey, "cipherKey", "", "加密key, 配置文件加密时使用")
	if !argv.Parsed() {
		_ = argv.Parse(os.Args[1:])
	}

	logger.Info("解析参数完成, 运行环境:%s, 配置文件路径: %s", info.Env, info.Config)
	return &info, nil
}

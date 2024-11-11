package core

import (
	"flag"
	"os"

	"github.com/zzsen/github.com/zzsen/gin_core/constant"
	"github.com/zzsen/github.com/zzsen/gin_core/logging"
)

type CmdArgs struct {
	Env    string
	Config string
}

func ParseCmdArgs() (*CmdArgs, error) {
	info := CmdArgs{}
	argv := flag.NewFlagSet(os.Args[0], 2)
	argv.StringVar(&info.Env, "env", constant.DefaultEnv, "运行环境，dev, test, prod等， 默认dev")
	argv.StringVar(&info.Config, "config", constant.DefaultConfigDirPath, "配置文件路径，默认./conf")
	if !argv.Parsed() {
		_ = argv.Parse(os.Args[1:])
	}

	logging.Info("解析参数完成, 运行环境:%s, 配置文件路径: %s", info.Env, info.Config)
	return &info, nil
}

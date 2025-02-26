package constant

// 默认启动环境
const DefaultEnv = "default"

// 正式环境
const ProdEnv = "prod"

// 默认配置文件夹路径
const DefaultConfigDirPath = "./conf"

// 默认配置文件名称
const DefaultConfigFileName = "config.default.yml"

// 自定义配置文件名称前缀
const CustomConfigFileNamePrefix = "config."

// 自定义配置文件名称后缀
const CustomConfigFileNameSuffix = ".yml"

type LogPrintFileEnum int

const (
	// 不打印文件信息
	NoPrintFile LogPrintFileEnum = iota
	// 打印相对文件路径
	PrintRelativeFile
	// 打印绝对文件路径
	PrintAbsoluteFile
)

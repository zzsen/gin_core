package config

import "fmt"

type DbInfo struct {
	AliasName     string   `yaml:"aliasName"`     // 数据库别名
	Host          string   `yaml:"host"`          // 数据库地址
	Port          int      `yaml:"port"`          // 数据库端口
	DBName        string   `yaml:"dbName"`        // 数据库名
	Username      string   `yaml:"username"`      // 数据库账号
	Password      string   `yaml:"password"`      // 数据库密码
	Loc           string   `yaml:"loc"`           // 时区
	MaxIdleConns  int      `yaml:"maxIdleConns"`  // 空闲中的最大连接数
	MaxOpenConns  int      `yaml:"maxOpenConns"`  // 打开到数据库的最大连接数
	Migrate       string   `yaml:"migrate"`       // 每次启动时更新数据库表的方式 update:增量更新表，create:删除所有表再重新建表, 其他则不执行任何动作
	EnableLog     bool     `yaml:"enableLog"`     // 是否开启日志
	SlowThreshold int      `yaml:"slowThreshold"` // 慢查询阈值
	Tables        []string `yaml:"tables"`        // 走该库查询的数据表
	TablePrefix   string   `yaml:"tablePrefix"`   // 表名前缀
}

func (dbInfo *DbInfo) Dsn() string {
	if dbInfo.Loc == "" {
		dbInfo.Loc = "Local"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=%s",
		dbInfo.Username,
		dbInfo.Password,
		dbInfo.Host,
		dbInfo.Port,
		dbInfo.DBName,
		dbInfo.Loc)
}

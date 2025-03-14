package config

import "fmt"

type DbInfo struct {
	AliasName                 string   `yaml:"aliasName"`                 // 数据库别名
	Host                      string   `yaml:"host"`                      // 数据库地址
	Port                      int      `yaml:"port"`                      // 数据库端口
	DBName                    string   `yaml:"dbName"`                    // 数据库名
	Username                  string   `yaml:"username"`                  // 数据库账号
	Password                  string   `yaml:"password"`                  // 数据库密码
	Charset                   string   `yaml:"charset"`                   // 数据库编码
	Loc                       string   `yaml:"loc"`                       // 时区
	MaxIdleConns              int      `yaml:"maxIdleConns"`              // 空闲中的最大连接数. 用于设置连接池中允许保持空闲状态的最大连接数。当连接池中的空闲连接数量超过这个值时, 多余的空闲连接会被关闭。
	MaxOpenConns              int      `yaml:"maxOpenConns"`              // 打开到数据库的最大连接数. 用于设置连接池中允许同时打开的最大连接数。当打开的连接数量达到这个值时, 新的连接请求会被阻塞, 直到有连接被释放
	ConnMaxIdleTime           int      `yaml:"connMaxIdleTime"`           // # 最大空闲时间, 单位: 秒. 用于设置连接在连接池中保持空闲状态的最大时间。当一个空闲连接的存活时间超过这个值时, 该连接会被关闭并从连接池中移除
	ConnMaxLifetime           int      `yaml:"connMaxLifetime"`           // # 最大连接存活时间, 单位: 秒. 用于设置连接在连接池中可以存活的最大时间。当一个连接的存活时间超过这个值时, 无论该连接是否处于空闲状态, 都会被关闭并从连接池中移除
	Migrate                   string   `yaml:"migrate"`                   // 每次启动时更新数据库表的方式 update:增量更新表, create:删除所有表再重新建表, 其他则不执行任何动作
	LogLevel                  *int     `yaml:"logLevel"`                  // 日志级别（1-关闭所有日志, 2-仅输出错误日志, 3-输出错误日志和慢查询, 4-输出错误日志和慢查询日志和所有sql）
	SlowThreshold             *int     `yaml:"slowThreshold"`             // 慢查询阈值（单位：毫秒）
	IgnoreRecordNotFoundError *bool    `yaml:"ignoreRecordNotFoundError"` // 忽略记录未找到错误
	Tables                    []string `yaml:"tables"`                    // 走该库查询的数据表
	TablePrefix               string   `yaml:"tablePrefix"`               // 表名前缀
}

func (dbInfo *DbInfo) Dsn() string {
	if dbInfo.Loc == "" {
		dbInfo.Loc = "Local"
	}
	if dbInfo.Charset == "" {
		dbInfo.Charset = "utf8mb4"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=%s",
		dbInfo.Username,
		dbInfo.Password,
		dbInfo.Host,
		dbInfo.Port,
		dbInfo.DBName,
		dbInfo.Charset,
		dbInfo.Loc)
}

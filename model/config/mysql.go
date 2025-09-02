// Package config 提供应用程序的配置结构定义
// 本文件定义了MySQL数据库的配置结构，包含连接参数、连接池配置和数据库迁移策略
package config

import "fmt"

// DbInfo MySQL数据库配置信息
// 该结构体包含了连接MySQL数据库所需的所有配置参数，支持连接池和迁移策略配置
type DbInfo struct {
	AliasName                 string   `yaml:"aliasName"`                 // 数据库别名，用于多数据库环境下的标识
	Host                      string   `yaml:"host"`                      // 数据库服务器地址，支持IP地址或域名
	Port                      int      `yaml:"port"`                      // 数据库服务器端口，MySQL默认端口为3306
	DBName                    string   `yaml:"dbName"`                    // 数据库名称，指定要连接的数据库
	Username                  string   `yaml:"username"`                  // 数据库访问用户名，用于身份认证
	Password                  string   `yaml:"password"`                  // 数据库访问密码，用于身份认证
	Charset                   string   `yaml:"charset"`                   // 数据库字符集，用于确保数据编码正确
	Loc                       string   `yaml:"loc"`                       // 数据库时区设置，影响时间字段的处理
	MaxIdleConns              int      `yaml:"maxIdleConns"`              // 空闲中的最大连接数，用于设置连接池中允许保持空闲状态的最大连接数。当连接池中的空闲连接数量超过这个值时，多余的空闲连接会被关闭。
	MaxOpenConns              int      `yaml:"maxOpenConns"`              // 打开到数据库的最大连接数，用于设置连接池中允许同时打开的最大连接数。当打开的连接数量达到这个值时，新的连接请求会被阻塞，直到有连接被释放
	ConnMaxIdleTime           int      `yaml:"connMaxIdleTime"`           // 最大空闲时间，单位：秒，用于设置连接在连接池中保持空闲状态的最大时间。当一个空闲连接的存活时间超过这个值时，该连接会被关闭并从连接池中移除
	ConnMaxLifetime           int      `yaml:"connMaxLifetime"`           // 最大连接存活时间，单位：秒，用于设置连接在连接池中可以存活的最大时间。当一个连接的存活时间超过这个值时，无论该连接是否处于空闲状态，都会被关闭并从连接池中移除
	Migrate                   string   `yaml:"migrate"`                   // 每次启动时更新数据库表的方式，update:增量更新表，create:删除所有表再重新建表，其他则不执行任何动作
	LogLevel                  *int     `yaml:"logLevel"`                  // 日志级别（1-关闭所有日志，2-仅输出错误日志，3-输出错误日志和慢查询，4-输出错误日志和慢查询日志和所有sql）
	SlowThreshold             *int     `yaml:"slowThreshold"`             // 慢查询阈值（单位：毫秒），超过此时间的SQL查询会被记录为慢查询
	IgnoreRecordNotFoundError *bool    `yaml:"ignoreRecordNotFoundError"` // 忽略记录未找到错误，当查询结果为空时是否记录错误日志
	Tables                    []string `yaml:"tables"`                    // 走该库查询的数据表，用于分库分表场景下的表路由
	TablePrefix               string   `yaml:"tablePrefix"`               // 表名前缀，所有表名都会自动添加此前缀
}

// Dsn 生成数据库连接字符串
// 该方法根据配置参数生成标准的MySQL DSN（Data Source Name）连接字符串
// 返回：
//   - string: MySQL数据库连接字符串
func (dbInfo *DbInfo) Dsn() string {
	// 设置默认时区为Local，如果配置中未指定
	if dbInfo.Loc == "" {
		dbInfo.Loc = "Local"
	}
	// 设置默认字符集为utf8mb4，如果配置中未指定
	if dbInfo.Charset == "" {
		dbInfo.Charset = "utf8mb4"
	}

	// 生成MySQL DSN连接字符串，包含所有必要的连接参数
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=%s",
		dbInfo.Username, // 用户名
		dbInfo.Password, // 密码
		dbInfo.Host,     // 主机地址
		dbInfo.Port,     // 端口号
		dbInfo.DBName,   // 数据库名
		dbInfo.Charset,  // 字符集
		dbInfo.Loc)      // 时区
}

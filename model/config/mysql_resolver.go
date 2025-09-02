// Package config 提供应用程序的配置结构定义
// 本文件定义了MySQL数据库解析器的配置结构，支持读写分离和分库分表
package config

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DbResolver 数据库解析器配置
// 该结构体定义了读写分离的配置，支持主从数据库配置和表级别路由
type DbResolver struct {
	Sources  []DbInfo `yaml:"sources"`  // 主库配置列表，用于写操作（INSERT、UPDATE、DELETE）
	Replicas []DbInfo `yaml:"replicas"` // 从库配置列表，用于读操作（SELECT）
	Tables   []any    `yaml:"tables"`   // 该解析器适用的表名列表，支持字符串或结构体类型
}

// IsValid 验证数据库解析器配置是否有效
// 该方法检查是否至少配置了一个主库，确保解析器可以正常工作
// 返回：
//   - bool: 配置是否有效
func (dbResolver *DbResolver) IsValid() bool {
	return len(dbResolver.Sources) > 0
}

// SourceConfigs 获取主库配置列表
// 该方法将主库配置转换为GORM的数据库驱动配置，用于建立写操作连接
// 返回：
//   - []gorm.Dialector: 主库数据库驱动配置列表
func (dbResolver *DbResolver) SourceConfigs() []gorm.Dialector {
	sourceConfigs := []gorm.Dialector{}
	for _, source := range dbResolver.Sources {
		sourceConfigs = append(sourceConfigs, mysql.Open(source.Dsn()))
	}
	return sourceConfigs
}

// ReplicaConfigs 获取从库配置列表
// 该方法将从库配置转换为GORM的数据库驱动配置，用于建立读操作连接
// 返回：
//   - []gorm.Dialector: 从库数据库驱动配置列表
func (dbResolver *DbResolver) ReplicaConfigs() []gorm.Dialector {
	replicaConfigs := []gorm.Dialector{}
	for _, replica := range dbResolver.Replicas {
		replicaConfigs = append(replicaConfigs, mysql.Open(replica.Dsn()))
	}
	return replicaConfigs
}

// DbResolvers 数据库解析器配置列表
// 该类型定义了多个数据库解析器的集合，支持复杂的数据库路由策略
type DbResolvers []DbResolver

// IsValid 验证所有数据库解析器配置是否有效
// 该方法遍历所有解析器配置，确保每个解析器都配置正确
// 返回：
//   - bool: 所有配置是否都有效
func (dbResolvers DbResolvers) IsValid() bool {
	for _, resolver := range dbResolvers {
		if !resolver.IsValid() {
			return false
		}
	}
	return true
}

// DefaultConfig 获取默认数据库配置
// 该方法返回第一个解析器的第一个主库配置，作为默认的数据库连接配置
// 返回：
//   - DbInfo: 默认数据库配置信息
func (dbResolvers DbResolvers) DefaultConfig() DbInfo {
	return dbResolvers[0].Sources[0]
}

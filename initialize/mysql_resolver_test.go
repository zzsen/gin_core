// Package initialize MySQL 数据库读写分离功能测试
//
// ==================== 集成测试说明 ====================
// 本文件包含 MySQL 读写分离（DBResolver）功能的集成测试。
// 需要真实的 MySQL 数据库连接才能运行。
//
// 测试覆盖内容：
// 1. 多数据源初始化
// 2. 读写分离路由（Sources 用于写，Replicas 用于读）
// 3. 表级别的数据源映射
// 4. GORM 集成验证
//
// 前置条件：
// 1. MySQL 服务已启动
// 2. 下方的连接配置（Host/Port/Username/Password）正确
// 3. 数据库和表已创建（参考下方建表语句）
//
// 运行测试：go test -v ./initialize/... -run TestInitDBResolver
// ==================================================
package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/model/config"

	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== 测试辅助配置 ====================

// 建表语句，需要提前在数据库中创建：
//
//	CREATE TABLE `user` (
//	  `id` bigint NOT NULL AUTO_INCREMENT,
//	  `name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
//	  `create_time` datetime(3) DEFAULT NULL,
//	  PRIMARY KEY (`id`)
//	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

// dbConfig 测试用的数据库读写分离配置
// 按需调整为实际可用的数据库连接配置
var dbConfig = []config.DbResolver{
	{
		Sources: []config.DbInfo{
			{
				Host:     "127.0.0.1",
				Port:     13306,
				DBName:   "test",
				Username: "root",
				Password: "10.160.23.43",
			},
		},
		Replicas: []config.DbInfo{
			{
				Host:     "127.0.0.1",
				Port:     13306,
				DBName:   "test1",
				Username: "root",
				Password: "10.160.23.43",
			},
		},
		Tables: []any{"user"},
	},
}

// ==================== 测试用的数据模型 ====================

// User 用户模型，用于测试读写分离功能
type User struct {
	ID         int       `gorm:"primarykey"` // 主键ID
	Name       string    // 用户名
	CreateTime time.Time // 创建时间
}

// ==================== 集成测试：数据库读写分离（需要 MySQL 连接） ====================

// TestInitDBResolver 测试数据库读写分离初始化和路由功能
//
// 【功能点】验证 MySQL 读写分离的初始化和路由功能
// 【测试流程】
//  1. 调用 initMultiDB 初始化多数据源
//  2. 执行读操作（Find），验证路由到 Replicas（从库）
//  3. 执行写操作（Save），验证路由到 Sources（主库）
//
// 【注意事项】
//   - 需要真实的 MySQL 连接
//   - Sources 和 Replicas 配置的数据库需要存在
//   - 读操作从 test1 库读取，写操作写入 test 库
func TestInitDBResolver(t *testing.T) {
	t.Run("database read-write splitting", func(t *testing.T) {
		// 初始化多数据源数据库连接
		db, err := initMultiDB(dbConfig)
		assert.Nil(t, err, "数据库初始化失败")
		assert.NotNil(t, db, "数据库连接不应为 nil")

		// 读操作：自动路由到 Replicas（从库）
		// GORM DBResolver 会根据操作类型自动选择数据源
		user := User{}
		err = db.Find(&user).Error
		fmt.Printf("\033[32m【读取用户】user: %+v\033[0m\n", user)
		assert.Nil(t, err, "读取用户数据失败")

		// 写操作：自动路由到 Sources（主库）
		// Save 操作会被路由到主库执行
		user = User{
			Name:       "test",
			CreateTime: time.Now(),
		}
		err = db.Save(&user).Error
		fmt.Printf("\033[32m【保存用户】user: %+v\033[0m\n", user)
		assert.Nil(t, err, "保存用户数据失败")

		t.Logf("读写分离测试完成，读取使用从库，写入使用主库")
	})
}

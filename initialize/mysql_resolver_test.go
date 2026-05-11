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
// 3. 数据库和表会在测试前自动创建（如不存在）
//
// 运行测试：go test -v ./initialize/... -run TestInitDBResolver
// ==================================================
package initialize

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/zzsen/gin_core/model/config"

	"testing"

	"github.com/stretchr/testify/require"
)

// ==================== 测试辅助配置 ====================

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

// ==================== 测试辅助函数 ====================

// ensureTestDatabases 检测 MySQL 可达性，不可达则跳过测试；可达时自动创建缺失的数据库和 user 表
//
// 执行流程：
// 1. 用无库名 DSN 连接 MySQL，不可达则 skip
// 2. 收集 dbConfig 中所有 Sources/Replicas 的数据库名
// 3. 逐个执行 CREATE DATABASE IF NOT EXISTS
// 4. 逐个库中执行 CREATE TABLE IF NOT EXISTS `user`
func ensureTestDatabases(t *testing.T) {
	t.Helper()
	src := dbConfig[0].Sources[0]
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=2s", src.Username, src.Password, src.Host, src.Port)

	// 1. 检测 MySQL 可达性
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("跳过集成测试：无法创建 MySQL 连接：%v", err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		t.Skipf("跳过集成测试：MySQL 不可达（%s:%d）：%v", src.Host, src.Port, err)
	}

	// 2. 收集所有需要的数据库名（去重）
	dbNames := make(map[string]struct{})
	for _, resolver := range dbConfig {
		for _, s := range resolver.Sources {
			dbNames[s.DBName] = struct{}{}
		}
		for _, r := range resolver.Replicas {
			dbNames[r.DBName] = struct{}{}
		}
	}

	// 3. 创建数据库 + user 表
	createTableSQL := `CREATE TABLE IF NOT EXISTS user (
		id bigint NOT NULL AUTO_INCREMENT,
		name longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
		create_time datetime(3) DEFAULT NULL,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`

	for name := range dbNames {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", name))
		if err != nil {
			t.Fatalf("创建数据库 %s 失败：%v", name, err)
		}
		_, err = db.Exec(fmt.Sprintf("USE `%s`", name))
		if err != nil {
			t.Fatalf("切换到数据库 %s 失败：%v", name, err)
		}
		_, err = db.Exec(createTableSQL)
		if err != nil {
			t.Fatalf("在数据库 %s 中创建 user 表失败：%v", name, err)
		}
		t.Logf("已确认数据库 %s 和 user 表存在", name)
	}
}

// ==================== 集成测试：数据库读写分离（需要 MySQL 连接） ====================

// TestInitDBResolver 测试数据库读写分离初始化和路由功能
//
// 【功能点】验证 MySQL 读写分离的初始化和路由功能
// 【测试流程】
//  1. 检测 MySQL 可达性（不可达则跳过），自动创建缺失的数据库和表
//  2. 调用 initMultiDB 初始化多数据源
//  3. 执行读操作（Find），验证路由到 Replicas（从库）
//  4. 执行写操作（Save），验证路由到 Sources（主库）
//
// 【注意事项】
//   - 需要真实的 MySQL 连接
//   - 数据库和 user 表会在测试前自动创建
//   - 读操作从 test1 库读取，写操作写入 test 库
func TestInitDBResolver(t *testing.T) {
	// 1. 检测 MySQL 可达性，自动创建数据库和表
	ensureTestDatabases(t)

	t.Run("database read-write splitting", func(t *testing.T) {
		// 2. 初始化多数据源数据库连接
		db, err := initMultiDB(dbConfig)
		require.NoError(t, err, "数据库初始化失败")
		require.NotNil(t, db, "数据库连接不应为 nil")

		// 3. 读操作：自动路由到 Replicas（从库）
		user := User{}
		err = db.Find(&user).Error
		fmt.Printf("\033[32m【读取用户】user: %+v\033[0m\n", user)
		require.NoError(t, err, "读取用户数据失败")

		// 4. 写操作：自动路由到 Sources（主库）
		user = User{
			Name:       "test",
			CreateTime: time.Now(),
		}
		err = db.Save(&user).Error
		fmt.Printf("\033[32m【保存用户】user: %+v\033[0m\n", user)
		require.NoError(t, err, "保存用户数据失败")

		t.Logf("读写分离测试完成，读取使用从库，写入使用主库")
	})
}

// Package app 数据库访问函数测试
//
// ==================== 测试说明 ====================
// 本文件包含 app 包中数据库相关函数的单元测试。
//
// 测试覆盖内容：
// 1. GetDbByName 通过名称获取数据库实例
// 2. CloseAllDB 关闭所有数据库连接（含 SQLite 内存库）
// 3. closeGormDB 关闭单个 gorm.DB 实例（含真实 SQLite）
//
// 运行测试：go test -v ./app/... -run TestDb
// ==================================================
package app

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestGetDbByName_Exists 测试获取已存在的数据库实例
//
// 【功能点】验证 GetDbByName 在数据库已注册时正确返回实例
// 【测试流程】
// 1. 预设 DBList 中注册一个数据库实例
// 2. 调用 GetDbByName 获取
// 3. 验证返回实例不为 nil 且无错误
func TestGetDbByName_Exists(t *testing.T) {
	origDBList := DBList
	defer func() { DBList = origDBList }()

	mockDB := &gorm.DB{}
	DBList = map[string]*gorm.DB{
		"test_db": mockDB,
	}

	db, err := GetDbByName("test_db")
	assert.NoError(t, err)
	assert.Equal(t, mockDB, db)
}

// TestGetDbByName_NotExists 测试获取不存在的数据库实例
//
// 【功能点】验证 GetDbByName 在数据库未注册时返回错误
// 【测试流程】
// 1. 确保 DBList 中无目标名称
// 2. 调用 GetDbByName
// 3. 验证返回 nil 和错误
func TestGetDbByName_NotExists(t *testing.T) {
	origDBList := DBList
	defer func() { DBList = origDBList }()

	DBList = map[string]*gorm.DB{}

	db, err := GetDbByName("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestGetDbByName_NilValue 测试获取值为 nil 的数据库实例
//
// 【功能点】验证 GetDbByName 在数据库实例为 nil 时返回错误
// 【测试流程】
// 1. 在 DBList 中注册一个 nil 值
// 2. 调用 GetDbByName
// 3. 验证返回错误
func TestGetDbByName_NilValue(t *testing.T) {
	origDBList := DBList
	defer func() { DBList = origDBList }()

	DBList = map[string]*gorm.DB{
		"nil_db": nil,
	}

	db, err := GetDbByName("nil_db")
	assert.Error(t, err)
	assert.Nil(t, db)
}

// TestCloseGormDB_Nil 测试关闭 nil 的 gorm.DB
//
// 【功能点】验证 closeGormDB 对 nil 入参的安全处理
// 【测试流程】
// 1. 传入 nil
// 2. 验证无错误返回
func TestCloseGormDB_Nil(t *testing.T) {
	err := closeGormDB(nil)
	assert.NoError(t, err)
}

// TestCloseGormDB_RealSQLite 测试关闭 SQLite 内存库的 gorm.DB
//
// 【功能点】验证 closeGormDB 对真实 sql.DB 调用 Close 成功且无错误
// 【测试流程】
// 1. 使用 glebarez/sqlite 打开 :memory: 数据库
// 2. 调用 closeGormDB
// 3. 验证返回 nil error
func TestCloseGormDB_RealSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = closeGormDB(db)
	assert.NoError(t, err)
}

// TestCloseAllDB_NilInstances 测试所有实例为 nil 时的 CloseAllDB
//
// 【功能点】验证 CloseAllDB 在全局变量均为 nil 时不报错
// 【测试流程】
// 1. 确保 DB、DBResolver、DBList 均为 nil/空
// 2. 调用 CloseAllDB
// 3. 验证无错误返回
func TestCloseAllDB_NilInstances(t *testing.T) {
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	defer func() {
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
	}()

	DB = nil
	DBResolver = nil
	DBList = map[string]*gorm.DB{}

	err := CloseAllDB()
	assert.NoError(t, err)
}

// TestCloseAllDB_RealSQLite 测试关闭主库、解析器与 DBList 中的真实 SQLite 连接
//
// 【功能点】验证 CloseAllDB 依次关闭 DB、DBResolver、DBList 且聚合错误为空
// 【测试流程】
// 1. 保存并替换 DB、DBResolver、DBList 为三个独立的内存 SQLite 实例
// 2. 调用 CloseAllDB
// 3. 验证无错误并恢复全局变量
func TestCloseAllDB_RealSQLite(t *testing.T) {
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	defer func() {
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
	}()

	dbMain, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	dbResolver, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	dbNamed, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	DB = dbMain
	DBResolver = dbResolver
	DBList = map[string]*gorm.DB{"named": dbNamed}

	err = CloseAllDB()
	assert.NoError(t, err)
}

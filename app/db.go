package app

import (
	"fmt"

	"gorm.io/gorm"
)

// GetDbByName 通过名称获取db，如果不存在则返回错误
// 参数：
//   - dbname: 数据库别名
//
// 返回：
//   - *gorm.DB: 数据库实例
//   - error: 如果数据库不存在或未初始化则返回错误
func GetDbByName(dbname string) (*gorm.DB, error) {
	lock.RLock()
	defer lock.RUnlock()
	db, ok := DBList[dbname]
	if !ok || db == nil {
		return nil, fmt.Errorf("[db] 数据库 `%s` 未初始化或不可用", dbname)
	}
	return db, nil
}

// closeGormDB 关闭单个 gorm.DB 实例的底层 sql.DB 连接
func closeGormDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CloseAllDB 关闭所有数据库连接，包括主库、读写分离解析器和多数据库列表
//
// 关闭流程：
// 1. 关闭主数据库连接（DB）
// 2. 关闭数据库读写分离解析器连接（DBResolver）
// 3. 关闭多数据库列表中的所有连接（DBList）
func CloseAllDB() error {
	var errs []error

	// 1. 关闭主数据库
	if err := closeGormDB(DB); err != nil {
		errs = append(errs, fmt.Errorf("[db] 关闭主数据库失败: %w", err))
	}

	// 2. 关闭读写分离解析器
	if err := closeGormDB(DBResolver); err != nil {
		errs = append(errs, fmt.Errorf("[db] 关闭数据库解析器失败: %w", err))
	}

	// 3. 关闭多数据库列表
	lock.RLock()
	for name, db := range DBList {
		if err := closeGormDB(db); err != nil {
			errs = append(errs, fmt.Errorf("[db] 关闭数据库 `%s` 失败: %w", name, err))
		}
	}
	lock.RUnlock()

	if len(errs) > 0 {
		return fmt.Errorf("关闭数据库连接时出现 %d 个错误: %v", len(errs), errs)
	}
	return nil
}

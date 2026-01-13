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

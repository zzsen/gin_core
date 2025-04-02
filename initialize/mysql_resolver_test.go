package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/model/config"

	"testing"

	"github.com/stretchr/testify/assert"
)

// 建表语句，需要提前创建好
// CREATE TABLE `user` (
//
//		`id` bigint NOT NULL AUTO_INCREMENT,
//		`name` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
//		`create_time` datetime(3) DEFAULT NULL,
//		PRIMARY KEY (`id`)
//	  ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

// 按需调整数据库连接配置
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
		Tables: []interface{}{"user"},
	},
}

type User struct {
	ID         int `gorm:"primarykey"`
	Name       string
	CreateTime time.Time
}

func TestInitDBResolver(t *testing.T) {
	t.Run("rsa sign and valid", func(t *testing.T) {
		db, err := initMultiDB(dbConfig)
		assert.Nil(t, err)
		assert.NotNil(t, db)

		// read data, user replicas
		user := User{}
		err = db.Find(&user).Error
		fmt.Printf("\033[32muser: %+v\033[0m\n", user)
		assert.Nil(t, err)

		// save data, user sources
		user = User{
			Name: "test",
		}
		err = db.Save(&user).Error
		fmt.Printf("\033[32muser: %+v\033[0m\n", user)
		assert.Nil(t, err)
	})
}

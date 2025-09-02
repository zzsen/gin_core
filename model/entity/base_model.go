// Package entity 提供数据实体的基础结构定义
// 本文件定义了所有数据实体的基础模型，包含通用的审计字段和软删除支持
package entity

import "time"

// BaseModel 数据实体基础模型
// 该结构体定义了所有数据实体的通用字段，包括创建者、删除标记、创建时间和更新时间
// 所有继承此结构体的实体都会自动包含这些基础字段，实现统一的审计和生命周期管理
type BaseModel struct {
	CreateUserId int       `gorm:"not null; column:create_user_id; comment:创建人id" json:"createUserId"`                      // 创建用户ID，记录数据创建者的身份标识
	IsDeleted    bool      `gorm:"not null; type:boolean; column:is_deleted; comment:是否已删除;default:false" json:"isDeleted"` // 软删除标记，true表示已删除，false表示正常状态
	CreateTime   time.Time `gorm:"not null; column:create_time; comment:创建时间" json:"createTime"`                            // 创建时间，记录数据首次创建的时间戳
	UpdateTime   time.Time `gorm:"not null; column:update_time; comment:更新时间" json:"updateTime"`                            // 更新时间，记录数据最后修改的时间戳
}

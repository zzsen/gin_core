package entity

import "time"

type BaseModel struct {
	CreateUserId int       `gorm:"not null; column:create_user_id; comment:创建人id" json:"createUserId"`
	IsDeleted    bool      `gorm:"not null; type:boolean; column:is_deleted; comment:是否已删除;default:false" json:"isDeleted"`
	CreateTime   time.Time `gorm:"not null; column:create_time; comment:创建时间" json:"createTime"`
	UpdateTime   time.Time `gorm:"not null; column:update_time; comment:更新时间" json:"updateTime"`
}

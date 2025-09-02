// Package request 提供HTTP请求参数的数据结构定义
// 本文件定义了分页查询的请求参数结构，包含分页逻辑和参数验证
package request

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// Page 分页查询通用输入参数结构
// 该结构体定义了分页查询所需的基本参数，支持页码、每页大小和关键字搜索
type Page struct {
	PageIndex int    `json:"pageIndex" form:"pageIndex"` // 页码，从1开始计数
	PageSize  int    `json:"pageSize" form:"pageSize"`   // 每页大小，控制每页返回的数据条数
	Keyword   string `json:"keyword" form:"keyword"`     // 关键字，用于模糊搜索匹配
}

// Paginate 生成GORM分页查询函数
// 该方法返回一个GORM查询函数，用于在数据库查询中应用分页逻辑
// 返回：
//   - func(db *gorm.DB) *gorm.DB: GORM分页查询函数
func (r *Page) Paginate() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// 页码验证和修正，确保页码从1开始
		if r.PageIndex <= 0 {
			r.PageIndex = 1
		}

		// 每页大小验证和修正，设置合理的上下限
		switch {
		case r.PageSize > 1000:
			r.PageSize = 1000 // 限制最大每页大小为1000，防止查询过大数据量
		case r.PageSize <= 0:
			r.PageSize = 10 // 设置默认每页大小为10
		}

		// 计算数据库查询的偏移量
		offset := (r.PageIndex - 1) * r.PageSize

		// 应用分页查询，使用Offset和Limit实现分页
		return db.Offset(offset).Limit(r.PageSize)
	}
}

// GetPageFromCtx 从Gin上下文中获取分页参数
// 该方法从HTTP请求中解析分页参数，并进行参数验证和默认值设置
// 参数：
//   - ctx: Gin上下文，包含HTTP请求信息
//
// 返回：
//   - Page: 解析后的分页参数，包含验证后的页码和每页大小
func GetPageFromCtx(ctx *gin.Context) Page {
	var page Page

	// 从请求中绑定分页参数，支持JSON和表单格式
	ctx.ShouldBind(&page)

	// 使用validator验证分页参数的有效性
	err := validator.New().Struct(page)
	if err != nil {
		// 如果验证失败，设置默认的分页参数
		page.PageIndex = 1 // 默认第一页
		page.PageSize = 20 // 默认每页20条
	}
	return page
}

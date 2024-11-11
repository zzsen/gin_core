package request

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// PageInfo Paging common input parameter structure
type Page struct {
	PageIndex int    `json:"pageIndex" form:"pageIndex"` // 页码
	PageSize  int    `json:"pageSize" form:"pageSize"`   // 每页大小
	Keyword   string `json:"keyword" form:"keyword"`     // 关键字
}

func (r *Page) Paginate() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.PageIndex <= 0 {
			r.PageIndex = 1
		}
		switch {
		case r.PageSize > 100:
			r.PageSize = 100
		case r.PageSize <= 0:
			r.PageSize = 10
		}
		offset := (r.PageIndex - 1) * r.PageSize
		return db.Offset(offset).Limit(r.PageSize)
	}
}

// GetPageFromCtx get page parameters
func GetPageFromCtx(ctx *gin.Context) Page {
	var page Page
	ctx.ShouldBind(&page)
	err := validator.New().Struct(page)
	if err != nil {
		page.PageIndex = 1
		page.PageSize = 20
	}
	return page
}

package request

// GetById Find by id structure
type GetByIdReqs struct {
	Id int `json:"id" form:"id"` // 主键ID
}

// GetById Find by id structure
type GetByIdsReqs struct {
	Ids []int `json:"ids" form:"ids"`
}

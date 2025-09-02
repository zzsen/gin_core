// Package request 提供HTTP请求参数的数据结构定义
// 本文件定义了通用的请求参数结构，包含按ID查询和批量ID查询的请求格式
package request

// GetByIdReqs 根据ID查询单个实体的请求结构
// 该结构体用于接收前端传递的单个ID参数，支持JSON和表单两种数据格式
type GetByIdReqs struct {
	Id int `json:"id" form:"id"` // 主键ID，用于查询指定的数据实体
}

// GetByIdsReqs 根据ID列表批量查询实体的请求结构
// 该结构体用于接收前端传递的多个ID参数，支持JSON和表单两种数据格式
type GetByIdsReqs struct {
	Ids []int `json:"ids" form:"ids"` // ID列表，用于批量查询多个数据实体
}

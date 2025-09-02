// Package response 提供HTTP响应数据的数据结构定义
// 本文件定义了分页查询结果的响应结构，用于返回分页数据和分页信息
package response

// PageResult 分页查询结果响应结构
// 该结构体定义了分页查询的标准响应格式，包含数据列表、总数和分页信息
type PageResult struct {
	List     any   `json:"list"`     // 数据列表，包含当前页的数据内容，支持任意类型
	Total    int64 `json:"total"`    // 数据总数，表示符合查询条件的所有数据条数
	Page     int   `json:"page"`     // 当前页码，表示当前返回的是第几页的数据
	PageSize int   `json:"pageSize"` // 每页大小，表示每页包含的数据条数
}

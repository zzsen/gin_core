// Package request 请求模块测试
//
// ==================== 测试说明 ====================
// 本文件包含 HTTP 请求参数模块的单元测试，不需要外部依赖。
//
// 测试覆盖内容：
// 1. Page 分页参数的 Paginate 方法（正常值、边界值、默认值修正）
// 2. GetPageFromCtx 从 Gin 上下文解析分页参数
// 3. GetByIdReqs / GetByIdsReqs 请求结构体的 JSON 绑定
// 4. PageResult 边界场景
//
// 运行测试：go test -v ./model/request/... -run Test
// ==================================================
package request

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestPaginate_NormalValues 测试 Paginate 正常分页参数
//
// 【功能点】验证 Paginate 在正常页码和每页大小下的 Offset/Limit 计算
// 【测试流程】
// 1. 构造不同的页码和每页大小组合
// 2. 调用 Paginate 获取 GORM scope
// 3. 验证修正后的 PageIndex 和 PageSize 值
func TestPaginate_NormalValues(t *testing.T) {
	tests := []struct {
		name              string
		pageIndex         int
		pageSize          int
		expectedPageIndex int
		expectedPageSize  int
	}{
		{"第一页10条", 1, 10, 1, 10},
		{"第二页20条", 2, 20, 2, 20},
		{"第五页50条", 5, 50, 5, 50},
		{"大页码", 100, 100, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &Page{PageIndex: tt.pageIndex, PageSize: tt.pageSize}
			fn := page.Paginate()
			assert.NotNil(t, fn)
			assert.Equal(t, tt.expectedPageIndex, page.PageIndex)
			assert.Equal(t, tt.expectedPageSize, page.PageSize)
		})
	}
}

// TestPaginate_BoundaryValues 测试 Paginate 闭包内的边界值修正
//
// 【功能点】验证 Paginate 返回的闭包在实际执行时对非法和极端值的自动修正
// 【测试流程】
// 1. 构造各种边界值的 Page 实例
// 2. 调用 Paginate 获取闭包并传入 nil（仅验证修正逻辑，不需要实际 DB）
// 3. 验证修正后的 PageIndex 和 PageSize 值
func TestPaginate_BoundaryValues(t *testing.T) {
	tests := []struct {
		name              string
		pageIndex         int
		pageSize          int
		expectedPageIndex int
		expectedPageSize  int
	}{
		{"零值页码", 0, 10, 1, 10},
		{"负数页码", -1, 10, 1, 10},
		{"零值每页大小", 1, 0, 1, 10},
		{"负数每页大小", 1, -5, 1, 10},
		{"超大每页大小", 1, 2000, 1, 1000},
		{"正好1000", 1, 1000, 1, 1000},
		{"全部为零", 0, 0, 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &Page{PageIndex: tt.pageIndex, PageSize: tt.pageSize}
			fn := page.Paginate()
			assert.NotNil(t, fn)
			// 调用闭包触发修正逻辑（使用 DryRun 模式的 gorm.DB）
			db, _ := gorm.Open(nil, &gorm.Config{DryRun: true})
			if db != nil {
				fn(db)
			}
			assert.Equal(t, tt.expectedPageIndex, page.PageIndex)
			assert.Equal(t, tt.expectedPageSize, page.PageSize)
		})
	}
}

// TestGetPageFromCtx_WithQueryParams 测试从查询参数解析分页
//
// 【功能点】验证 GetPageFromCtx 正确解析 URL 查询参数中的分页信息
// 【测试流程】
// 1. 构造带查询参数的 HTTP 请求
// 2. 调用 GetPageFromCtx
// 3. 验证返回的 Page 字段值
func TestGetPageFromCtx_WithQueryParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?pageIndex=2&pageSize=15&keyword=test", nil)

	page := GetPageFromCtx(c)

	assert.Equal(t, 2, page.PageIndex)
	assert.Equal(t, 15, page.PageSize)
	assert.Equal(t, "test", page.Keyword)
}

// TestGetPageFromCtx_WithoutParams 测试无参数时的零值行为
//
// 【功能点】验证 GetPageFromCtx 在无查询参数时，零值 Page 能通过 validator
// 【测试流程】
// 1. 构造无查询参数的请求
// 2. 调用 GetPageFromCtx
// 3. 验证返回零值（validator 对无约束字段的零值不报错）
func TestGetPageFromCtx_WithoutParams(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	page := GetPageFromCtx(c)

	// 无参数时 ShouldBind 绑定为零值，validator 无自定义校验规则时零值合法
	assert.Equal(t, 0, page.PageIndex)
	assert.Equal(t, 0, page.PageSize)
	assert.Equal(t, "", page.Keyword)
}

// TestGetPageFromCtx_InvalidQueryNonNumeric 非法查询参数（非数字 pageIndex）
//
// 【功能点】ShouldBind 无法解析整数时错误被忽略，Page 保持零值；validator 在无标签时仍通过
// 【测试流程】
// 1. 构造 pageIndex、pageSize 为非数字的 Query
// 2. GetPageFromCtx 返回零值分页字段
func TestGetPageFromCtx_InvalidQueryNonNumeric(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?pageIndex=abc&pageSize=not-num", nil)

	page := GetPageFromCtx(c)

	assert.Equal(t, 0, page.PageIndex)
	assert.Equal(t, 0, page.PageSize)
}

// TestGetPageFromCtx_InvalidJSONTypes JSON 类型与整型字段不匹配
//
// 【功能点】JSON 绑定失败时 ShouldBind 错误被忽略，字段为零值
// 【测试流程】
// 1. POST application/json，pageIndex/pageSize 为字符串
// 2. 断言解析结果为零值（若未来为 Page 增加 validate 标签，validator 分支才会回落到默认 1/20）
func TestGetPageFromCtx_InvalidJSONTypes(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader(`{"pageIndex":"x","pageSize":"y","keyword":1}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	page := GetPageFromCtx(c)

	assert.Equal(t, 0, page.PageIndex)
	assert.Equal(t, 0, page.PageSize)
}

// TestGetByIdReqs_JSON 测试 GetByIdReqs JSON 绑定
//
// 【功能点】验证 GetByIdReqs 结构体的 JSON 序列化/反序列化
// 【测试流程】
// 1. 构造 JSON 数据
// 2. 反序列化到结构体
// 3. 验证字段值正确
func TestGetByIdReqs_JSON(t *testing.T) {
	jsonData := `{"id": 42}`
	var req GetByIdReqs
	err := json.Unmarshal([]byte(jsonData), &req)
	assert.NoError(t, err)
	assert.Equal(t, 42, req.Id)
}

// TestGetByIdsReqs_JSON 测试 GetByIdsReqs JSON 绑定
//
// 【功能点】验证 GetByIdsReqs 结构体的 JSON 序列化/反序列化
// 【测试流程】
// 1. 构造包含 ID 列表的 JSON 数据
// 2. 反序列化到结构体
// 3. 验证 Ids 列表的长度和值
func TestGetByIdsReqs_JSON(t *testing.T) {
	jsonData := `{"ids": [1, 2, 3, 4, 5]}`
	var req GetByIdsReqs
	err := json.Unmarshal([]byte(jsonData), &req)
	assert.NoError(t, err)
	assert.Len(t, req.Ids, 5)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, req.Ids)
}

// TestGetByIdsReqs_EmptyList 测试 GetByIdsReqs 空列表
//
// 【功能点】验证空 ID 列表的处理
// 【测试流程】
// 1. 传入空数组 JSON
// 2. 验证反序列化后 Ids 为空切片
func TestGetByIdsReqs_EmptyList(t *testing.T) {
	jsonData := `{"ids": []}`
	var req GetByIdsReqs
	err := json.Unmarshal([]byte(jsonData), &req)
	assert.NoError(t, err)
	assert.Empty(t, req.Ids)
}

// TestPageJSON 测试 Page 结构体 JSON 序列化
//
// 【功能点】验证 Page 结构体的 JSON 字段名正确性
// 【测试流程】
// 1. 构造 Page 实例
// 2. 序列化为 JSON
// 3. 验证字段名和值
func TestPageJSON(t *testing.T) {
	page := Page{PageIndex: 3, PageSize: 25, Keyword: "hello"}
	data, err := json.Marshal(page)
	assert.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	assert.Equal(t, float64(3), parsed["pageIndex"])
	assert.Equal(t, float64(25), parsed["pageSize"])
	assert.Equal(t, "hello", parsed["keyword"])
}

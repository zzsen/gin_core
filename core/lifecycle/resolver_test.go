// Package lifecycle 依赖解析器测试
//
// ==================== 测试说明 ====================
// 本文件包含依赖解析器（DependencyResolver）的单元测试。
//
// 测试覆盖内容：
// 1. 无依赖场景下的拓扑排序
// 2. 链式依赖的层级排序
// 3. 菱形依赖（共享依赖）的层级排序
// 4. 循环依赖检测
// 5. 缺失依赖验证
// 6. 单服务依赖顺序查询
// 7. 空服务列表处理
//
// 运行测试：go test -v ./core/lifecycle/... -run TestResolver
// ==================================================
package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// mockService 用于测试的 mock 服务实现
type mockService struct {
	name         string
	priority     int
	dependencies []string
	shouldInit   bool
	initFn       func(ctx context.Context) error
	closeFn      func(ctx context.Context) error
}

func (m *mockService) Name() string                           { return m.name }
func (m *mockService) Priority() int                          { return m.priority }
func (m *mockService) Dependencies() []string                 { return m.dependencies }
func (m *mockService) ShouldInit(_ *config.BaseConfig) bool   { return m.shouldInit }
func (m *mockService) Init(ctx context.Context) error {
	if m.initFn != nil {
		return m.initFn(ctx)
	}
	return nil
}
func (m *mockService) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

// newMock 快速创建 mock 服务
func newMock(name string, priority int, deps ...string) *mockService {
	return &mockService{
		name:         name,
		priority:     priority,
		dependencies: deps,
		shouldInit:   true,
	}
}

// TestResolver_NoDependencies 测试无依赖场景
//
// 【功能点】所有服务无依赖时，应在同一层级，按优先级排序
// 【测试流程】
// 1. 创建 3 个无依赖服务，优先级不同
// 2. 解析后应产生 1 层
// 3. 层内按优先级排序
func TestResolver_NoDependencies(t *testing.T) {
	services := map[string]Service{
		"c": newMock("c", 3),
		"a": newMock("a", 1),
		"b": newMock("b", 2),
	}

	resolver := NewDependencyResolver(services)
	layers, err := resolver.Resolve()

	require.NoError(t, err)
	require.Len(t, layers, 1)
	assert.Equal(t, []string{"a", "b", "c"}, layers[0])
}

// TestResolver_ChainDependency 测试链式依赖
//
// 【功能点】A→B→C 链式依赖应产生 3 层
// 【测试流程】
// 1. C 无依赖，B 依赖 C，A 依赖 B
// 2. 解析后应产生 3 层：[C] → [B] → [A]
func TestResolver_ChainDependency(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1, "B"),
		"B": newMock("B", 1, "C"),
		"C": newMock("C", 1),
	}

	resolver := NewDependencyResolver(services)
	layers, err := resolver.Resolve()

	require.NoError(t, err)
	require.Len(t, layers, 3)
	assert.Equal(t, []string{"C"}, layers[0])
	assert.Equal(t, []string{"B"}, layers[1])
	assert.Equal(t, []string{"A"}, layers[2])
}

// TestResolver_DiamondDependency 测试菱形依赖
//
// 【功能点】D 依赖 B 和 C，B 和 C 都依赖 A → 3 层
// 【测试流程】
// 1. A 无依赖，B/C 依赖 A，D 依赖 B+C
// 2. 解析后：[A] → [B, C] → [D]
func TestResolver_DiamondDependency(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1),
		"B": newMock("B", 1, "A"),
		"C": newMock("C", 2, "A"),
		"D": newMock("D", 1, "B", "C"),
	}

	resolver := NewDependencyResolver(services)
	layers, err := resolver.Resolve()

	require.NoError(t, err)
	require.Len(t, layers, 3)
	assert.Equal(t, []string{"A"}, layers[0])
	assert.Equal(t, []string{"B", "C"}, layers[1])
	assert.Equal(t, []string{"D"}, layers[2])
}

// TestResolver_CyclicDependency 测试循环依赖检测
//
// 【功能点】A→B→C→A 形成环，应返回错误
// 【测试流程】
// 1. 构造循环依赖图
// 2. Resolve 应返回包含"循环依赖"的错误
func TestResolver_CyclicDependency(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1, "B"),
		"B": newMock("B", 1, "C"),
		"C": newMock("C", 1, "A"),
	}

	resolver := NewDependencyResolver(services)
	_, err := resolver.Resolve()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "循环依赖")
}

// TestResolver_ValidateDependencies 测试缺失依赖验证
//
// 【功能点】服务依赖不存在的服务时，ValidateDependencies 应报告
// 【测试流程】
// 1. A 依赖 "missing"（未注册的服务）
// 2. ValidateDependencies 应返回 A → ["missing"]
func TestResolver_ValidateDependencies(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1, "missing"),
		"B": newMock("B", 1),
	}

	resolver := NewDependencyResolver(services)
	missing := resolver.ValidateDependencies()

	require.Contains(t, missing, "A")
	assert.Equal(t, []string{"missing"}, missing["A"])
	assert.NotContains(t, missing, "B")
}

// TestResolver_ValidateDependencies_AllPresent 测试所有依赖都存在
//
// 【功能点】所有依赖都存在时返回空 map
// 【测试流程】
// 1. A 依赖 B，B 已注册
// 2. ValidateDependencies 返回空
func TestResolver_ValidateDependencies_AllPresent(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1, "B"),
		"B": newMock("B", 1),
	}

	resolver := NewDependencyResolver(services)
	missing := resolver.ValidateDependencies()
	assert.Empty(t, missing)
}

// TestResolver_GetDependencyOrder 测试单服务依赖顺序
//
// 【功能点】获取某服务的初始化顺序（扁平化）
// 【测试流程】
// 1. C→B→A 链式依赖
// 2. GetDependencyOrder("C") 应返回 [A, B, C]
func TestResolver_GetDependencyOrder(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1),
		"B": newMock("B", 1, "A"),
		"C": newMock("C", 1, "B"),
	}

	resolver := NewDependencyResolver(services)
	order, err := resolver.GetDependencyOrder("C")

	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, order)
}

// TestResolver_GetDependencyOrder_UnknownService 查询不存在的服务名返回完整拓扑序
//
// 【功能点】GetDependencyOrder 在 serviceName 未出现时遍历全部层后返回完整扁平序列
// 【测试流程】
// 1. 构造 A→B 链
// 2. GetDependencyOrder("Z") 返回 [A,B]
func TestResolver_GetDependencyOrder_UnknownService(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1),
		"B": newMock("B", 1, "A"),
	}
	resolver := NewDependencyResolver(services)
	order, err := resolver.GetDependencyOrder("Z")
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, order)
}

// TestResolver_Empty 测试空服务列表
//
// 【功能点】空服务列表应返回空层级
// 【测试流程】
// 1. 传入空 map
// 2. Resolve 返回空 layers，无错误
func TestResolver_Empty(t *testing.T) {
	resolver := NewDependencyResolver(map[string]Service{})
	layers, err := resolver.Resolve()

	require.NoError(t, err)
	assert.Empty(t, layers)
}

// TestResolver_IgnoreExternalDependency 测试忽略外部依赖
//
// 【功能点】依赖不在 services 中的服务应被忽略（不计入入度）
// 【测试流程】
// 1. A 依赖 "external"（不在 services 中）
// 2. Resolve 不报错，A 在第一层
func TestResolver_IgnoreExternalDependency(t *testing.T) {
	services := map[string]Service{
		"A": newMock("A", 1, "external"),
	}

	resolver := NewDependencyResolver(services)
	layers, err := resolver.Resolve()

	require.NoError(t, err)
	require.Len(t, layers, 1)
	assert.Equal(t, []string{"A"}, layers[0])
}

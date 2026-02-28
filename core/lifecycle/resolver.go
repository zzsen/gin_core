package lifecycle

import (
	"fmt"
	"sort"
)

// DependencyResolver 依赖解析器
// 负责解析服务之间的依赖关系，并生成按层级分组的初始化顺序
type DependencyResolver struct {
	services map[string]Service
}

// NewDependencyResolver 创建依赖解析器
func NewDependencyResolver(services map[string]Service) *DependencyResolver {
	return &DependencyResolver{
		services: services,
	}
}

// Resolve 解析依赖关系，返回按层级分组的初始化顺序。
// 返回值 [][]string 中，每个内层数组是可以并行初始化的服务组。
//
// 执行流程：
// 1. 使用 DFS 检测循环依赖，发现环则返回错误
// 2. 构建入度表（每个服务依赖数）和邻接表（被依赖关系）
// 3. 使用 Kahn 算法逐层剥离入度为 0 的节点，按优先级排序后加入当前层
// 4. 每剥离一层，更新依赖它的服务的入度，直到所有服务处理完毕
func (r *DependencyResolver) Resolve() ([][]string, error) {
	// 1. 检测循环依赖
	if err := r.detectCycle(); err != nil {
		return nil, err
	}

	// 2. 构建入度表和邻接表
	inDegree := make(map[string]int)
	dependents := make(map[string][]string)

	// 初始化所有服务的入度为0
	for name := range r.services {
		inDegree[name] = 0
	}

	// 计算入度和被依赖关系
	for name, service := range r.services {
		deps := service.Dependencies()
		for _, dep := range deps {
			// 只计算存在的依赖
			if _, exists := r.services[dep]; exists {
				inDegree[name]++
				dependents[dep] = append(dependents[dep], name)
			}
		}
	}

	// 3. Kahn 算法：逐层剥离入度为 0 的服务
	var layers [][]string

	for {
		// 找出当前层所有入度为0的服务
		var currentLayer []string
		for name, degree := range inDegree {
			if degree == 0 {
				currentLayer = append(currentLayer, name)
			}
		}

		// 如果没有入度为0的服务，说明处理完成
		if len(currentLayer) == 0 {
			break
		}

		// 按优先级排序（同一层级内）
		sort.Slice(currentLayer, func(i, j int) bool {
			return r.services[currentLayer[i]].Priority() < r.services[currentLayer[j]].Priority()
		})

		// 添加到结果
		layers = append(layers, currentLayer)

		// 4. 移除当前层节点，更新后续节点入度
		for _, name := range currentLayer {
			delete(inDegree, name)
			for _, dependent := range dependents[name] {
				if _, exists := inDegree[dependent]; exists {
					inDegree[dependent]--
				}
			}
		}
	}

	return layers, nil
}

// detectCycle 检测循环依赖。
// 使用 DFS（三色标记法）检测有向图中的环：
//   - 状态 0（白色）：未访问
//   - 状态 1（灰色）：访问中（在当前递归栈上）
//   - 状态 2（黑色）：已完成（所有后继已处理）
//
// 执行流程：
// 1. 对每个未访问的节点启动 DFS
// 2. 进入节点时标记为"访问中"并加入路径栈
// 3. 遇到"访问中"节点说明存在环，从路径栈中提取循环路径
// 4. 所有后继处理完毕后标记为"已完成"并回溯
func (r *DependencyResolver) detectCycle() error {
	// 状态：0=未访问，1=访问中，2=已完成
	state := make(map[string]int)
	path := make([]string, 0) // 记录当前路径，用于报告循环依赖

	var dfs func(name string) error
	dfs = func(name string) error {
		if state[name] == 1 {
			// 找到环，构建循环路径
			cycleStart := -1
			for i, n := range path {
				if n == name {
					cycleStart = i
					break
				}
			}
			cyclePath := append(path[cycleStart:], name)
			return fmt.Errorf("检测到循环依赖: %v", cyclePath)
		}
		if state[name] == 2 {
			return nil // 已处理
		}

		state[name] = 1 // 标记为访问中
		path = append(path, name)

		service, exists := r.services[name]
		if exists {
			for _, dep := range service.Dependencies() {
				// 只检查存在的依赖
				if _, depExists := r.services[dep]; depExists {
					if err := dfs(dep); err != nil {
						return err
					}
				}
			}
		}

		path = path[:len(path)-1] // 回溯
		state[name] = 2           // 标记为已完成
		return nil
	}

	// 对所有服务执行 DFS
	for name := range r.services {
		if state[name] == 0 {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateDependencies 验证所有依赖是否存在
// 返回缺失的依赖列表
func (r *DependencyResolver) ValidateDependencies() map[string][]string {
	missing := make(map[string][]string)

	for name, service := range r.services {
		for _, dep := range service.Dependencies() {
			if _, exists := r.services[dep]; !exists {
				missing[name] = append(missing[name], dep)
			}
		}
	}

	return missing
}

// GetDependencyOrder 获取单个服务的依赖初始化顺序（扁平化）
func (r *DependencyResolver) GetDependencyOrder(serviceName string) ([]string, error) {
	layers, err := r.Resolve()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, layer := range layers {
		for _, name := range layer {
			result = append(result, name)
			if name == serviceName {
				return result, nil
			}
		}
	}

	return result, nil
}

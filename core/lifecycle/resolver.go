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

// Resolve 解析依赖关系，返回按层级分组的初始化顺序
// 返回值: [][]string，每个内层数组是可以并行初始化的服务
// 算法：使用 Kahn 算法进行拓扑排序，同时按层级分组
func (r *DependencyResolver) Resolve() ([][]string, error) {
	// 1. 检测循环依赖
	if err := r.detectCycle(); err != nil {
		return nil, err
	}

	// 2. 构建入度表和邻接表
	inDegree := make(map[string]int)        // 入度（依赖数量）
	dependents := make(map[string][]string) // 被依赖关系（谁依赖我）

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

	// 3. 使用 Kahn 算法进行拓扑排序，同时按层级分组
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

		// 从图中移除这些服务，更新入度
		for _, name := range currentLayer {
			delete(inDegree, name)
			// 更新依赖这些服务的其他服务的入度
			for _, dependent := range dependents[name] {
				if _, exists := inDegree[dependent]; exists {
					inDegree[dependent]--
				}
			}
		}
	}

	return layers, nil
}

// detectCycle 检测循环依赖
// 使用 DFS 检测有向图中的环
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

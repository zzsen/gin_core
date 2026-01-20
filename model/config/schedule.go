// Package config 提供应用程序的配置结构定义
// 本文件定义了定时任务的配置结构，支持cron表达式和函数执行
package config

import (
	"reflect"
	"runtime"
)

// ScheduleInfo 定时任务配置信息
// 该结构体定义了定时任务的执行策略和要执行的函数
type ScheduleInfo struct {
	Name                 string `yaml:"name"`                   // 定时任务名称
	Cron                 string `yaml:"cron"`                   // cron表达式，定义定时任务的执行时间规则
	Cmd                  func() `yaml:"cmd"`                    // 定时任务执行的函数，无参数无返回值的函数类型
	ShouldRunImmediately bool   `yaml:"should_run_immediately"` // 是否在服务启动后立即执行
}

// GetFuncInfo 获取定时任务函数的详细信息
// 该方法通过反射获取传入函数的名称，用于日志记录和调试
// 返回：
//   - string: 函数名称，如果获取失败则返回空字符串
func (s *ScheduleInfo) GetFuncInfo() string {
	// 使用 reflect.ValueOf 获取传入函数的反射值
	value := reflect.ValueOf(s.Cmd)

	// 检查反射值是否为函数类型
	if value.Kind() != reflect.Func {
		return ""
	}

	// 获取函数的指针，用于后续获取函数信息
	pc := value.Pointer()

	// 根据函数指针获取函数的详细信息
	funcInfo := runtime.FuncForPC(pc)
	if funcInfo == nil {
		return ""
	}

	// 返回函数名，包含包路径和函数名
	return funcInfo.Name()
}

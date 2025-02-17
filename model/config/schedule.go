package config

import (
	"reflect"
	"runtime"
)

type ScheduleInfo struct {
	// cron表达式
	Cron string
	// 定时任务执行的函数
	Cmd func()
}

func (s *ScheduleInfo) GetFuncInfo() string {
	// 使用 reflect.ValueOf 获取传入函数的反射值
	value := reflect.ValueOf(s.Cmd)
	if value.Kind() != reflect.Func {
		return ""
	}
	// 获取函数的指针
	pc := value.Pointer()
	// 根据函数指针获取函数的详细信息
	funcInfo := runtime.FuncForPC(pc)
	if funcInfo == nil {
		return ""
	}
	// 返回函数名
	return funcInfo.Name()
}

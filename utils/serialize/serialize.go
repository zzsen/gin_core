// Package serialize 提供结构体与 map 之间的序列化转换工具。
//
// 内部通过 encoding/json 实现中间转换，因此字段映射遵循 json tag 规则。
package serialize

import (
	"encoding/json"
	"reflect"
)

// MapToStruct 将 map 对象转换为结构体对象。
// 内部先将 map 序列化为 JSON，再反序列化到目标结构体，因此字段映射遵循 json tag。
//
// 参数：
//   - m: 源 map，key 应与目标结构体的 json tag 一致
//   - s: 目标结构体指针，必须传入指针类型以接收转换结果
//
// 返回：
//   - error: 序列化或反序列化失败时返回错误
func MapToStruct(m map[string]any, s any) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return err
	}

	return nil
}

// StructToMap 将结构体对象转换为 map 对象，支持传入结构体值或结构体指针。
// map 的 key 优先使用字段的 json tag，若未定义则使用字段名。
//
// 参数：
//   - s: 源结构体值或指针，传入指针时会自动解引用
//
// 返回：
//   - map[string]any: 转换后的 map，key 为字段名或 json tag
func StructToMap(s any) map[string]any {
	m := make(map[string]any)

	v := reflect.Indirect(reflect.ValueOf(s))
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i).Tag.Get("json")
		if field == "" {
			field = t.Field(i).Name
		}
		m[field] = v.Field(i).Interface()
	}

	return m
}

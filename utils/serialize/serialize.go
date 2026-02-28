package serialize

import (
	"encoding/json"
	"reflect"
)

// MapToStruct将map对象转换为结构体对象
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

// StructToMap 将结构体对象转换为 map 对象，支持传入结构体值或结构体指针
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

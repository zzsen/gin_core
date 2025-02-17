package serialize

import (
	"encoding/json"
	"reflect"
)

// MapToStruct将map对象转换为结构体对象
func MapToStruct(m map[string]interface{}, s interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return err
	}

	return nil
}

// StructToMap将结构体对象转换为map对象
func StructToMap(s interface{}) map[string]interface{} {
	m := make(map[string]interface{})

	// 获取结构体类型和值
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)

	// 遍历结构体字段并将其添加到map对象中
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i).Tag.Get("json")
		if field == "" {
			field = t.Field(i).Name
		}
		m[field] = v.Field(i).Interface()
	}

	return m
}

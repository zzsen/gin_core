// Package serialize 序列化工具功能测试
package serialize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStruct 测试用的结构体
type TestStruct struct {
	Name     string                 `json:"name"`
	Age      int                    `json:"age"`
	Email    string                 `json:"email"`
	Active   bool                   `json:"active"`
	Score    float64                `json:"score"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
}

// TestStructWithCustomTags 测试自定义JSON标签的结构体
type TestStructWithCustomTags struct {
	UserName  string                 `json:"user_name"`
	UserAge   int                    `json:"user_age"`
	UserEmail string                 `json:"user_email"`
	IsActive  bool                   `json:"is_active"`
	UserScore float64                `json:"user_score"`
	UserTags  []string               `json:"user_tags"`
	UserData  map[string]interface{} `json:"user_data"`
}

// TestStructWithoutTags 测试没有JSON标签的结构体
type TestStructWithoutTags struct {
	Name     string
	Age      int
	Email    string
	Active   bool
	Score    float64
	Tags     []string
	Metadata map[string]interface{}
}

// TestStructWithEmptyTags 测试空JSON标签的结构体
type TestStructWithEmptyTags struct {
	Name   string `json:""`
	Age    int    `json:"-"`
	Email  string `json:"email"`
	Active bool   `json:"active"`
}

// TestMapToStruct 测试MapToStruct函数
func TestMapToStruct(t *testing.T) {
	tests := []struct {
		name     string         // 测试用例名称
		input    map[string]any // 输入map
		target   any            // 目标结构体
		expected any            // 期望结果
		wantErr  bool           // 是否期望出错
	}{
		{
			name: "simple struct conversion",
			input: map[string]any{
				"name":   "John Doe",
				"age":    30,
				"email":  "john@example.com",
				"active": true,
				"score":  95.5,
				"tags":   []string{"golang", "testing"},
				"metadata": map[string]interface{}{
					"department": "engineering",
					"level":      "senior",
				},
			},
			target: &TestStruct{},
			expected: &TestStruct{
				Name:   "John Doe",
				Age:    30,
				Email:  "john@example.com",
				Active: true,
				Score:  95.5,
				Tags:   []string{"golang", "testing"},
				Metadata: map[string]interface{}{
					"department": "engineering",
					"level":      "senior",
				},
			},
			wantErr: false,
		},
		{
			name: "struct with custom tags",
			input: map[string]any{
				"user_name":  "Jane Smith",
				"user_age":   25,
				"user_email": "jane@example.com",
				"is_active":  true,
				"user_score": 88.0,
				"user_tags":  []string{"python", "data"},
				"user_data": map[string]interface{}{
					"role": "data scientist",
				},
			},
			target: &TestStructWithCustomTags{},
			expected: &TestStructWithCustomTags{
				UserName:  "Jane Smith",
				UserAge:   25,
				UserEmail: "jane@example.com",
				IsActive:  true,
				UserScore: 88.0,
				UserTags:  []string{"python", "data"},
				UserData: map[string]interface{}{
					"role": "data scientist",
				},
			},
			wantErr: false,
		},
		{
			name: "struct without json tags",
			input: map[string]any{
				"Name":   "Bob Wilson",
				"Age":    35,
				"Email":  "bob@example.com",
				"Active": false,
				"Score":  77.3,
				"Tags":   []string{"java", "spring"},
				"Metadata": map[string]interface{}{
					"team": "backend",
				},
			},
			target: &TestStructWithoutTags{},
			expected: &TestStructWithoutTags{
				Name:   "Bob Wilson",
				Age:    35,
				Email:  "bob@example.com",
				Active: false,
				Score:  77.3,
				Tags:   []string{"java", "spring"},
				Metadata: map[string]interface{}{
					"team": "backend",
				},
			},
			wantErr: false,
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			target:   &TestStruct{},
			expected: &TestStruct{},
			wantErr:  false,
		},
		{
			name: "map with extra fields",
			input: map[string]any{
				"name":          "Alice",
				"age":           28,
				"email":         "alice@example.com",
				"active":        true,
				"score":         92.0,
				"tags":          []string{"react", "nodejs"},
				"extra_field":   "this should be ignored",
				"another_extra": 123,
			},
			target: &TestStruct{},
			expected: &TestStruct{
				Name:   "Alice",
				Age:    28,
				Email:  "alice@example.com",
				Active: true,
				Score:  92.0,
				Tags:   []string{"react", "nodejs"},
			},
			wantErr: false,
		},
		{
			name: "map with missing fields",
			input: map[string]any{
				"name": "Charlie",
				"age":  22,
			},
			target: &TestStruct{},
			expected: &TestStruct{
				Name: "Charlie",
				Age:  22,
			},
			wantErr: false,
		},
		{
			name:     "nil map",
			input:    nil,
			target:   &TestStruct{},
			expected: &TestStruct{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行转换
			err := MapToStruct(tt.input, tt.target)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, tt.target)
			}
		})
	}
}

// TestMapToStruct_InvalidTarget 测试无效目标类型
func TestMapToStruct_InvalidTarget(t *testing.T) {
	t.Run("non-pointer target", func(t *testing.T) {
		input := map[string]any{"name": "test"}
		var target TestStruct // 不是指针

		err := MapToStruct(input, target)
		assert.Error(t, err)
	})

	t.Run("nil target", func(t *testing.T) {
		input := map[string]any{"name": "test"}

		err := MapToStruct(input, nil)
		assert.Error(t, err)
	})
}

// TestStructToMap 测试StructToMap函数
func TestStructToMap(t *testing.T) {
	tests := []struct {
		name     string         // 测试用例名称
		input    any            // 输入结构体
		expected map[string]any // 期望结果
	}{
		{
			name: "simple struct with json tags",
			input: TestStruct{
				Name:   "John Doe",
				Age:    30,
				Email:  "john@example.com",
				Active: true,
				Score:  95.5,
				Tags:   []string{"golang", "testing"},
				Metadata: map[string]interface{}{
					"department": "engineering",
					"level":      "senior",
				},
			},
			expected: map[string]any{
				"name":   "John Doe",
				"age":    30,
				"email":  "john@example.com",
				"active": true,
				"score":  95.5,
				"tags":   []string{"golang", "testing"},
				"metadata": map[string]interface{}{
					"department": "engineering",
					"level":      "senior",
				},
			},
		},
		{
			name: "struct with custom json tags",
			input: TestStructWithCustomTags{
				UserName:  "Jane Smith",
				UserAge:   25,
				UserEmail: "jane@example.com",
				IsActive:  true,
				UserScore: 88.0,
				UserTags:  []string{"python", "data"},
				UserData: map[string]interface{}{
					"role": "data scientist",
				},
			},
			expected: map[string]any{
				"user_name":  "Jane Smith",
				"user_age":   25,
				"user_email": "jane@example.com",
				"is_active":  true,
				"user_score": 88.0,
				"user_tags":  []string{"python", "data"},
				"user_data": map[string]interface{}{
					"role": "data scientist",
				},
			},
		},
		{
			name: "struct without json tags",
			input: TestStructWithoutTags{
				Name:   "Bob Wilson",
				Age:    35,
				Email:  "bob@example.com",
				Active: false,
				Score:  77.3,
				Tags:   []string{"java", "spring"},
				Metadata: map[string]interface{}{
					"team": "backend",
				},
			},
			expected: map[string]any{
				"Name":   "Bob Wilson",
				"Age":    35,
				"Email":  "bob@example.com",
				"Active": false,
				"Score":  77.3,
				"Tags":   []string{"java", "spring"},
				"Metadata": map[string]interface{}{
					"team": "backend",
				},
			},
		},
		{
			name: "struct with empty json tags",
			input: TestStructWithEmptyTags{
				Name:   "Alice",
				Age:    28,
				Email:  "alice@example.com",
				Active: true,
			},
			expected: map[string]any{
				"Name":   "Alice",
				"-":      28, // Age字段的JSON标签是"-"，所以键名是"-"
				"email":  "alice@example.com",
				"active": true,
			},
		},
		{
			name:  "empty struct",
			input: TestStruct{},
			expected: map[string]any{
				"name":     "",
				"age":      0,
				"email":    "",
				"active":   false,
				"score":    0.0,
				"tags":     []string(nil),
				"metadata": map[string]interface{}(nil),
			},
		},
		{
			name: "struct with zero values",
			input: TestStruct{
				Name:     "",
				Age:      0,
				Email:    "",
				Active:   false,
				Score:    0.0,
				Tags:     []string{},
				Metadata: map[string]interface{}{},
			},
			expected: map[string]any{
				"name":     "",
				"age":      0,
				"email":    "",
				"active":   false,
				"score":    0.0,
				"tags":     []string{},
				"metadata": map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行转换
			result := StructToMap(tt.input)

			// 验证结果
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStructToMap_EdgeCases 测试边界情况
func TestStructToMap_EdgeCases(t *testing.T) {
	t.Run("pointer to struct", func(t *testing.T) {
		input := &TestStruct{
			Name: "Pointer Test",
			Age:  40,
		}

		// 这应该会panic，因为StructToMap期望的是结构体值，不是指针
		assert.Panics(t, func() {
			StructToMap(input)
		})
	})

	t.Run("nil struct", func(t *testing.T) {
		var input *TestStruct = nil

		// 这应该会panic，因为reflect.TypeOf(nil)会panic
		assert.Panics(t, func() {
			StructToMap(input)
		})
	})
}

// TestRoundTrip 测试往返转换
func TestRoundTrip(t *testing.T) {
	t.Run("struct to map to struct", func(t *testing.T) {
		original := TestStruct{
			Name:   "Round Trip Test",
			Age:    33,
			Email:  "roundtrip@example.com",
			Active: true,
			Score:  85.7,
			Tags:   []string{"test", "roundtrip"},
			Metadata: map[string]interface{}{
				"test": true,
				"id":   123,
			},
		}

		// 结构体转map
		m := StructToMap(original)
		assert.NotNil(t, m)

		// map转结构体
		var converted TestStruct
		err := MapToStruct(m, &converted)
		assert.Nil(t, err)

		// 验证往返转换结果一致（除了数字类型可能的变化）
		assert.Equal(t, original.Name, converted.Name)
		assert.Equal(t, original.Age, converted.Age)
		assert.Equal(t, original.Email, converted.Email)
		assert.Equal(t, original.Active, converted.Active)
		assert.Equal(t, original.Score, converted.Score)
		assert.Equal(t, original.Tags, converted.Tags)
		// 对于Metadata中的数字，JSON解析可能会改变类型
		assert.Equal(t, original.Metadata["test"], converted.Metadata["test"])
		assert.Equal(t, float64(original.Metadata["id"].(int)), converted.Metadata["id"])
	})

	t.Run("map to struct to map", func(t *testing.T) {
		original := map[string]any{
			"name":   "Map Round Trip",
			"age":    29,
			"email":  "map@example.com",
			"active": false,
			"score":  91.2,
			"tags":   []string{"map", "test"},
			"metadata": map[string]interface{}{
				"source": "map",
				"count":  456,
			},
		}

		// map转结构体
		var s TestStruct
		err := MapToStruct(original, &s)
		assert.Nil(t, err)

		// 结构体转map
		converted := StructToMap(s)

		// 验证往返转换结果一致（除了数字类型可能的变化）
		assert.Equal(t, original["name"], converted["name"])
		assert.Equal(t, original["age"], converted["age"])
		assert.Equal(t, original["email"], converted["email"])
		assert.Equal(t, original["active"], converted["active"])
		assert.Equal(t, original["score"], converted["score"])
		assert.Equal(t, original["tags"], converted["tags"])
		// 对于Metadata中的数字，JSON解析可能会改变类型
		assert.Equal(t, original["metadata"].(map[string]interface{})["source"], converted["metadata"].(map[string]interface{})["source"])
		assert.Equal(t, float64(original["metadata"].(map[string]interface{})["count"].(int)), converted["metadata"].(map[string]interface{})["count"])
	})
}

// TestConcurrent 测试并发安全性
func TestConcurrent(t *testing.T) {
	t.Run("concurrent map to struct", func(t *testing.T) {
		input := map[string]any{
			"name":   "Concurrent Test",
			"age":    25,
			"email":  "concurrent@example.com",
			"active": true,
		}

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				var result TestStruct
				err := MapToStruct(input, &result)
				assert.Nil(t, err)
				assert.Equal(t, "Concurrent Test", result.Name)
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent struct to map", func(t *testing.T) {
		input := TestStruct{
			Name:   "Concurrent Test",
			Age:    25,
			Email:  "concurrent@example.com",
			Active: true,
		}

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				result := StructToMap(input)
				assert.Equal(t, "Concurrent Test", result["name"])
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

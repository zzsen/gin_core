package core

import (
	"reflect"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// overrideValidator 重写Gin框架默认的验证器
// 将Gin的默认验证器替换为自定义的验证器实现
// 这允许我们自定义验证规则、错误处理和验证行为
// 必须在服务启动早期调用，确保所有后续的请求都使用自定义验证器
func overrideValidator() {
	binding.Validator = new(defaultValidator)
}

// defaultValidator 自定义的结构体验证器
// 基于go-playground/validator/v10库实现，提供强大的结构体字段验证能力
// 使用单例模式确保验证器实例的唯一性和性能优化
//
// 特性：
// - 延迟初始化：验证器实例在首次使用时才创建
// - 线程安全：使用sync.Once确保在并发环境下的安全初始化
// - 自定义标签：使用"binding"作为验证标签名称
// - 扩展性：支持添加自定义验证规则
//
// 使用场景：
// - HTTP请求参数验证
// - 结构体字段约束检查
// - 数据完整性验证
type defaultValidator struct {
	once     sync.Once           // 确保验证器只初始化一次
	validate *validator.Validate // validator/v10 验证器实例
}

// 确保defaultValidator实现了binding.StructValidator接口
// 这是编译时检查，如果没有实现接口中的所有方法，编译将失败
var _ binding.StructValidator = &defaultValidator{}

// ValidateStruct 验证结构体字段
// 这是验证器的核心方法，对传入的结构体进行字段验证
// 支持各种验证标签，如required、min、max、email等
//
// 参数 obj: 需要验证的对象，通常是结构体或结构体指针
// 返回值: 验证错误，如果验证通过则返回nil
//
// 验证流程：
// 1. 检查传入对象是否为结构体类型
// 2. 延迟初始化验证器实例
// 3. 执行结构体字段验证
// 4. 返回验证结果
//
// 支持的验证标签示例：
//
//	type User struct {
//	  Name  string `binding:"required,min=2,max=50"`
//	  Email string `binding:"required,email"`
//	  Age   int    `binding:"min=18,max=120"`
//	}
func (v *defaultValidator) ValidateStruct(obj any) error {
	// 检查传入的对象是否为结构体类型
	// 只有结构体类型才需要进行字段验证
	if kindOfData(obj) == reflect.Struct {
		// 延迟初始化验证器实例（线程安全）
		v.lazyinit()

		// 执行结构体验证，检查所有带有binding标签的字段
		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}

	return nil
}

// Engine 获取底层验证器引擎
// 返回validator/v10的验证器实例，允许外部代码访问更高级的验证功能
// 主要用于需要直接使用validator库特性的场景
//
// 返回值: validator.Validate实例，可用于自定义验证逻辑
//
// 使用场景：
// - 注册自定义验证函数
// - 配置验证器的高级选项
// - 执行复杂的跨字段验证
//
// 使用示例：
//
//	engine := validator.Engine().(*validator.Validate)
//	engine.RegisterValidation("custom", customValidationFunc)
func (v *defaultValidator) Engine() any {
	// 确保验证器已初始化
	v.lazyinit()
	return v.validate
}

// lazyinit 延迟初始化验证器
// 使用sync.Once确保验证器只被初始化一次，即使在并发环境下也是线程安全的
// 延迟初始化模式可以避免不必要的资源消耗，只在真正需要时才创建验证器实例
//
// 初始化配置：
// - 创建新的validator实例
// - 设置验证标签名称为"binding"
// - 预留自定义验证规则扩展点
//
// 扩展说明：
// 在"add any custom validations etc. here"注释处可以添加：
// - 自定义验证函数：v.validate.RegisterValidation("tagname", func)
// - 自定义类型验证：v.validate.RegisterCustomTypeFunc(func, types...)
// - 跨字段验证：v.validate.RegisterStructValidation(func, struct{})
func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		// 创建新的validator实例
		v.validate = validator.New()

		// 设置验证标签名称为"binding"
		// 这意味着结构体字段需要使用`binding:"..."`标签来定义验证规则
		v.validate.SetTagName("binding")

		// 在此处添加自定义验证规则
		// 示例：
		// v.validate.RegisterValidation("phone", validatePhone)
		// v.validate.RegisterValidation("idcard", validateIDCard)
	})
}

// kindOfData 获取数据的反射类型
// 这是一个工具函数，用于确定传入数据的实际类型
// 自动处理指针类型，返回指针指向的实际数据类型
//
// 参数 data: 任意类型的数据
// 返回值: reflect.Kind，表示数据的实际类型
//
// 处理逻辑：
// - 如果传入的是指针，则返回指针指向的类型
// - 如果传入的是值类型，则直接返回该类型
//
// 使用场景：
// - 验证前的类型检查
// - 统一处理值类型和指针类型
// - 反射相关的类型判断
//
// 示例：
//
//	kindOfData(&User{})    // 返回 reflect.Struct
//	kindOfData(User{})     // 返回 reflect.Struct
//	kindOfData("string")   // 返回 reflect.String
//	kindOfData(&"string")  // 返回 reflect.String
func kindOfData(data any) reflect.Kind {
	// 获取数据的反射值
	value := reflect.ValueOf(data)
	// 获取值的类型
	valueType := value.Kind()

	// 如果是指针类型，则获取指针指向的实际类型
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}

	return valueType
}

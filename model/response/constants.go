// Package response 提供HTTP响应数据的数据结构定义
// 本文件定义了统一的响应码常量和响应消息，用于标准化API响应格式
package response

// responseCode 响应码结构体
// 该结构体定义了响应码和对应的消息文本，用于统一管理API响应状态
type responseCode struct {
	code int    // 响应状态码，用于标识请求处理结果
	msg  string // 响应消息文本，用于描述响应状态
}

// 预定义的响应码常量，按照功能模块和错误类型进行分类
var (
	// 特殊响应码
	ResponseNull = responseCode{code: -1} // 空回复，该回复不写入到responseBody中，一般用于文件下载等特殊场景

	// 成功响应码
	ResponseSuccess = responseCode{code: 20000, msg: "操作成功"} // 标准成功响应

	// 认证相关响应码（41xxx系列）
	ResponseLoginNotLogin  = responseCode{code: 41000, msg: "未登录"}   // 用户未登录状态
	ResponseLoginButUnAuth = responseCode{code: 41001, msg: "未认证"}   // 未通过双因子认证
	ResponseLoginInvalid   = responseCode{code: 41002, msg: "登录失效"}  // 登录会话已过期
	ResponseAuthFailed     = responseCode{code: 41010, msg: "无权限访问"} // 权限不足，拒绝访问

	// 业务逻辑响应码（50xxx系列）
	ResponseFail           = responseCode{code: 50000, msg: "操作失败"}    // 通用操作失败
	ResponseParamInvalid   = responseCode{code: 53001, msg: "参数校验不通过"} // 请求参数验证失败
	ResponseParamTypeError = responseCode{code: 50002, msg: "参数类型错误"}  // 请求参数类型不匹配

	// 系统异常响应码（90xxx系列）
	ResponseExceptionCommon  = responseCode{code: 90000, msg: "服务端异常"}     // 通用服务端异常
	ResponseExceptionRpc     = responseCode{code: 90001, msg: "调用rpc服务异常"} // RPC服务调用异常
	ResponseExceptionUnknown = responseCode{code: 90002, msg: "未知异常"}      // 未分类的系统异常
)

// GetCode 获取响应状态码
// 该方法返回响应码结构体中的状态码值
// 返回：
//   - int: 响应状态码
func (r *responseCode) GetCode() int {
	return r.code
}

// GetMsg 获取响应消息文本
// 该方法返回响应码结构体中的消息文本
// 返回：
//   - string: 响应消息文本
func (r *responseCode) GetMsg() string {
	return r.msg
}

package response

type responseCode struct {
	code int
	msg  string
}

var (
	ResponseNull             = responseCode{code: -1} //空回复，该回复不写入到responseBody中，一般用于文件下载
	ResponseSuccess          = responseCode{code: 20000, msg: "操作成功"}
	ResponseLoginNotLogin    = responseCode{code: 41000, msg: "未登录"}
	ResponseLoginButUnAuth   = responseCode{code: 41001, msg: "未认证"} //未通过双因子认证
	ResponseLoginInvalid     = responseCode{code: 41002, msg: "登录失效"}
	ResponseAuthFailed       = responseCode{code: 41010, msg: "无权限访问"}
	ResponseFail             = responseCode{code: 50000, msg: "操作失败"}
	ResponseParamInvalid     = responseCode{code: 50003, msg: "参数校验不通过"}
	ResponseParamTypeError   = responseCode{code: 50002, msg: "参数类型错误"}
	ResponseExceptionCommon  = responseCode{code: 90000, msg: "服务端异常"} //通用异常
	ResponseExceptionRpc     = responseCode{code: 90001, msg: "调用rpc服务异常"}
	ResponseExceptionUnknown = responseCode{code: 90002, msg: "未知异常"}
)

func (r *responseCode) GetCode() int {
	return r.code
}

func (r *responseCode) GetMsg() string {
	return r.msg
}

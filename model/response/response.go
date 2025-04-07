package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int    `json:"code"`
	Data any    `json:"data"`
	Msg  string `json:"msg"`
}

func Result(c *gin.Context, code int, data any, msg string) {
	c.JSON(http.StatusOK, Response{
		code,
		data,
		msg,
	})
}

func Ok(c *gin.Context) {
	Result(c, ResponseSuccess.code, map[string]any{}, ResponseSuccess.msg)
}

func OkWithMessage(c *gin.Context, message string) {
	Result(c, ResponseSuccess.code, map[string]any{}, message)
}

func OkWithData(c *gin.Context, data any) {
	Result(c, ResponseSuccess.code, data, ResponseSuccess.msg)
}

func OkWithDetail(c *gin.Context, message string, data any) {
	Result(c, ResponseSuccess.code, data, message)
}

func Fail(c *gin.Context) {
	Result(c, ResponseFail.code, map[string]any{}, ResponseFail.msg)
}

func FailWithMessage(c *gin.Context, message string) {
	Result(c, ResponseFail.code, map[string]any{}, message)
}

func FailWithDetail(c *gin.Context, message string, data any) {
	Result(c, ResponseFail.code, data, message)
}

func NoAuth(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		7,
		nil,
		message,
	})
}

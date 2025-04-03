package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func Result(c *gin.Context, code int, data interface{}, msg string) {
	c.JSON(http.StatusOK, Response{
		code,
		data,
		msg,
	})
}

func Ok(c *gin.Context) {
	Result(c, ResponseSuccess.code, map[string]interface{}{}, ResponseSuccess.msg)
}

func OkWithMessage(c *gin.Context, message string) {
	Result(c, ResponseSuccess.code, map[string]interface{}{}, message)
}

func OkWithData(c *gin.Context, data interface{}) {
	Result(c, ResponseSuccess.code, data, ResponseSuccess.msg)
}

func OkWithDetail(c *gin.Context, message string, data interface{}) {
	Result(c, ResponseSuccess.code, data, message)
}

func Fail(c *gin.Context) {
	Result(c, ResponseFail.code, map[string]interface{}{}, ResponseFail.msg)
}

func FailWithMessage(c *gin.Context, message string) {
	Result(c, ResponseFail.code, map[string]interface{}{}, message)
}

func FailWithDetail(c *gin.Context, message string, data interface{}) {
	Result(c, ResponseFail.code, data, message)
}

func NoAuth(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		7,
		nil,
		message,
	})
}

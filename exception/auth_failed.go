package exception

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)

type AuthFailed struct {
}

func (_ AuthFailed) Error() string {
	return response.ResponseAuthFailed.GetMsg()
}

func (authFailed AuthFailed) OnException(*gin.Context) (msg string, code int) {
	return authFailed.Error(), response.ResponseAuthFailed.GetCode()
}

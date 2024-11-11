package exception

import "github.com/gin-gonic/gin"

type Handler interface {
	OnException(ctx *gin.Context) (msg string, code int)
}

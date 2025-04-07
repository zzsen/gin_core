package request

import (
	"github.com/go-playground/validator/v10"
	"github.com/zzsen/gin_core/exception"
)

func Validate(s any) {
	err := validator.New().Struct(s)
	if err != nil {
		panic(exception.NewInvalidParam(err.Error()))
	}
}

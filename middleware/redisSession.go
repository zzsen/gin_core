package middleware

import (
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logging"
	"github.com/zzsen/gin_core/sessionStore"
)

func init() {
	err := core.RegisterMiddleware("redisSessionHandler", RedisSessionHandler)
	if err != nil {
		logging.Error(err.Error())
	}
}

func RedisSessionHandler() gin.HandlerFunc {
	redisStore, err := sessionStore.NewRedisStoreWithDB(
		10,
		"tcp",
		global.BaseConfig.Redis.Addr,
		global.BaseConfig.Redis.Password,
		strconv.Itoa(global.BaseConfig.Redis.DB),
		[]byte("secret"))
	if err != nil {
		panic("初始化redis session 异常：" + err.Error())
	}
	redisStore.SessionsOptions.HttpOnly = true
	redisStore.SetMaxAge(global.BaseConfig.Service.SessionExpire) //设置缓存的有效时间长
	redisStore.SetKeyPrefix(global.BaseConfig.Service.SessionPrefix)
	return sessions.Sessions(global.BaseConfig.Service.CookieKey, redisStore)
}

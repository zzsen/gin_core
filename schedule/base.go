package schedule

import (
	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/logger"
)

func StartCron() {
	c := cron.New()
	//c.AddFunc("* * * * *", Print)
	c.AddFunc("@every 10s", Print)
	// 暂不启动定时任务
	c.Start()
}

func Print() {
	logger.Info("schedule run")
}

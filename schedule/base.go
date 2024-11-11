package schedule

import (
	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/logging"
)

func StartCron() {
	c := cron.New()
	//c.AddFunc("* * * * *", Print)
	c.AddFunc("@every 10s", Print)
	// 暂不启动定时任务
	c.Start()
}

func Print() {
	logging.Info("schedule run")
}

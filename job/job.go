package job

import (
	"fmt"
	"gacha-simulator/model"
	"os"
	"time"
)

func InitJobs() {
	ticker := time.NewTicker(time.Hour * time.Duration(1))
	go cleanObsoleteResults(ticker)
}

func cleanObsoleteResults(ticker *time.Ticker) {
	for range ticker.C {
		oneWeekAgo := time.Now().Add(-(time.Hour * time.Duration(24*7)))
		var count int64
		if err := model.DB.
			Model(&model.Result{}).
			Where("time <= ?", oneWeekAgo).
			Count(&count).
			Error; err != nil {
			errorf("[ERROR]:%s\n", err)
		}
		if count > 0 {
			if err := model.DB.
				Where("time <= ?", oneWeekAgo).
				Delete(&model.Result{}).
				Error; err != nil {
				errorf("[ERROR]:%s\n", err)
			}
		}
	}
}

func errorf(format string, args ...interface{}) {
	buf := fmt.Sprintf(format, args...)
	os.Stderr.Write([]byte(buf))
}

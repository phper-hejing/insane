package utils

import "time"

// 获取当前时间（毫秒）
func Now() int64 {
	return time.Now().UnixNano() / 1e6
}

// 获取当前时间距离指定时间相差的秒数
func CurSecond(startTime uint64) uint64 {
	curTime := uint64(Now())
	curSecond := (curTime - startTime) / 1000
	if curSecond == 0 {
		return 1
	}
	return curSecond
}

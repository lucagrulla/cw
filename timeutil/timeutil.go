package timeutil

import (
	"time"
)

var (
	TimeFormat = "2006-01-02T15:04:05"
)

func ParseTime(timeStr string) time.Time {
	loc, _ := time.LoadLocation("UTC")
	t, _ := time.ParseInLocation(TimeFormat, timeStr, loc)

	return t
}

func FormatTimestamp(ts int64) string {
	return time.Unix(ts, 0).Format(TimeFormat)
}

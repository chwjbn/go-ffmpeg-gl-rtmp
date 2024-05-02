package glib

import "time"

func TimeNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func TimeStr(timeVal time.Time) string {
	return timeVal.Format("2006-01-02 15:04:05")
}

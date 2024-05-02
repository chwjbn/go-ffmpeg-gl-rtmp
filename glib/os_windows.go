package glib

import (
	"fmt"
	"golang.org/x/sys/windows"
)

func OsVer() string {

	osVer := "unknown"

	dwVersion, verErr := windows.GetVersion()
	if verErr != nil {
		return osVer
	}

	major := int(dwVersion & 0xFF)
	minor := int((dwVersion & 0xFF00) >> 8)
	build := int((dwVersion & 0xFFFF0000) >> 16)

	osVer = fmt.Sprintf("%v.%v.%v", major, minor, build)

	return osVer
}

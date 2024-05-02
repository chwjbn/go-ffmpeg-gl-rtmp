package glib

import (
	"os"
	"os/user"
	"runtime"
)

func OsType() string {
	return runtime.GOOS
}

func OsArch() string {
	return runtime.GOARCH
}

func OsHost() string {

	osHost := "unknown"

	hostName, _ := os.Hostname()
	if len(hostName) > 0 {
		osHost = hostName
	}

	return osHost

}

func OsUser() string {

	osUser := "unknown"

	userData, err := user.Current()
	if err != nil {
		return osUser
	}

	if userData == nil {
		return osUser
	}

	osUser = userData.Username

	return osUser

}

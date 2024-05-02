package glib

import "net"

func TextIsIPV4(data string) bool {

	bRet := false

	ipAddr := net.ParseIP(data)

	if ipAddr != nil {
		bRet = true
	}

	return bRet

}

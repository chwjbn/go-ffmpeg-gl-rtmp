package glib

import "strings"

func ProcShell(timeOutSec int, shellData string) (string, error) {

	retData := ""

	shellPath := "/bin/bash"

	retDataBuff, retErr := ProcExcCmd(timeOutSec, shellPath, "-c", shellData)
	if len(retDataBuff) > 0 {
		retData = string(retDataBuff)
	}

	retData = strings.TrimSpace(retData)

	return retData, retErr
}

package glib

import (
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/simplifiedchinese"
	"os"
	"path"
	"strings"
)

type Charset string

const (
	UTF8    = Charset("UTF-8")
	GB18030 = Charset("GB18030")
)

func GetCodePage() uint32 {
	return windows.GetACP()
}

func ProcShell(timeOutSec int, shellData string) (string, error) {

	retData := ""

	cmdPath := os.Getenv("ComSpec")
	if len(cmdPath) < 1 {
		sysRoot := os.Getenv("SystemRoot")
		if len(sysRoot) > 0 {
			cmdPath = path.Join(sysRoot, "system32", "cmd.exe")
		}
	}

	if !FileExists(cmdPath) {
		cmdPath = `c:\windows\system32\cmd.exe`
	}

	retDataBuff, retErr := ProcExcCmd(timeOutSec, cmdPath, "/c", shellData)
	if len(retDataBuff) > 0 {
		retData = ConvertByte2String(retDataBuff)
	}

	retData = strings.TrimSpace(retData)

	return retData, retErr
}

func ConvertByte2String(byte []byte) string {
	var str string

	charset := UTF8

	xCodePage := GetCodePage()
	if xCodePage == 936 {
		charset = GB18030
	}

	switch charset {
	case GB18030:
		var decodeBytes, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case UTF8:
		fallthrough
	default:
		str = string(byte)
	}
	return str
}

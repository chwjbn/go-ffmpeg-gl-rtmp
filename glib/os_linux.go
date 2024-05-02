package glib

import (
	"bufio"
	"os"
	"strings"
)

func OsVer() string {

	osVer := "unknown"

	osFilePath := "/etc/os-release"

	file, err := os.Open(osFilePath)
	if err != nil {
		return osVer
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineStr := scanner.Text()
		if strings.Contains(lineStr, "PRETTY_NAME") {

			iDex := strings.Index(lineStr, "=")
			if iDex > 0 {
				lineStr = lineStr[iDex+1:]
				osVer = strings.TrimSpace(lineStr)
				osVer = strings.Trim(lineStr, "\"")
			}

			break
		}

	}

	return osVer

}

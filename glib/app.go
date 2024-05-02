package glib

import (
	"os"
	"path/filepath"
)

func AppBaseDir() string {

	sRet := ""

	xFilePath, xFilePathErr := filepath.Abs(os.Args[0])
	if xFilePathErr != nil {
		return sRet
	}

	sRet = filepath.Dir(xFilePath)

	return sRet
}

func AppFileName() string {

	sRet := ""

	xFilePath, xFilePathErr := filepath.Abs(os.Args[0])
	if xFilePathErr != nil {
		return sRet
	}

	sRet = filepath.Base(xFilePath)

	return sRet

}

func AppFullPath() string {

	sRet := ""

	xFilePath, xFilePathErr := filepath.Abs(os.Args[0])
	if xFilePathErr != nil {
		return sRet
	}

	sRet = xFilePath

	return sRet

}

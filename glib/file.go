package glib

import (
	"io"
	"io/ioutil"
	"os"
)

func FileExists(path string) bool {

	bRet := false

	xFileInfo, xFileInfoErr := os.Stat(path)
	if xFileInfoErr != nil {
		return bRet
	}

	if !xFileInfo.IsDir() {
		bRet = true
	}

	return bRet

}

func FileDelete(path string) bool {

	bRet := false

	xErr := os.Remove(path)
	if xErr == nil {
		bRet = true
	}

	return bRet

}

func FileRename(oldPath string, newPath string) bool {

	bRet := false

	xErr := os.Rename(oldPath, newPath)
	if xErr == nil {
		bRet = true
	}

	return bRet

}

func FileCopy(srcPath string, desPath string) bool {

	bRet := false

	if !FileExists(srcPath) {
		return bRet
	}

	if FileExists(desPath) {
		return bRet
	}

	xDesFile, xDesFileErr := os.Create(desPath)
	if xDesFileErr != nil {
		return bRet
	}

	defer xDesFile.Close()

	xSrcFile, xSrcFileErr := os.Open(srcPath)
	if xSrcFileErr != nil {
		return bRet
	}

	defer xSrcFile.Close()

	_, xCopyErr := io.Copy(xDesFile, xSrcFile)
	if xCopyErr != nil {
		return bRet
	}

	bRet = true

	return bRet

}

func FileReadAllText(path string) string {

	sData := ""

	xFileData, xFileDataErr := os.ReadFile(path)
	if xFileDataErr != nil {
		return sData
	}

	sData = string(xFileData)

	return sData

}

func FileWriteAllText(path string, data string) bool {

	bRet := false

	xFileDataErr := ioutil.WriteFile(path, []byte(data), 0644)
	if xFileDataErr == nil {
		bRet = true
	}

	return bRet

}

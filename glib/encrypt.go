package glib

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

func EncryptMd5(data string) string {

	xHash := md5.New()
	xHash.Write([]byte(data))

	return hex.EncodeToString(xHash.Sum(nil))
}

func EncryptBase64Encode(data string) string {
	xData := base64.StdEncoding.EncodeToString([]byte(data))
	return xData
}

func EncryptBase64Decode(data string) string {

	xData := ""

	xDataBuf, xDataErr := base64.StdEncoding.DecodeString(data)

	if xDataErr != nil {
		return xData
	}

	xData = string(xDataBuf)

	return xData

}

func EncryptNewId(seed string) string {

	xData := fmt.Sprintf("%v|%v", seed, time.Now().UnixNano())

	xData = EncryptMd5(xData)

	return xData

}

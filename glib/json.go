package glib

import "encoding/json"

func JsonToStruct(data interface{}, dataJson string) error {
	var xErr error = nil
	xErr = json.Unmarshal([]byte(dataJson), data)
	return xErr
}

func JsonFromStruct(data interface{}) string {

	xData := "{}"

	jonData, jsonErr := json.Marshal(data)

	if jsonErr != nil {
		return xData
	}

	xData = string(jonData)

	return xData
}

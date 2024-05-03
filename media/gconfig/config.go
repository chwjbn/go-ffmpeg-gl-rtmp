package gconfig

import (
	"github.com/chwjbn/live-hub/glib"
	"path"
)

type TaskMeta struct {
	VideoStreamType string
	VideoStreamPath string

	DstVideoWidth  int
	DstVideoHeight int
}

func GetTaskMeta(taskId string) TaskMeta {

	var taskData TaskMeta

	taskData.VideoStreamType = "local"
	taskData.VideoStreamPath = path.Join(glib.AppBaseDir(), "test.mp4")

	taskData.DstVideoWidth = 1080
	taskData.DstVideoHeight = 1920

	return taskData

}

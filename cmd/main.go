package main

import (
	"fmt"
	"github.com/chwjbn/live-hub/glog"
	"github.com/chwjbn/live-hub/media/effect"
	"github.com/chwjbn/live-hub/media/gav"
	"github.com/chwjbn/live-hub/media/gconfig"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	_ "net/http/pprof"
)

func main() {

	glog.Info("app begin")

	go func() {
		http.ListenAndServe("0.0.0.0:54565", nil)
	}()

	test()

	fmt.Scanln()

	glog.Info("app end")

}

func test() {
	var xErr error

	xEffectProcessor, xErr := effect.NewEffectProcessor()

	if xErr != nil {
		glog.Error(xErr.Error())
	}

	xTaskMeta := gconfig.GetTaskMeta("")

	xAvProcessor, xErr := gav.NewAvProcessor(xEffectProcessor, &xTaskMeta)
	if xErr != nil {
		glog.Error(xErr.Error())
		return
	}

	xAvProcessor.RunProccess()
}

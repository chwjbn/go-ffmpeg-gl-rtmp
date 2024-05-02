package glib

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func ProcExcCmd(timeoutSec int, command string, args ...string) ([]byte, error) {

	var xError error
	var xData []byte

	var xResultBuf bytes.Buffer
	var xErrorBuf bytes.Buffer

	var xCmdHandle *exec.Cmd
	xCmdHandle = exec.Command(command, args...)

	xCmdHandle.Stdout = &xResultBuf
	xCmdHandle.Stderr = &xErrorBuf
	xCmdHandle.Env = os.Environ()

	xError = xCmdHandle.Start()
	if xError != nil {
		return xData, xError
	}

	xExitChan := make(chan error)
	go func() {
		xExitChan <- xCmdHandle.Wait()
	}()

	if timeoutSec < 0 {
		timeoutSec = 5
	}

	select {

	case <-time.After(time.Duration(timeoutSec) * time.Second):

		xError = errors.New(fmt.Sprintf("timeout after %ds", timeoutSec))

		if timeoutSec > 0 {
			xCmdHandle.Process.Signal(syscall.SIGINT)
			time.Sleep(50 * time.Millisecond)
			xCmdHandle.Process.Kill()
		}

	case xExitErr := <-xExitChan:
		if xExitErr != nil {
			xError = errors.New(fmt.Sprintf("done with exit error:%s", xExitErr.Error()))
		}

	}

	if xErrorBuf.Len() > 0 {
		xData = xErrorBuf.Bytes()
		return xData, xError
	}

	if xResultBuf.Len() > 0 {
		xData = xResultBuf.Bytes()
	}

	return xData, xError
}

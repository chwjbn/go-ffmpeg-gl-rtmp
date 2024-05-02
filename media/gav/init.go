package gav

import "runtime"

func init() {
	runtime.LockOSThread()
}

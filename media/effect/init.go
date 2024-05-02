package effect

import "runtime"

func init() {
	runtime.LockOSThread()
}

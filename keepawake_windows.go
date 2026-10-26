//go:build windows

package main

import (
	"runtime"
	"sync"
	"syscall"
)

// keepAwake: amíg van várakozó időzített küldés, a Windows nem altatja el a gépet (a kijelző
// kikapcsolhat). A SetThreadExecutionState szálhoz kötött, ezért egy saját, rögzített szálon fut.
var (
	awakeMu   sync.Mutex
	awakeStop chan struct{}
)

func keepAwake(on bool) {
	awakeMu.Lock()
	defer awakeMu.Unlock()
	if on == (awakeStop != nil) {
		return
	}
	if !on {
		close(awakeStop)
		awakeStop = nil
		return
	}
	stop := make(chan struct{})
	awakeStop = stop
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		proc := syscall.NewLazyDLL("kernel32.dll").NewProc("SetThreadExecutionState")
		const esContinuous, esSystemRequired = 0x80000000, 0x00000001
		_, _, _ = proc.Call(uintptr(esContinuous | esSystemRequired))
		<-stop
		_, _, _ = proc.Call(uintptr(esContinuous))
	}()
}

//go:build windows

package power

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

const (
	esContinuous     = 0x80000000
	esSystemRequired = 0x00000001
)

var setThreadExecutionState = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetThreadExecutionState")

func acquire() (func(), bool, error) {
	started := make(chan error, 1)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(done)
		result, _, callErr := setThreadExecutionState.Call(esContinuous | esSystemRequired)
		if result == 0 {
			started <- fmt.Errorf("SetThreadExecutionState: %w", callErr)
			return
		}
		started <- nil
		<-stop
		_, _, _ = setThreadExecutionState.Call(esContinuous)
	}()
	if err := <-started; err != nil {
		<-done
		return nil, false, err
	}
	return func() {
		close(stop)
		<-done
	}, true, nil
}

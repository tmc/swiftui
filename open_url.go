package swiftui

import (
	"runtime"
	"sync"
)

var (
	openURLLibOnce sync.Once

	_SUIViewEnvironmentOpenURL func(uintptr, uintptr) uintptr
)

func ensureOpenURLLibFuncs() {
	openURLLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_SUIViewEnvironmentOpenURL, libHandle, "SUIViewEnvironmentOpenURL")
		}
	})
}

// EnvironmentOpenURL intercepts link openings in the view subtree.
//
// Return true to mark the URL handled. Return false to allow the system to
// open it normally.
func (v View) EnvironmentOpenURL(action func(string) bool) View {
	if action == nil {
		return v
	}
	ensureOpenURLLibFuncs()
	if _SUIViewEnvironmentOpenURL == nil {
		return v
	}
	actionID := registerStringCallback(action)
	ptr := _SUIViewEnvironmentOpenURL(v.ptr, actionID)
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(actionID)
	runtime.KeepAlive(v.retained)
	return ret
}

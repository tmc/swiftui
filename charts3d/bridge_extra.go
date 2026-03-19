package charts3d

import (
	"sync"

	"github.com/ebitengine/purego"
)

var (
	surfaceFuncMu   sync.Mutex
	surfaceFuncMap  = map[uintptr]func(float64, float64) float64{}
	surfaceFuncNext uintptr
)

func registerSurfaceFunc(fn func(float64, float64) float64) uintptr {
	surfaceFuncMu.Lock()
	defer surfaceFuncMu.Unlock()
	surfaceFuncNext++
	surfaceFuncMap[surfaceFuncNext] = fn
	return surfaceFuncNext
}

func surfaceCallbackTrampoline(id uintptr, x, z float64) float64 {
	surfaceFuncMu.Lock()
	fn := surfaceFuncMap[id]
	surfaceFuncMu.Unlock()
	if fn == nil {
		return 0
	}
	return fn(x, z)
}

var (
	surfaceCallbackOnce sync.Once
	surfaceCallbackPtr  uintptr
	surfaceSetOnce      sync.Once
)

func ensureSurfaceCallbackPtr() uintptr {
	surfaceCallbackOnce.Do(func() {
		defer func() {
			if recover() != nil {
				surfaceCallbackPtr = 0
			}
		}()
		surfaceCallbackPtr = purego.NewCallback(surfaceCallbackTrampoline)
	})
	return surfaceCallbackPtr
}

func ensureSurfaceCallback() {
	surfaceSetOnce.Do(func() {
		ptr := ensureSurfaceCallbackPtr()
		if ptr != 0 {
			_CHSetSurfaceCallback(ptr)
		}
	})
}

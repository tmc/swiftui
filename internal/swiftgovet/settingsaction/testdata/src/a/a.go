package a

import "github.com/tmc/swiftui"

var dirtyFlag = swiftui.NewIntState(1)

func resetDirty() {
	dirtyFlag.Set(0)
}

func f() {
	changes := swiftui.NewIntState(1)
	saved := 0

	swiftui.Button("Save", func() { // want `Save button action only resets changes tracking state; persist or restore the underlying settings as well`
		changes.Set(0)
	})

	revert := func() {
		dirtyFlag.Set(0)
	}
	swiftui.Button("Revert", revert) // want `Revert button action only resets dirtyFlag tracking state; persist or restore the underlying settings as well`

	swiftui.Button("Apply", resetDirty) // want `Apply button action only resets dirtyFlag tracking state; persist or restore the underlying settings as well`

	swiftui.Button("Save", func() {
		saved = changes.Get()
		changes.Set(0)
	})

	swiftui.Button("Save to Palette", func() {
		changes.Set(0)
	})

	swiftui.Button("Revert", func() { // want `Revert button action only resets dirtyFlag tracking state; persist or restore the underlying settings as well`
		dirtyFlag.SetAnimatedWith(0, 1)
	})

	_ = saved
}

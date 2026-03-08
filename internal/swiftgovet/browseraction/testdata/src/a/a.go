package a

import "github.com/tmc/swiftui"

func empty() {}

func returns() { return }

func reload() {}

func f() {
	swiftui.ButtonWithImage("chevron.left", func() { // want `browser action button "chevron.left" has an empty callback; wire it to navigation or sharing behavior, or remove the control`
	})

	back := func() {}
	swiftui.Button("Back", back) // want `browser action button "Back" has an empty callback; wire it to navigation or sharing behavior, or remove the control`

	swiftui.ButtonWithImage("square.and.arrow.up", empty) // want `browser action button "square.and.arrow.up" has an empty callback; wire it to navigation or sharing behavior, or remove the control`

	swiftui.Button("Home", returns) // want `browser action button "Home" has an empty callback; wire it to navigation or sharing behavior, or remove the control`

	swiftui.ButtonWithImage("arrow.clockwise", func() {
		reload()
	})

	swiftui.Button("Share", func() {
		reload()
	})

	swiftui.ButtonWithImage("slider.horizontal.3", func() {
	})
}

package swiftui

// AppConfig configures the application window.
type AppConfig struct {
	Title  string
	Width  float64
	Height float64
}

// MenuBarConfig configures a menu bar status item.
type MenuBarConfig struct {
	Label       string  // Text shown next to the icon
	SystemImage string  // SF Symbol name for the icon
	Width       float64 // Popover width
	Height      float64 // Popover height
}

// Run starts the application event loop with the given root view.
// This blocks until the application exits.
func Run(config AppConfig, root View) {
	withCString(config.Title, func(title *byte) {
		_SUIRun(root.ptr, title, config.Width, config.Height)
	})
}

// RunMenuBar starts a menu-bar-only app with no dock icon.
// The content View is shown in a popover when the status item is clicked.
// This blocks until the application exits.
func RunMenuBar(config MenuBarConfig, content View) {
	withCString(config.Label, func(label *byte) {
		withCString(config.SystemImage, func(img *byte) {
			_SUIRunMenuBar(label, img, content.ptr, config.Width, config.Height)
		})
	})
}

// RunWithMenuBar starts a windowed app that also has a menu bar item.
// The menuContent View is shown in a popover when the status item is clicked.
// This blocks until the application exits.
func RunWithMenuBar(appConfig AppConfig, content View, menuConfig MenuBarConfig, menuContent View) {
	withCString(appConfig.Title, func(title *byte) {
		withCString(menuConfig.Label, func(menuLabel *byte) {
			withCString(menuConfig.SystemImage, func(menuImg *byte) {
				_SUIRunWithMenuBar(content.ptr, title, appConfig.Width, appConfig.Height,
					menuLabel, menuImg, menuContent.ptr, menuConfig.Width, menuConfig.Height)
			})
		})
	})
}

// UpdateMenuBarLabel changes the status item text at runtime.
func UpdateMenuBarLabel(label string) {
	withCString(label, func(l *byte) {
		_SUIUpdateMenuBarLabel(l)
	})
}

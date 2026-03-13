package swiftui

// AppConfig configures the application window.
type AppConfig struct {
	Title  string
	Width  float64
	Height float64
}

// Run starts the application event loop with the given root view.
// This blocks until the application exits.
func Run(config AppConfig, root View) {
	withCString(config.Title, func(title *byte) {
		_SUIRun(root.ptr, title, config.Width, config.Height)
	})
}

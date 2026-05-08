package swiftui

// AppLifecycle holds callbacks for application lifecycle events.
//
// Runtime surface.
//
// All fields are optional; nil callbacks are not registered. ShouldTerminate
// returns true to allow termination, false to cancel it.
type AppLifecycle struct {
	OnLaunched     func()
	OnActivate     func()
	OnResignActive func()
	ShouldTerminate func() bool
	OnTerminate    func()
}

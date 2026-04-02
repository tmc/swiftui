package a2uiruntime

type RuntimeError struct {
	Code        string
	SurfaceID   string
	ComponentID string
	Path        string
	Message     string
}

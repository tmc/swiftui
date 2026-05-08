package swiftui

// Material names a system material for BackgroundMaterial.
//
// Bridge surface.
//
// Values correspond 1:1 to SwiftUI's Material type on the Swift side and
// are forwarded as-is to [View.BackgroundStyle].
type Material string

const (
	MaterialRegular   Material = "regularMaterial"
	MaterialThin      Material = "thinMaterial"
	MaterialUltraThin Material = "ultraThinMaterial"
	MaterialThick     Material = "thickMaterial"
	MaterialBar       Material = "bar"
)

// BackgroundMaterial applies a system material as the view's background.
// For non-material background styles (e.g. "windowBackground"), use
// [View.BackgroundStyle] directly.
func (v View) BackgroundMaterial(m Material) View {
	return v.BackgroundStyle(string(m))
}

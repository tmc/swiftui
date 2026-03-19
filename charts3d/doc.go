// Package charts3d provides Go bindings for the 3D portion of Apple's Charts framework.
//
// The package is separate from [github.com/tmc/swiftui/charts] so the 2D API
// stays focused. It currently covers practical 3D chart construction:
// point, rule, and rectangle marks; surface plots; domains and scale types;
// axis labels; and camera pose and projection controls.
//
// The package requires the macOS 26 Charts APIs.
//
// # Quick start
//
//	import (
//		"github.com/tmc/swiftui/charts3d"
//	)
//
//	view := charts3d.Chart3D(
//		charts3d.PointMark(
//			charts3d.XFloat("Step", 1),
//			charts3d.YFloat("Loss", 0.42),
//			charts3d.ZFloat("Depth", 3),
//		),
//	).View()
//
//	_ = view
package charts3d

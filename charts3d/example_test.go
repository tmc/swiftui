package charts3d_test

import (
	"github.com/tmc/swiftui/charts3d"
)

func ExampleChart3D() {
	view := charts3d.Chart3D(
		charts3d.PointMark(
			charts3d.XFloat("Step", 1),
			charts3d.YFloat("Loss", 0.42),
			charts3d.ZFloat("Depth", 3),
		),
	).ChartZAxisLabel("Depth").View()

	_ = view
}

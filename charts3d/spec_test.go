package charts3d

import "testing"

func TestPointMarkSpec(t *testing.T) {
	spec := PointMark(
		XFloat("Step", 1),
		YFloat("Loss", 0.42),
		ZFloat("Depth", 3),
	).toSpec()

	if got, want := spec.Kind, int32(markPoint); got != want {
		t.Fatalf("kind = %d, want %d", got, want)
	}
	if got, want := len(spec.Dims), 3; got != want {
		t.Fatalf("dims = %d, want %d", got, want)
	}
}

func TestChart3DSpec(t *testing.T) {
	spec := Chart3D(
		PointMark(
			XFloat("Step", 1),
			YFloat("Loss", 0.42),
			ZFloat("Depth", 3),
		),
	).ChartXScaleDomain(NumberDomain(0, 10)).
		ChartZScaleType(ScaleTypeLog).
		ChartZAxisLabel("Depth").
		builder.toSpec()

	if spec.XDomain == nil {
		t.Fatal("xDomain = nil")
	}
	if got, want := spec.ZScaleType.Kind, int32(scaleTypeLog); got != want {
		t.Fatalf("z scale kind = %d, want %d", got, want)
	}
	if spec.ZAxisLabel == nil || spec.ZAxisLabel.Text != "Depth" {
		t.Fatalf("z axis label = %#v, want Depth", spec.ZAxisLabel)
	}
}

func TestSurfacePlotSpec(t *testing.T) {
	surface := SurfacePlot("X", "Y", "Z", func(x, z float64) float64 { return x + z })
	spec := Chart3D().Surface(surface).builder.toSpec()
	if spec.Surface == nil {
		t.Fatal("surface = nil")
	}
	if spec.Surface.CallbackID == 0 {
		t.Fatal("surface callback id = 0")
	}
}

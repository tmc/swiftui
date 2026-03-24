package charts

import (
	"testing"
	"time"
)

func TestChartHoverOverlaySpec(t *testing.T) {
	hover := NewChartHoverState()
	defer hover.Release()

	spec := Chart(
		PointMark(XFloat("Step", 1), YFloat("Score", 0.5)),
	).ChartOverlay(ChartHoverOverlay(hover)).
		ChartBackground(ChartHoverBackground(hover)).
		builder.toSpec()

	if got, want := len(spec.Overlays), 1; got != want {
		t.Fatalf("overlays = %d, want %d", got, want)
	}
	if got, want := spec.Overlays[0].Kind, int32(proxyLayerHoverEvent); got != want {
		t.Fatalf("overlay kind = %d, want %d", got, want)
	}
	if spec.Overlays[0].CallbackID == 0 {
		t.Fatal("overlay callback id = 0, want non-zero")
	}
	if got, want := len(spec.Backgrounds), 1; got != want {
		t.Fatalf("backgrounds = %d, want %d", got, want)
	}
	if got, want := spec.Backgrounds[0].Kind, int32(proxyLayerHoverEvent); got != want {
		t.Fatalf("background kind = %d, want %d", got, want)
	}
	if spec.Backgrounds[0].CallbackID == 0 {
		t.Fatal("background callback id = 0, want non-zero")
	}
}

func TestChartHoverStateOnChange(t *testing.T) {
	hover := NewChartHoverState()
	defer hover.Release()

	changes := make(chan ChartHoverEvent, 3)
	cancel := hover.OnChange(func(event ChartHoverEvent) {
		changes <- event
	})
	defer cancel()

	dateSeconds := 1_700_000_000.25
	chartHoverCallbackTrampoline(
		hover.id,
		1,
		12.5,
		7.25,
		3,
		4,
		100,
		50,
		int32(ChartHoverValueNumber),
		42.5,
		0,
		int32(ChartHoverValueDate),
		0,
		dateSeconds,
	)

	got := <-changes
	if !got.Active {
		t.Fatal("active = false, want true")
	}
	if got.PlotX != 12.5 || got.PlotY != 7.25 {
		t.Fatalf("plot position = (%v,%v), want (12.5,7.25)", got.PlotX, got.PlotY)
	}
	if got.FrameMinX != 3 || got.FrameMinY != 4 || got.FrameWidth != 100 || got.FrameHeight != 50 {
		t.Fatalf("frame = (%v,%v,%v,%v), want (3,4,100,50)", got.FrameMinX, got.FrameMinY, got.FrameWidth, got.FrameHeight)
	}
	if value, ok := got.XValue.Number(); !ok || value != 42.5 {
		t.Fatalf("x value = (%v,%v), want (42.5,true)", value, ok)
	}
	wantDate := time.UnixMilli(int64(dateSeconds * 1000))
	if value, ok := got.YValue.Date(); !ok || !value.Equal(wantDate) {
		t.Fatalf("y value = (%v,%v), want (%v,true)", value, ok, wantDate)
	}

	current := hover.Get()
	if !current.Active {
		t.Fatal("Get().Active = false, want true")
	}

	chartHoverCallbackTrampoline(
		hover.id,
		0,
		0,
		0,
		3,
		4,
		100,
		50,
		0,
		0,
		0,
		0,
		0,
		0,
	)

	got = <-changes
	if got.Active {
		t.Fatal("inactive event reported active")
	}
	if got.XValue.Kind() != ChartHoverValueNone || got.YValue.Kind() != ChartHoverValueNone {
		t.Fatalf("inactive value kinds = (%v,%v), want none", got.XValue.Kind(), got.YValue.Kind())
	}
	if hover.Get().Active {
		t.Fatal("Get().Active = true after ended event")
	}
}

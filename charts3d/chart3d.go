package charts3d

import (
	"encoding/json"
	"sync"

	"github.com/tmc/swiftui"
)

type chart3DPoseKind int32

const (
	chart3DPoseDefault chart3DPoseKind = iota
	chart3DPoseFront
	chart3DPoseBack
	chart3DPoseTop
	chart3DPoseBottom
	chart3DPoseLeft
	chart3DPoseRight
	chart3DPoseCustom
)

// Chart3DPose controls the viewpoint of a 3D chart.
type Chart3DPose struct {
	kind        chart3DPoseKind
	azimuth     float64
	inclination float64
}

var (
	Chart3DPoseDefault = Chart3DPose{kind: chart3DPoseDefault}
	Chart3DPoseFront   = Chart3DPose{kind: chart3DPoseFront}
	Chart3DPoseBack    = Chart3DPose{kind: chart3DPoseBack}
	Chart3DPoseTop     = Chart3DPose{kind: chart3DPoseTop}
	Chart3DPoseBottom  = Chart3DPose{kind: chart3DPoseBottom}
	Chart3DPoseLeft    = Chart3DPose{kind: chart3DPoseLeft}
	Chart3DPoseRight   = Chart3DPose{kind: chart3DPoseRight}
)

func CustomChart3DPose(azimuthDegrees, inclinationDegrees float64) Chart3DPose {
	return Chart3DPose{kind: chart3DPoseCustom, azimuth: azimuthDegrees, inclination: inclinationDegrees}
}

// CameraProjection controls the chart camera projection.
type CameraProjection int32

const (
	CameraProjectionAutomatic CameraProjection = iota
	CameraProjectionOrthographic
	CameraProjectionPerspective
)

type surfacePlotSpec struct {
	XLabel     string `json:"xLabel"`
	YLabel     string `json:"yLabel"`
	ZLabel     string `json:"zLabel"`
	CallbackID uint64 `json:"callbackID"`
}

// SurfacePlotContent is a bridged Swift Charts surface plot.
type SurfacePlotContent struct {
	spec surfacePlotSpec
}

// SurfacePlot creates a 3D surface plot from a Go callback.
func SurfacePlot(xLabel, yLabel, zLabel string, f func(x, z float64) float64) SurfacePlotContent {
	id := registerSurfaceFunc(f)
	return SurfacePlotContent{
		spec: surfacePlotSpec{
			XLabel:     xLabel,
			YLabel:     yLabel,
			ZLabel:     zLabel,
			CallbackID: uint64(id),
		},
	}
}

// Chart3DView is a lazily-built 3D chart view.
type Chart3DView struct {
	builder *chart3DBuilder
}

type chart3DBuilder struct {
	marks   []Mark
	surface *SurfacePlotContent

	xDomain    *Domain
	yDomain    *Domain
	zDomain    *Domain
	xScaleType ScaleType
	yScaleType ScaleType
	zScaleType ScaleType

	xAxisLabel *axisLabel
	yAxisLabel *axisLabel
	zAxisLabel *axisLabel
	pose       *Chart3DPose
	projection CameraProjection

	once sync.Once
	view swiftui.View
}

type chart3DSpec struct {
	Marks      []markSpec       `json:"marks,omitempty"`
	Surface    *surfacePlotSpec `json:"surface,omitempty"`
	XDomain    *domainSpec      `json:"xDomain,omitempty"`
	YDomain    *domainSpec      `json:"yDomain,omitempty"`
	ZDomain    *domainSpec      `json:"zDomain,omitempty"`
	XScaleType scaleTypeSpec    `json:"xScaleType"`
	YScaleType scaleTypeSpec    `json:"yScaleType"`
	ZScaleType scaleTypeSpec    `json:"zScaleType"`
	XAxisLabel *axisLabelSpec   `json:"xAxisLabel,omitempty"`
	YAxisLabel *axisLabelSpec   `json:"yAxisLabel,omitempty"`
	ZAxisLabel *axisLabelSpec   `json:"zAxisLabel,omitempty"`
	Pose       *chart3DPoseSpec `json:"pose,omitempty"`
	Projection int32            `json:"projection"`
}

type chart3DPoseSpec struct {
	Kind        int32   `json:"kind"`
	Azimuth     float64 `json:"azimuth"`
	Inclination float64 `json:"inclination"`
}

func (p Chart3DPose) toSpec() chart3DPoseSpec {
	return chart3DPoseSpec{
		Kind:        int32(p.kind),
		Azimuth:     p.azimuth,
		Inclination: p.inclination,
	}
}

// Chart3D creates a 3D chart from z-aware marks.
func Chart3D(marks ...Mark) Chart3DView {
	return Chart3DView{
		builder: &chart3DBuilder{
			marks:      append([]Mark(nil), marks...),
			xScaleType: ScaleTypeAutomatic,
			yScaleType: ScaleTypeAutomatic,
			zScaleType: ScaleTypeAutomatic,
			projection: CameraProjectionAutomatic,
		},
	}
}

func (c Chart3DView) cloneBuilder() *chart3DBuilder {
	if c.builder == nil {
		return &chart3DBuilder{
			xScaleType: ScaleTypeAutomatic,
			yScaleType: ScaleTypeAutomatic,
			zScaleType: ScaleTypeAutomatic,
			projection: CameraProjectionAutomatic,
		}
	}
	out := &chart3DBuilder{
		marks:      append([]Mark(nil), c.builder.marks...),
		xScaleType: c.builder.xScaleType,
		yScaleType: c.builder.yScaleType,
		zScaleType: c.builder.zScaleType,
		projection: c.builder.projection,
	}
	if c.builder.xDomain != nil {
		d := *c.builder.xDomain
		out.xDomain = &d
	}
	if c.builder.yDomain != nil {
		d := *c.builder.yDomain
		out.yDomain = &d
	}
	if c.builder.zDomain != nil {
		d := *c.builder.zDomain
		out.zDomain = &d
	}
	if c.builder.xAxisLabel != nil {
		label := *c.builder.xAxisLabel
		out.xAxisLabel = &label
	}
	if c.builder.yAxisLabel != nil {
		label := *c.builder.yAxisLabel
		out.yAxisLabel = &label
	}
	if c.builder.zAxisLabel != nil {
		label := *c.builder.zAxisLabel
		out.zAxisLabel = &label
	}
	if c.builder.pose != nil {
		pose := *c.builder.pose
		out.pose = &pose
	}
	if c.builder.surface != nil {
		surface := *c.builder.surface
		out.surface = &surface
	}
	return out
}

func (c Chart3DView) with(mut func(*chart3DBuilder)) Chart3DView {
	builder := c.cloneBuilder()
	mut(builder)
	return Chart3DView{builder: builder}
}

func buildAxisLabel(text string, opts []AxisLabelOption) *axisLabel {
	label := axisLabel{text: text}
	for _, opt := range opts {
		switch opt.kind {
		case axisLabelOptionPosition:
			label.position = opt.position
			label.hasPosition = true
		case axisLabelOptionAlignment:
			label.alignment = opt.alignment
			label.hasAlignment = true
		case axisLabelOptionSpacing:
			label.spacing = opt.spacing
			label.hasSpacing = true
		}
	}
	return &label
}

func (c Chart3DView) Surface(surface SurfacePlotContent) Chart3DView {
	return c.with(func(b *chart3DBuilder) {
		value := surface
		b.surface = &value
	})
}

func (c Chart3DView) ChartXScaleDomain(domain Domain) Chart3DView {
	return c.with(func(b *chart3DBuilder) { d := domain; b.xDomain = &d })
}

func (c Chart3DView) ChartYScaleDomain(domain Domain) Chart3DView {
	return c.with(func(b *chart3DBuilder) { d := domain; b.yDomain = &d })
}

func (c Chart3DView) ChartZScaleDomain(domain Domain) Chart3DView {
	return c.with(func(b *chart3DBuilder) { d := domain; b.zDomain = &d })
}

func (c Chart3DView) ChartXScaleType(scaleType ScaleType) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.xScaleType = scaleType })
}

func (c Chart3DView) ChartYScaleType(scaleType ScaleType) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.yScaleType = scaleType })
}

func (c Chart3DView) ChartZScaleType(scaleType ScaleType) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.zScaleType = scaleType })
}

func (c Chart3DView) ChartXAxisLabel(text string, opts ...AxisLabelOption) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.xAxisLabel = buildAxisLabel(text, opts) })
}

func (c Chart3DView) ChartYAxisLabel(text string, opts ...AxisLabelOption) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.yAxisLabel = buildAxisLabel(text, opts) })
}

func (c Chart3DView) ChartZAxisLabel(text string, opts ...AxisLabelOption) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.zAxisLabel = buildAxisLabel(text, opts) })
}

func (c Chart3DView) Chart3DPose(pose Chart3DPose) Chart3DView {
	return c.with(func(b *chart3DBuilder) { p := pose; b.pose = &p })
}

func (c Chart3DView) Chart3DCameraProjection(projection CameraProjection) Chart3DView {
	return c.with(func(b *chart3DBuilder) { b.projection = projection })
}

func (b *chart3DBuilder) toSpec() chart3DSpec {
	spec := chart3DSpec{
		XScaleType: b.xScaleType.toSpec(),
		YScaleType: b.yScaleType.toSpec(),
		ZScaleType: b.zScaleType.toSpec(),
		Projection: int32(b.projection),
	}
	if len(b.marks) > 0 {
		spec.Marks = make([]markSpec, len(b.marks))
		for i, mark := range b.marks {
			spec.Marks[i] = mark.toSpec()
		}
	}
	if b.surface != nil {
		surface := b.surface.spec
		spec.Surface = &surface
	}
	if b.xDomain != nil {
		d := b.xDomain.toSpec()
		spec.XDomain = &d
	}
	if b.yDomain != nil {
		d := b.yDomain.toSpec()
		spec.YDomain = &d
	}
	if b.zDomain != nil {
		d := b.zDomain.toSpec()
		spec.ZDomain = &d
	}
	if b.xAxisLabel != nil {
		label := b.xAxisLabel.toSpec()
		spec.XAxisLabel = &label
	}
	if b.yAxisLabel != nil {
		label := b.yAxisLabel.toSpec()
		spec.YAxisLabel = &label
	}
	if b.zAxisLabel != nil {
		label := b.zAxisLabel.toSpec()
		spec.ZAxisLabel = &label
	}
	if b.pose != nil {
		pose := b.pose.toSpec()
		spec.Pose = &pose
	}
	return spec
}

func (c Chart3DView) build() swiftui.View {
	if c.builder == nil {
		return swiftui.ViewFromPointer(0)
	}
	c.builder.once.Do(func() {
		ensureSurfaceCallback()
		data, err := json.Marshal(c.builder.toSpec())
		if err != nil || len(data) == 0 {
			return
		}
		ptr := _CHBuildChart3D((*byte)(&data[0]), int32(len(data)))
		c.builder.view = swiftui.ViewFromPointer(ptr)
	})
	return c.builder.view
}

func (c Chart3DView) Pointer() uintptr     { return c.build().Pointer() }
func (c Chart3DView) View() swiftui.View   { return c.build() }
func (c Chart3DView) AsView() swiftui.View { return c.build() }

func (c Chart3DView) Frame(width, height float64) swiftui.View {
	return c.build().Frame(width, height)
}

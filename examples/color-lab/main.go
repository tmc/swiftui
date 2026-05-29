//go:build darwin
// +build darwin

// Command color-lab is an interactive color mixing and exploration tool.
//
// It provides RGB+alpha sliders, a color picker, complementary/analogous/triadic
// color display, a saveable palette, WCAG contrast checking, and blend mode previews.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

// savedColor holds an RGB color saved to the palette.
type savedColor struct {
	r, g, b float64
}

var (
	palette   []savedColor
	paletteMu sync.Mutex
)

func main() {
	rState := swiftui.NewFloatState(0.2)
	gState := swiftui.NewFloatState(0.5)
	bState := swiftui.NewFloatState(0.8)
	aState := swiftui.NewFloatState(1.0)
	colorState := swiftui.NewColorState(0.2, 0.5, 0.8, 1.0)
	version := swiftui.NewIntState(0) // triggers DynamicView rebuilds

	syncFromSliders := func() {
		colorState.Set(rState.Get(), gState.Get(), bState.Get(), aState.Get())
		version.Set(version.Get() + 1)
	}
	syncFromPicker := func() {
		rState.Set(colorState.R())
		gState.Set(colorState.G())
		bState.Set(colorState.B())
		aState.Set(colorState.A())
		version.Set(version.Get() + 1)
	}

	leftColumn := swiftui.VStackSpaced(12,
		swiftui.GroupBox("Color Mixer",
			swiftui.VStackSpaced(8,
				swiftui.FloatSlider("Red", rState, 0, 1, syncFromSliders),
				swiftui.FloatSlider("Green", gState, 0, 1, syncFromSliders),
				swiftui.FloatSlider("Blue", bState, 0, 1, syncFromSliders),
				swiftui.FloatSlider("Alpha", aState, 0, 1, syncFromSliders),
				swiftui.ColorPicker("Pick Color", colorState, syncFromPicker),
			).Padding(4),
		),
		swiftui.GroupBox("Preview",
			swiftui.VStackSpaced(8,
				swiftui.DynamicView(version, func(_ int) swiftui.View {
					r, g, b, a := rState.Get(), gState.Get(), bState.Get(), aState.Get()
					return swiftui.VStack(
						swiftui.RoundedRectangle(16).
							Fill(swiftui.RGBA(r, g, b, a)).
							Frame(200, 200).
							AsView().
							Border(swiftui.RGBA(0.5, 0.5, 0.5, 0.3), 1),
						swiftui.Text(hexString(r, g, b)).
							Font(swiftui.FontSystemDesign(16, swiftui.WeightMedium, swiftui.DesignMonospaced)).
							ForegroundStyleNamed("secondary").
							AsView(),
					)
				}),
			).Padding(4),
		),
	).MaxFrame(-1, 0)

	rightColumn := swiftui.VStackSpaced(12,
		// Complementary colors
		swiftui.GroupBox("Related Colors",
			swiftui.DynamicView(version, func(_ int) swiftui.View {
				r, g, b := rState.Get(), gState.Get(), bState.Get()
				return swiftui.VStackSpaced(10,
					colorRow("Complementary", []savedColor{{1 - r, 1 - g, 1 - b}}),
					colorRow("Analogous", analogous(r, g, b)),
					colorRow("Triadic", triadic(r, g, b)),
				)
			}),
		),

		// Palette saver
		swiftui.GroupBox("Palette",
			swiftui.VStackSpaced(8,
				swiftui.Button("Save to Palette", func() {
					paletteMu.Lock()
					palette = append(palette, savedColor{rState.Get(), gState.Get(), bState.Get()})
					paletteMu.Unlock()
					version.Set(version.Get() + 1)
				}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
				swiftui.DynamicView(version, func(_ int) swiftui.View {
					paletteMu.Lock()
					items := make([]savedColor, len(palette))
					copy(items, palette)
					paletteMu.Unlock()
					if len(items) == 0 {
						return swiftui.Text("No saved colors").
							ForegroundStyleNamed("secondary").
							Font(swiftui.FontCaption).
							AsView()
					}
					var views []swiftui.Viewable
					for _, c := range items {
						hex := hexString(c.r, c.g, c.b)
						views = append(views,
							swiftui.VStack(
								swiftui.RoundedRectangle(6).
									Fill(swiftui.RGBA(c.r, c.g, c.b, 1)).
									Frame(32, 32).
									AsView(),
								swiftui.Text(hex).
									Font(swiftui.FontCaption2).
									ForegroundStyleNamed("secondary").
									AsView(),
							),
						)
					}
					return swiftui.ScrollView(swiftui.HStackSpaced(6, views...))
				}),
			).Padding(4),
		),

		// Contrast checker
		swiftui.GroupBox("Contrast Checker",
			swiftui.DynamicView(version, func(_ int) swiftui.View {
				r, g, b := rState.Get(), gState.Get(), bState.Get()
				return swiftui.VStackSpaced(6,
					contrastSample("On Color", r, g, b, contrastTextR(r, g, b), contrastTextG(r, g, b), contrastTextB(r, g, b)),
					contrastSample("On White", 1, 1, 1, r, g, b),
					contrastSample("On Black", 0, 0, 0, r, g, b),
				)
			}),
		),

		// Blend modes
		swiftui.GroupBox("Blend Modes",
			swiftui.DynamicView(version, func(_ int) swiftui.View {
				r, g, b := rState.Get(), gState.Get(), bState.Get()
				cr, cg, cb := 0.9, 0.3, 0.2 // fixed comparison color
				return swiftui.VStackSpaced(6,
					swiftui.HStackSpaced(8,
						blendSwatch("Current", r, g, b),
						blendSwatch("Compare", cr, cg, cb),
					),
					swiftui.HStackSpaced(8,
						blendSwatch("Multiply", r*cr, g*cg, b*cb),
						blendSwatch("Screen", screen(r, cr), screen(g, cg), screen(b, cb)),
						blendSwatch("Overlay", overlay(r, cr), overlay(g, cg), overlay(b, cb)),
					),
				)
			}),
		),
	).MaxFrame(-1, 0)
	if err := swiftui.Run(swiftui.App{Windows: []swiftui.WindowConfig{{
		Title:  "Color Laboratory",
		Width:  800,
		Height: 600,
		Root: swiftui.ScrollView(
			swiftui.HStackSpaced(16,
				leftColumn,
				rightColumn,
			).Padding(20),
		),
	}}}); err != nil {
		log.Fatal(err)
	}
}

func hexString(r, g, b float64) string {
	return fmt.Sprintf("#%02X%02X%02X", clampByte(r), clampByte(g), clampByte(b))
}

func clampByte(v float64) uint8 {
	n := int(math.Round(v * 255))
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return uint8(n)
}

// luminance computes relative luminance using the BT.601 formula.
func luminance(r, g, b float64) float64 {
	return 0.299*r + 0.587*g + 0.114*b
}

// contrastRatio returns the approximate WCAG contrast ratio between two luminances.
func contrastRatio(l1, l2 float64) float64 {
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func contrastLabel(fg, bg [3]float64) string {
	ratio := contrastRatio(luminance(fg[0], fg[1], fg[2]), luminance(bg[0], bg[1], bg[2]))
	if ratio >= 7 {
		return "AAA"
	}
	if ratio >= 4.5 {
		return "AA"
	}
	return "Fail"
}

func contrastTextR(r, g, b float64) float64 {
	if luminance(r, g, b) > 0.5 {
		return 0
	}
	return 1
}

func contrastTextG(r, g, b float64) float64 {
	if luminance(r, g, b) > 0.5 {
		return 0
	}
	return 1
}

func contrastTextB(r, g, b float64) float64 {
	if luminance(r, g, b) > 0.5 {
		return 0
	}
	return 1
}

func contrastSample(label string, bgR, bgG, bgB, fgR, fgG, fgB float64) swiftui.View {
	cl := contrastLabel([3]float64{fgR, fgG, fgB}, [3]float64{bgR, bgG, bgB})
	cr, cg, cb, ca := contrastColor(cl)
	return swiftui.HStack(
		swiftui.RoundedRectangle(6).
			Fill(swiftui.RGBA(bgR, bgG, bgB, 1)).
			Frame(80, 30).
			AsView(),
		swiftui.Text(fmt.Sprintf("%s: %s", label, cl)).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightBold).
			ForegroundStyle(swiftui.RGBA(cr, cg, cb, ca)),
		swiftui.Spacer(),
	)
}

func contrastColor(label string) (float64, float64, float64, float64) {
	switch label {
	case "AAA":
		return 0.2, 0.8, 0.3, 1
	case "AA":
		return 0.9, 0.7, 0.1, 1
	default:
		return 0.9, 0.3, 0.3, 1
	}
}

// HSL/RGB conversions for color theory calculations.
func rgbToHSL(r, g, b float64) (h, s, l float64) {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2
	if max == min {
		return 0, 0, l
	}
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h *= 60
	return
}

func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	h /= 360
	r = hueToRGB(p, q, h+1.0/3.0)
	g = hueToRGB(p, q, h)
	b = hueToRGB(p, q, h-1.0/3.0)
	return
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}

func analogous(r, g, b float64) []savedColor {
	h, s, l := rgbToHSL(r, g, b)
	r1, g1, b1 := hslToRGB(math.Mod(h+30+360, 360), s, l)
	r2, g2, b2 := hslToRGB(math.Mod(h-30+360, 360), s, l)
	return []savedColor{{r1, g1, b1}, {r2, g2, b2}}
}

func triadic(r, g, b float64) []savedColor {
	h, s, l := rgbToHSL(r, g, b)
	r1, g1, b1 := hslToRGB(math.Mod(h+120, 360), s, l)
	r2, g2, b2 := hslToRGB(math.Mod(h+240, 360), s, l)
	return []savedColor{{r1, g1, b1}, {r2, g2, b2}}
}

func colorRow(label string, colors []savedColor) swiftui.View {
	var views []swiftui.Viewable
	views = append(views,
		swiftui.Text(label).
			Font(swiftui.FontCaption).
			FontWeight(swiftui.WeightMedium).
			Frame(100, 0),
	)
	for _, c := range colors {
		views = append(views,
			swiftui.VStack(
				swiftui.Circle().
					Fill(swiftui.RGBA(c.r, c.g, c.b, 1)).
					Frame(28, 28).
					AsView(),
				swiftui.Text(hexString(c.r, c.g, c.b)).
					Font(swiftui.FontCaption2).
					ForegroundStyleNamed("secondary").
					AsView(),
			),
		)
	}
	return swiftui.HStackSpaced(8, views...)
}

// Blend mode math.

func screen(a, b float64) float64 {
	return 1 - (1-a)*(1-b)
}

func overlay(a, b float64) float64 {
	if a < 0.5 {
		return 2 * a * b
	}
	return 1 - 2*(1-a)*(1-b)
}

func blendSwatch(label string, r, g, b float64) swiftui.View {
	return swiftui.VStack(
		swiftui.RoundedRectangle(6).
			Fill(swiftui.RGBA(r, g, b, 1)).
			Frame(48, 32).
			AsView(),
		swiftui.Text(label).
			Font(swiftui.FontCaption2).
			ForegroundStyleNamed("secondary").
			AsView(),
	)
}

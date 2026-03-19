//go:build darwin
// +build darwin

// Command password-generator demonstrates a password generator and strength
// analyzer built with SwiftUI from Go. It combines reactive state management,
// form controls, and dynamic views to create an interactive security tool.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"
	"strings"
	"sync"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

const (
	sepNone  = 0
	sepDash  = 1
	sepSpace = 2
	sepDot   = 3
)

// historyEntry holds a generated password and its entropy.
type historyEntry struct {
	password string
	entropy  float64
}

var (
	historyMu      sync.Mutex
	historyEntries []historyEntry
)

func addHistory(pw string, entropy float64) {
	historyMu.Lock()
	defer historyMu.Unlock()
	historyEntries = append(historyEntries, historyEntry{pw, entropy})
	if len(historyEntries) > 5 {
		historyEntries = historyEntries[len(historyEntries)-5:]
	}
}

func getHistory() []historyEntry {
	historyMu.Lock()
	defer historyMu.Unlock()
	out := make([]historyEntry, len(historyEntries))
	copy(out, historyEntries)
	return out
}

func generate(length int, upper, lower, digits, symbols int, exclude string, sepType int, sepN int) string {
	var pool []rune
	if upper == 1 {
		pool = append(pool, []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")...)
	}
	if lower == 1 {
		pool = append(pool, []rune("abcdefghijklmnopqrstuvwxyz")...)
	}
	if digits == 1 {
		pool = append(pool, []rune("0123456789")...)
	}
	if symbols == 1 {
		pool = append(pool, []rune("!@#$%^&*()-_=+[]{}|;:',.<>?/`~")...)
	}
	// Remove excluded characters.
	if exclude != "" {
		excl := map[rune]bool{}
		for _, r := range exclude {
			excl[r] = true
		}
		var filtered []rune
		for _, r := range pool {
			if !excl[r] {
				filtered = append(filtered, r)
			}
		}
		pool = filtered
	}
	if len(pool) == 0 {
		return "(no characters available)"
	}
	var b strings.Builder
	sep := ""
	switch sepType {
	case sepDash:
		sep = "-"
	case sepSpace:
		sep = " "
	case sepDot:
		sep = "."
	}
	charCount := 0
	for i := range length {
		_ = i
		if sep != "" && charCount > 0 && charCount%sepN == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(pool[rand.IntN(len(pool))])
		charCount++
	}
	return b.String()
}

func poolSize(upper, lower, digits, symbols int, exclude string) int {
	n := 0
	if upper == 1 {
		n += 26
	}
	if lower == 1 {
		n += 26
	}
	if digits == 1 {
		n += 10
	}
	if symbols == 1 {
		n += 29
	}
	for range exclude {
		n--
	}
	if n < 1 {
		n = 1
	}
	return n
}

func entropy(length int, pool int) float64 {
	if pool <= 1 {
		return 0
	}
	return float64(length) * math.Log2(float64(pool))
}

func strengthLevel(bits float64) int {
	switch {
	case bits >= 80:
		return 3 // excellent
	case bits >= 60:
		return 2 // strong
	case bits >= 40:
		return 1 // fair
	default:
		return 0 // weak
	}
}

func strengthLabel(level int) string {
	switch level {
	case 3:
		return "Excellent"
	case 2:
		return "Strong"
	case 1:
		return "Fair"
	default:
		return "Weak"
	}
}

func strengthColor(level int) (r, g, b float64) {
	switch level {
	case 3:
		return 0.2, 0.8, 0.3
	case 2:
		return 0.9, 0.85, 0.1
	case 1:
		return 1.0, 0.6, 0.1
	default:
		return 0.9, 0.2, 0.2
	}
}

func crackTime(bits float64) string {
	// Assume 10 billion guesses per second.
	if bits <= 0 {
		return "instant"
	}
	secs := math.Pow(2, bits) / 1e10
	switch {
	case secs < 1:
		return "instant"
	case secs < 60:
		return fmt.Sprintf("~%.0f seconds", secs)
	case secs < 3600:
		return fmt.Sprintf("~%.0f minutes", secs/60)
	case secs < 86400:
		return fmt.Sprintf("~%.0f hours", secs/3600)
	case secs < 86400*365:
		return fmt.Sprintf("~%.0f days", secs/86400)
	case secs < 86400*365*1e3:
		return fmt.Sprintf("~%.0f years", secs/(86400*365))
	case secs < 86400*365*1e6:
		return fmt.Sprintf("~%.0fK years", secs/(86400*365*1e3))
	case secs < 86400*365*1e9:
		return fmt.Sprintf("~%.0fM years", secs/(86400*365*1e6))
	default:
		return fmt.Sprintf("~%.0fB years", secs/(86400*365*1e9))
	}
}

func main() {
	// State.
	password := swiftui.NewStringState("")
	copyLabel := swiftui.NewStringState("")
	length := swiftui.NewIntState(16)
	upper := swiftui.NewIntState(1)
	lower := swiftui.NewIntState(1)
	digits := swiftui.NewIntState(1)
	symbols := swiftui.NewIntState(1)
	exclude := swiftui.NewStringState("")
	sepType := swiftui.NewIntState(sepNone)
	sepN := swiftui.NewIntState(4)
	version := swiftui.NewIntState(0) // triggers regeneration
	histVer := swiftui.NewIntState(0) // triggers history rebuild

	regen := func() {
		pw := generate(length.Get(), upper.Get(), lower.Get(), digits.Get(), symbols.Get(),
			exclude.Get(), sepType.Get(), sepN.Get())
		password.Set(pw)
		ps := poolSize(upper.Get(), lower.Get(), digits.Get(), symbols.Get(), exclude.Get())
		ent := entropy(length.Get(), ps)
		addHistory(pw, ent)
		version.Set(version.Get() + 1)
		histVer.Set(histVer.Get() + 1)
		copyLabel.Set("")
	}

	// Generate initial password.
	regen()

	onChange := func() { regen() }

	swiftui.Run(swiftui.AppConfig{
		Title:  "Password Generator",
		Width:  560,
		Height: 700,
	}, swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			// -- Password display --
			swiftui.GroupBox("Generated Password",
				swiftui.VStackSpaced(10,
					swiftui.TextFromString(password).
						Font(swiftui.FontSystemDesign(18, swiftui.WeightMedium, swiftui.DesignMonospaced)).
						AsView().
						MaxFrame(-1, 0).
						Padding(8),
					swiftui.SecureField("Password", password, func() {}),
					swiftui.HStackSpaced(12,
						swiftui.Button("Regenerate", func() {
							regen()
						}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
							ControlSize(swiftui.ControlSizeLarge),
						swiftui.Button("Copy", func() {
							copyLabel.Set("Copied!")
						}).ButtonStyle(swiftui.ButtonStyleBordered).
							ControlSize(swiftui.ControlSizeLarge),
						swiftui.TextFromString(copyLabel).
							ForegroundStyle(0.2, 0.8, 0.3, 1.0).
							Font(swiftui.FontCaption).
							AsView(),
						swiftui.Spacer(),
					),
				).Padding(8),
			).MaxFrame(-1, 0),

			// -- Configuration --
			swiftui.Form(
				swiftui.Section("Length",
					swiftui.HStack(
						swiftui.Slider("", length, 8, 64, onChange),
						swiftui.TextFrom(length).
							Font(swiftui.FontBody).
							FontWeight(swiftui.WeightSemibold).
							MonospacedDigit().
							Frame(30, 0),
					),
				),
				swiftui.Section("Character Types",
					swiftui.VStack(
						swiftui.Toggle("Uppercase (A-Z)", upper, onChange),
						swiftui.Toggle("Lowercase (a-z)", lower, onChange),
						swiftui.Toggle("Digits (0-9)", digits, onChange),
						swiftui.Toggle("Symbols (!@#...)", symbols, onChange),
					),
				),
				swiftui.Section("Exclusions",
					swiftui.TextField("Characters to exclude", exclude, func() { regen() }),
				),
				swiftui.Section("Separator",
					swiftui.VStack(
						swiftui.PickerSegmented("Style", sepType,
							swiftui.VStack(
								swiftui.Text("None").AsView().Tag(0),
								swiftui.Text("Dash").AsView().Tag(1),
								swiftui.Text("Space").AsView().Tag(2),
								swiftui.Text("Dot").AsView().Tag(3),
							), onChange),
						swiftui.Stepper("Every N chars", sepN, 3, 6, onChange),
					),
				),
			).Frame(500, 300),

			// -- Strength meter --
			swiftui.GroupBox("Strength",
				swiftui.DynamicView(version, func(_ int) swiftui.View {
					ps := poolSize(upper.Get(), lower.Get(), digits.Get(), symbols.Get(), exclude.Get())
					bits := entropy(length.Get(), ps)
					level := strengthLevel(bits)
					label := strengthLabel(level)
					cr, cg, cb := strengthColor(level)
					crack := crackTime(bits)

					bars := make([]swiftui.Viewable, 4)
					for i := range 4 {
						if i <= level {
							bars[i] = swiftui.RoundedRectangle(3).
								Fill(cr, cg, cb, 1.0).
								Frame(112, 8).
								AsView().MaxFrame(-1, 0)
						} else {
							bars[i] = swiftui.RoundedRectangle(3).
								Fill(0.7, 0.7, 0.7, 0.3).
								Frame(112, 8).
								AsView().MaxFrame(-1, 0)
						}
					}

					return swiftui.VStackSpaced(8,
						swiftui.HStackSpaced(4, bars...),
						swiftui.HStack(
							swiftui.Text(label).
								Font(swiftui.FontBody).
								FontWeight(swiftui.WeightSemibold).
								ForegroundStyle(cr, cg, cb, 1.0).
								AsView(),
							swiftui.Spacer(),
							swiftui.Text(fmt.Sprintf("%.0f bits", bits)).
								Font(swiftui.FontCaption).
								ForegroundStyleNamed("secondary").
								AsView(),
						),
						swiftui.HStack(
							swiftui.Text("Time to crack:").
								Font(swiftui.FontCaption).
								ForegroundStyleNamed("secondary").
								AsView(),
							swiftui.Spacer(),
							swiftui.Text(crack).
								Font(swiftui.FontCaption).
								FontWeight(swiftui.WeightMedium).
								AsView(),
						),
					)
				}).Padding(8),
			).MaxFrame(-1, 0),

			// -- History --
			swiftui.GroupBox("History",
				swiftui.DynamicView(histVer, func(_ int) swiftui.View {
					entries := getHistory()
					if len(entries) == 0 {
						return swiftui.Text("No passwords generated yet.").
							ForegroundStyleNamed("secondary").
							Font(swiftui.FontCaption).
							AsView()
					}
					rows := make([]swiftui.Viewable, len(entries))
					for i, e := range entries {
						display := e.password
						if len(display) > 24 {
							display = display[:24] + "..."
						}
						level := strengthLevel(e.entropy)
						dr, dg, db := strengthColor(level)
						rows[i] = swiftui.HStack(
							swiftui.Text(display).
								Font(swiftui.FontSystemDesign(12, swiftui.WeightRegular, swiftui.DesignMonospaced)).
								AsView(),
							swiftui.Spacer(),
							swiftui.Circle().
								Fill(dr, dg, db, 1.0).
								Frame(8, 8).
								AsView(),
						)
					}
					return swiftui.List(rows...)
				}).Frame(500, 150),
			).MaxFrame(-1, 0),
		).Padding(20),
	))
}

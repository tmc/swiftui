//go:build darwin
// +build darwin

// Command calculator demonstrates a working calculator app built with SwiftUI from Go.
//
// It renders a 4x5 grid of buttons for digits, operators, and functions,
// with a reactive display using StringState. All arithmetic logic is
// implemented as a pure Go state machine.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"math"
	"runtime"
	"strconv"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

type calc struct {
	display   *swiftui.StringState
	firstOp   float64
	pendingOp string
	newDigit  bool
}

func (c *calc) inputDigit(d string) {
	cur := c.display.Get()
	if c.newDigit || cur == "0" {
		c.display.Set(d)
		c.newDigit = false
	} else {
		c.display.Set(cur + d)
	}
}

func (c *calc) inputDot() {
	cur := c.display.Get()
	if c.newDigit {
		c.display.Set("0.")
		c.newDigit = false
		return
	}
	for _, ch := range cur {
		if ch == '.' {
			return
		}
	}
	c.display.Set(cur + ".")
}

func (c *calc) inputOp(op string) {
	c.evaluate()
	c.firstOp = c.displayVal()
	c.pendingOp = op
	c.newDigit = true
}

func (c *calc) evaluate() {
	if c.pendingOp == "" {
		return
	}
	second := c.displayVal()
	var result float64
	switch c.pendingOp {
	case "+":
		result = c.firstOp + second
	case "-":
		result = c.firstOp - second
	case "×":
		result = c.firstOp * second
	case "÷":
		if second == 0 {
			c.display.Set("Error")
			c.pendingOp = ""
			c.newDigit = true
			return
		}
		result = c.firstOp / second
	}
	c.pendingOp = ""
	c.display.Set(formatNumber(result))
	c.newDigit = true
}

func (c *calc) clear() {
	c.display.Set("0")
	c.firstOp = 0
	c.pendingOp = ""
	c.newDigit = false
}

func (c *calc) toggleSign() {
	v := c.displayVal()
	c.display.Set(formatNumber(-v))
}

func (c *calc) percent() {
	v := c.displayVal()
	c.display.Set(formatNumber(v / 100))
}

func (c *calc) displayVal() float64 {
	v, err := strconv.ParseFloat(c.display.Get(), 64)
	if err != nil {
		return 0
	}
	return v
}

func formatNumber(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return "Error"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return fmt.Sprintf("%g", v)
	}
	return strconv.FormatFloat(v, 'g', 10, 64)
}

func main() {
	c := &calc{
		display:  swiftui.NewStringState("0"),
		newDigit: false,
	}

	// Button builder helpers.
	numBtn := func(label string) swiftui.View {
		l := label
		return swiftui.Button(l, func() { c.inputDigit(l) }).
			ControlSize(swiftui.ControlSizeLarge).
			ButtonStyle(swiftui.ButtonStyleBordered).
			MaxFrame(-1, 0)
	}
	opBtn := func(label string) swiftui.View {
		l := label
		return swiftui.Button(l, func() { c.inputOp(l) }).
			ControlSize(swiftui.ControlSizeLarge).
			ButtonStyle(swiftui.ButtonStyleBorderedProminent).
			MaxFrame(-1, 0)
	}
	funcBtn := func(label string, action func()) swiftui.View {
		return swiftui.Button(label, action).
			ControlSize(swiftui.ControlSizeLarge).
			ButtonStyle(swiftui.ButtonStyleBordered).
			ForegroundStyleNamed("secondary").
			MaxFrame(-1, 0)
	}

	swiftui.Run(swiftui.AppConfig{
		Title:  "Calculator",
		Width:  320,
		Height: 450,
	}, swiftui.VStackSpaced(14,
		swiftui.HStack(
			swiftui.VStackSpaced(4,
				swiftui.HStack(
					swiftui.Text("Calculator").
						Font(swiftui.FontTitle3).
						FontWeight(swiftui.WeightBold),
					swiftui.Spacer(),
				),
				swiftui.HStack(
					swiftui.Text("Standard arithmetic with reactive Go state.").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				),
			).MaxFrame(-1, 0),
		),

		swiftui.VStackSpaced(8,
			swiftui.HStack(
				swiftui.Spacer(),
				swiftui.Text("Result").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary"),
			),
			swiftui.HStack(
				swiftui.Spacer(),
				swiftui.TextFromString(c.display).
					Font(swiftui.FontSystem(52)).
					FontWeight(swiftui.WeightLight).
					MonospacedDigit(),
			),
		).Padding(16).
			Background(0.18, 0.19, 0.22, 0.55).
			CornerRadius(18),

		swiftui.VStackSpaced(8,
			swiftui.HStackSpaced(8,
				funcBtn("C", c.clear),
				funcBtn("±", c.toggleSign),
				funcBtn("%", c.percent),
				opBtn("÷"),
			),
			swiftui.HStackSpaced(8,
				numBtn("7"), numBtn("8"), numBtn("9"),
				opBtn("×"),
			),
			swiftui.HStackSpaced(8,
				numBtn("4"), numBtn("5"), numBtn("6"),
				opBtn("-"),
			),
			swiftui.HStackSpaced(8,
				numBtn("1"), numBtn("2"), numBtn("3"),
				opBtn("+"),
			),
			swiftui.HStackSpaced(8,
				numBtn("0"),
				swiftui.Button(".", func() { c.inputDot() }).
					ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBordered).
					MaxFrame(-1, 0),
				swiftui.Button("=", func() { c.evaluate() }).
					ControlSize(swiftui.ControlSizeLarge).
					ButtonStyle(swiftui.ButtonStyleBorderedProminent).
					Tint(0.2, 0.6, 1.0, 1.0).
					MaxFrame(-1, 0),
			),
		),
	).Padding(18))
}

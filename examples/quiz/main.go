//go:build darwin
// +build darwin

// Command quiz demonstrates an interactive Go trivia game with animated
// transitions, scoring, and results using SwiftUI from Go.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

type question struct {
	text    string
	options [4]string
	answer  int // correct option index
}

var questions = []question{
	{
		text:    "What keyword starts a new goroutine?",
		options: [4]string{"async", "go", "spawn", "thread"},
		answer:  1,
	},
	{
		text:    "Which type is used for communication between goroutines?",
		options: [4]string{"pipe", "mutex", "channel", "queue"},
		answer:  2,
	},
	{
		text:    "What is the zero value of a slice?",
		options: [4]string{"[]", "empty slice", "nil", "undefined"},
		answer:  2,
	},
	{
		text:    "How many values can a map lookup return?",
		options: [4]string{"1", "2", "3", "depends on the map"},
		answer:  1,
	},
	{
		text:    "What does defer do?",
		options: [4]string{"Cancels a function", "Delays execution until return", "Runs in a goroutine", "Skips the next line"},
		answer:  1,
	},
	{
		text:    "Which interface has zero methods?",
		options: [4]string{"any", "io.Reader", "error", "fmt.Stringer"},
		answer:  0,
	},
	{
		text:    "What package provides formatted I/O?",
		options: [4]string{"io", "os", "fmt", "bufio"},
		answer:  2,
	},
	{
		text:    "When does init() run?",
		options: [4]string{"When called explicitly", "After main()", "Before main()", "On goroutine start"},
		answer:  2,
	},
	{
		text:    "What is the error interface method?",
		options: [4]string{"Err() string", "Error() string", "String() error", "Message() string"},
		answer:  1,
	},
	{
		text:    "How do you make a buffered channel of size 5?",
		options: [4]string{"make(chan int, 5)", "chan(int, 5)", "new(chan int, 5)", "chan int{5}"},
		answer:  0,
	},
}

func main() {
	screen := swiftui.NewIntState(0)      // 0=start, 1=playing, 2=results
	questionIdx := swiftui.NewIntState(0) // current question
	score := swiftui.NewIntState(0)       // total correct
	selected := swiftui.NewIntState(-1)   // selected answer (-1 = none)

	total := len(questions)

	// Track per-question correctness: encode as bitmask in an IntState.
	correctMask := swiftui.NewIntState(0)

	var mu sync.Mutex
	answering := false

	swiftui.Run(swiftui.AppConfig{
		Title:  "Go Quiz",
		Width:  500,
		Height: 650,
	}, swiftui.AnimatedDynamicView(screen, swiftui.TransitionPush, func(s int) swiftui.View {
		switch s {
		case 0:
			return startScreen(screen)
		case 1:
			return playScreen(questionIdx, score, selected, screen, correctMask, total, &mu, &answering)
		default:
			return resultsScreen(score, correctMask, screen, questionIdx, selected, total, &mu, &answering)
		}
	}))
}

func startScreen(screen *swiftui.IntState) swiftui.View {
	return swiftui.VStackSpaced(16,
		swiftui.Spacer(),
		swiftui.Image("brain.head.profile").
			ForegroundStyle(0.3, 0.6, 1.0, 1.0).
			ImageScale(swiftui.ImageScaleLarge),
		swiftui.Text("Go Quiz").
			Font(swiftui.FontLargeTitle).
			FontWeight(swiftui.WeightBold),
		swiftui.Text("Test your Go knowledge").
			Font(swiftui.FontBody).
			ForegroundStyleNamed("secondary"),
		swiftui.Label(fmt.Sprintf("%d Questions", len(questions)), "list.number").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").AsView(),
		swiftui.Spacer(),
		swiftui.Button("Start Quiz", func() {
			screen.SetAnimated(1)
		}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
			ControlSize(swiftui.ControlSizeLarge),
		swiftui.Spacer(),
	).Padding(30)
}

func playScreen(
	questionIdx, score, selected, screen, correctMask *swiftui.IntState,
	total int,
	mu *sync.Mutex, answering *bool,
) swiftui.View {
	return swiftui.DynamicView(questionIdx, func(qi int) swiftui.View {
		if qi >= total {
			screen.SetAnimated(2)
			return swiftui.EmptyView()
		}
		q := questions[qi]
		return swiftui.DynamicView(selected, func(sel int) swiftui.View {
			answered := sel >= 0
			return swiftui.VStackSpaced(12,
				// Progress bar
				swiftui.ProgressLinear(float64(qi+1), float64(total)).
					Tint(0.3, 0.6, 1.0, 1.0),

				// Question header
				swiftui.HStack(
					swiftui.Text(fmt.Sprintf("Question %d of %d", qi+1, total)).
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
					swiftui.Text(fmt.Sprintf("Score: %d/%d", score.Get(), qi)).
						Font(swiftui.FontCaption).
						FontWeight(swiftui.WeightMedium),
				),

				// Question text
				swiftui.Text(q.text).
					Font(swiftui.FontTitle2).
					FontWeight(swiftui.WeightSemibold).
					AsView().
					MaxFrame(-1, 0),

				swiftui.Spacer(),

				// Answer buttons
				answerButton(q, 0, sel, answered, q.answer, score, selected, questionIdx, correctMask, qi, total, mu, answering),
				answerButton(q, 1, sel, answered, q.answer, score, selected, questionIdx, correctMask, qi, total, mu, answering),
				answerButton(q, 2, sel, answered, q.answer, score, selected, questionIdx, correctMask, qi, total, mu, answering),
				answerButton(q, 3, sel, answered, q.answer, score, selected, questionIdx, correctMask, qi, total, mu, answering),

				swiftui.Spacer(),
			).Padding(30)
		})
	})
}

func answerButton(
	q question, idx, sel int, answered bool, correct int,
	score, selected, questionIdx, correctMask *swiftui.IntState,
	qi, total int,
	mu *sync.Mutex, answering *bool,
) swiftui.View {
	label := fmt.Sprintf("  %s  %s", string(rune('A'+idx)), q.options[idx])

	// Determine appearance based on answer state
	if answered {
		if idx == correct {
			label = "  \u2713  " + q.options[idx]
		} else if idx == sel {
			label = "  \u2717  " + q.options[idx]
		}
	}

	btn := swiftui.Button(label, func() {
		mu.Lock()
		if *answering {
			mu.Unlock()
			return
		}
		*answering = true
		mu.Unlock()

		selected.Set(idx)
		if idx == correct {
			score.Set(score.Get() + 1)
			correctMask.Set(correctMask.Get() | (1 << qi))
		}

		go func() {
			time.Sleep(1500 * time.Millisecond)
			selected.Set(-1)
			questionIdx.SetAnimated(qi + 1)
			mu.Lock()
			*answering = false
			mu.Unlock()
		}()
	}).ButtonStyle(swiftui.ButtonStyleBordered).
		ControlSize(swiftui.ControlSizeLarge).
		MaxFrame(-1, 0).
		Disabled(answered)

	if answered && idx == correct {
		btn = btn.ForegroundStyle(0.2, 0.75, 0.3, 1.0)
	} else if answered && idx == sel {
		btn = btn.ForegroundStyle(0.9, 0.25, 0.2, 1.0)
	}

	return btn
}

func resultsScreen(
	score, correctMask, screen, questionIdx, selected *swiftui.IntState,
	total int,
	mu *sync.Mutex, answering *bool,
) swiftui.View {
	s := score.Get()
	pct := float64(s) / float64(total)
	grade := gradeFor(pct)
	pctText := fmt.Sprintf("%d%%", int(pct*100))

	// Build question result list
	resultRows := make([]swiftui.Viewable, total)
	mask := correctMask.Get()
	for i := 0; i < total; i++ {
		correct := (mask & (1 << i)) != 0
		dot := "\u25cf" // filled circle
		var row swiftui.View
		if correct {
			row = swiftui.HStack(
				swiftui.Text(dot).ForegroundStyle(0.2, 0.75, 0.3, 1.0).AsView(),
				swiftui.Text(fmt.Sprintf(" Q%d: %s", i+1, questions[i].text)).
					Font(swiftui.FontCaption).AsView(),
				swiftui.Spacer(),
			)
		} else {
			row = swiftui.HStack(
				swiftui.Text(dot).ForegroundStyle(0.9, 0.25, 0.2, 1.0).AsView(),
				swiftui.Text(fmt.Sprintf(" Q%d: %s", i+1, questions[i].text)).
					Font(swiftui.FontCaption).AsView(),
				swiftui.Spacer(),
			)
		}
		resultRows[i] = row
	}

	return swiftui.ScrollView(
		swiftui.VStackSpaced(16,
			swiftui.Spacer(),

			// Score circle
			swiftui.ZStack(
				swiftui.Circle().
					Stroke(0.85, 0.85, 0.85, 0.3, 8).
					Frame(120, 120).
					AsView(),
				swiftui.Circle().
					Stroke(scoreColor(pct, 0), scoreColor(pct, 1), scoreColor(pct, 2), 1.0, 8).
					Frame(120, 120).
					AsView(),
				swiftui.VStack(
					swiftui.Text(pctText).
						Font(swiftui.FontTitle).
						FontWeight(swiftui.WeightBold),
					swiftui.Text(grade).
						Font(swiftui.FontLargeTitle).
						FontWeight(swiftui.WeightHeavy).
						ForegroundStyle(scoreColor(pct, 0), scoreColor(pct, 1), scoreColor(pct, 2), 1.0),
				),
			),

			swiftui.Text(fmt.Sprintf("%d out of %d correct", s, total)).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightSemibold),

			swiftui.Divider(),

			// Question breakdown
			swiftui.Text("Results").
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightMedium),

			swiftui.VStackSpaced(4, resultRows...),

			swiftui.Spacer(),

			// Play again button
			swiftui.Button("Play Again", func() {
				score.Set(0)
				correctMask.Set(0)
				questionIdx.Set(0)
				selected.Set(-1)
				mu.Lock()
				*answering = false
				mu.Unlock()
				screen.SetAnimated(0)
			}).ButtonStyle(swiftui.ButtonStyleBorderedProminent).
				ControlSize(swiftui.ControlSizeLarge),

			swiftui.Spacer(),
		).Padding(30),
	)
}

func gradeFor(pct float64) string {
	switch {
	case pct >= 0.9:
		return "A"
	case pct >= 0.8:
		return "B"
	case pct >= 0.7:
		return "C"
	case pct >= 0.6:
		return "D"
	default:
		return "F"
	}
}

func scoreColor(pct float64, component int) float64 {
	// Green for high, orange for mid, red for low
	colors := [3][3]float64{
		{0.2, 0.75, 0.3},   // >= 70%
		{0.95, 0.65, 0.15}, // >= 40%
		{0.9, 0.25, 0.2},   // < 40%
	}
	idx := 0
	if pct < 0.7 {
		idx = 1
	}
	if pct < 0.4 {
		idx = 2
	}
	return colors[idx][component]
}

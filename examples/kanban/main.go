//go:build darwin
// +build darwin

// Command kanban demonstrates a three-column Kanban board with card management.
//
// Cards can be created with title, description, and priority. Cards support
// context menu actions to move between columns or delete. The board updates
// reactively through IntState bindings.
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

type Priority int

const (
	PriorityHigh Priority = iota
	PriorityMedium
	PriorityLow
)

type Card struct {
	Title       string
	Description string
	Priority    Priority
	CreatedAt   time.Time
}

var (
	mu      sync.Mutex
	columns = [3][]Card{
		{ // To Do
			{Title: "Design mockups", Description: "Create wireframes for new feature", Priority: PriorityHigh, CreatedAt: time.Now()},
			{Title: "Write tests", Description: "Add unit tests for auth module", Priority: PriorityMedium, CreatedAt: time.Now()},
		},
		{ // In Progress
			{Title: "API endpoints", Description: "Implement REST handlers", Priority: PriorityHigh, CreatedAt: time.Now()},
		},
		{ // Done
			{Title: "Setup CI", Description: "Configure GitHub Actions", Priority: PriorityLow, CreatedAt: time.Now()},
		},
	}
)

func priorityColor(p Priority) (r, g, b float64) {
	switch p {
	case PriorityHigh:
		return 0.9, 0.3, 0.3
	case PriorityMedium:
		return 0.9, 0.6, 0.2
	default:
		return 0.3, 0.8, 0.4
	}
}

func priorityLabel(p Priority) string {
	switch p {
	case PriorityHigh:
		return "High"
	case PriorityMedium:
		return "Medium"
	default:
		return "Low"
	}
}

var columnNames = [3]string{"To Do", "In Progress", "Done"}

func main() {
	// Version counters drive DynamicView rebuilds per column.
	versions := [3]*swiftui.IntState{
		swiftui.NewIntState(0),
		swiftui.NewIntState(0),
		swiftui.NewIntState(0),
	}

	// Sheet state: 0=hidden, 1/2/3=show add form for column 0/1/2.
	sheetState := swiftui.NewIntState(0)
	titleInput := swiftui.NewStringState("")
	descInput := swiftui.NewStringState("")
	priorityPick := swiftui.NewIntState(1) // default medium

	// Column header colors.
	headerColors := [3][3]float64{
		{0.3, 0.55, 0.95}, // blue
		{0.9, 0.6, 0.2},   // orange
		{0.3, 0.75, 0.45}, // green
	}

	// Build columns.
	var cols []swiftui.Viewable
	for i := 0; i < 3; i++ {
		col := i
		cols = append(cols, buildColumn(col, headerColors[col], versions, sheetState))
	}

	// Sheet form content.
	sheetContent := swiftui.DynamicView(sheetState, func(val int) swiftui.View {
		if val == 0 {
			return swiftui.Text("").AsView().Frame(0, 0)
		}
		colIdx := val - 1
		return swiftui.VStackSpaced(16,
			swiftui.Text(fmt.Sprintf("Add Card to %s", columnNames[colIdx])).
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.TextField("Title", titleInput, func() {}).
				TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
			swiftui.TextField("Description", descInput, func() {}).
				TextFieldStyle(swiftui.TextFieldStyleRoundedBorder),
			swiftui.PickerSegmented("Priority", priorityPick,
				swiftui.VStack(
					swiftui.Text("High").AsView().Tag(0),
					swiftui.Text("Medium").AsView().Tag(1),
					swiftui.Text("Low").AsView().Tag(2),
				),
				func() {},
			),
			swiftui.HStackSpaced(12,
				swiftui.Button("Cancel", func() {
					sheetState.Set(0)
				}).ButtonStyle(swiftui.ButtonStyleBordered),
				swiftui.Button("Create", func() {
					t := titleInput.Get()
					if t == "" {
						return
					}
					d := descInput.Get()
					p := Priority(priorityPick.Get())
					mu.Lock()
					columns[colIdx] = append(columns[colIdx], Card{
						Title:       t,
						Description: d,
						Priority:    p,
						CreatedAt:   time.Now(),
					})
					mu.Unlock()
					titleInput.Set("")
					descInput.Set("")
					priorityPick.Set(1)
					versions[colIdx].Set(versions[colIdx].Get() + 1)
					sheetState.Set(0)
				}).ButtonStyle(swiftui.ButtonStyleBorderedProminent),
			),
			swiftui.Spacer(),
		).Padding(24).Frame(350, 300)
	})

	// Overall stats bar.
	totalState := swiftui.NewIntState(0)
	statsView := swiftui.DynamicView(totalState, func(_ int) swiftui.View {
		mu.Lock()
		counts := [3]int{len(columns[0]), len(columns[1]), len(columns[2])}
		total := counts[0] + counts[1] + counts[2]
		mu.Unlock()
		return swiftui.HStack(
			swiftui.Label(fmt.Sprintf("%d cards total", total), "rectangle.stack.fill").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.Spacer(),
			swiftui.Label(fmt.Sprintf("%d to do", counts[0]), "circle.fill").
				Font(swiftui.FontCaption).
				ForegroundStyle(0.3, 0.55, 0.95, 1),
			swiftui.Label(fmt.Sprintf("%d active", counts[1]), "circle.fill").
				Font(swiftui.FontCaption).
				ForegroundStyle(0.9, 0.6, 0.2, 1),
			swiftui.Label(fmt.Sprintf("%d done", counts[2]), "circle.fill").
				Font(swiftui.FontCaption).
				ForegroundStyle(0.3, 0.75, 0.45, 1),
		).Padding(12)
	})

	// Refresh stats whenever any column changes.
	refreshStats := func() {
		totalState.Set(totalState.Get() + 1)
	}
	// Wire column version changes to stats refresh via wrapper.
	// We do this by wrapping the version bump in a helper.
	_ = refreshStats // used below in buildColumn via closure

	// The main layout.
	board := swiftui.VStack(
		// Title bar
		swiftui.HStack(
			swiftui.Label("Kanban Board", "square.grid.3x1.below.line.grid.1x2.fill").
				Font(swiftui.FontTitle2).
				FontWeight(swiftui.WeightBold),
			swiftui.Spacer(),
		).Padding(16),
		swiftui.Divider(),
		// Board columns
		swiftui.HStackSpaced(12, cols...).Padding(12).MaxFrame(-1, -1),
		swiftui.Divider(),
		// Stats footer
		statsView,
	).Sheet(sheetState, sheetContent)

	// Patch version bumps to also refresh stats.
	for i := 0; i < 3; i++ {
		origVer := versions[i]
		_ = origVer
	}
	// Use a goroutine to poll version changes for stats refresh.
	go func() {
		var last [3]int
		for {
			time.Sleep(100 * time.Millisecond)
			changed := false
			for i := 0; i < 3; i++ {
				v := versions[i].Get()
				if v != last[i] {
					last[i] = v
					changed = true
				}
			}
			if changed {
				refreshStats()
			}
		}
	}()

	swiftui.Run(swiftui.AppConfig{
		Title:  "Kanban Board",
		Width:  900,
		Height: 600,
	}, board)
}

func buildColumn(col int, hc [3]float64, versions [3]*swiftui.IntState, sheetState *swiftui.IntState) swiftui.View {
	header := swiftui.VStack(
		swiftui.HStack(
			swiftui.Circle().Fill(hc[0], hc[1], hc[2], 1).Frame(10, 10).AsView(),
			swiftui.Text(columnNames[col]).
				Font(swiftui.FontHeadline).
				FontWeight(swiftui.WeightSemibold),
			swiftui.Spacer(),
			swiftui.ButtonWithImage("plus.circle.fill", func() {
				sheetState.Set(col + 1) // 1-indexed to distinguish from hidden (0)
			}).Help(fmt.Sprintf("Add card to %s", columnNames[col])),
		),
	).Padding(10).Background(hc[0], hc[1], hc[2], 0.12).CornerRadius(8)

	cardList := swiftui.ScrollView(
		swiftui.AnimatedDynamicView(versions[col], swiftui.TransitionOpacity, func(_ int) swiftui.View {
			mu.Lock()
			snapshot := make([]Card, len(columns[col]))
			copy(snapshot, columns[col])
			mu.Unlock()

			if len(snapshot) == 0 {
				return swiftui.VStack(
					swiftui.Spacer(),
					swiftui.Image("tray").
						ForegroundStyleNamed("secondary").
						ImageScale(swiftui.ImageScaleLarge),
					swiftui.Text("No cards").
						Font(swiftui.FontCaption).
						ForegroundStyleNamed("secondary"),
					swiftui.Spacer(),
				).Padding(20)
			}

			var rows []swiftui.Viewable
			for ci, card := range snapshot {
				rows = append(rows, buildCard(card, col, ci, versions))
			}
			return swiftui.VStackSpaced(8, rows...)
		}),
	)

	return swiftui.VStack(
		header,
		cardList,
	).MaxFrame(-1, -1)
}

func buildCard(card Card, col, idx int, versions [3]*swiftui.IntState) swiftui.View {
	pr, pg, pb := priorityColor(card.Priority)
	timeStr := card.CreatedAt.Format("3:04 PM")

	cardView := swiftui.VStackSpaced(4,
		swiftui.HStack(
			swiftui.Text(card.Title).
				Font(swiftui.FontBody).
				FontWeight(swiftui.WeightSemibold),
			swiftui.Spacer(),
			swiftui.Circle().Fill(pr, pg, pb, 1).Frame(8, 8).AsView(),
		),
		swiftui.Text(card.Description).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
		swiftui.HStack(
			swiftui.Text(priorityLabel(card.Priority)).
				Font(swiftui.FontCaption2).
				ForegroundStyle(pr, pg, pb, 1),
			swiftui.Spacer(),
			swiftui.Label(timeStr, "clock").
				Font(swiftui.FontCaption2).
				ForegroundStyleNamed("tertiary"),
		),
	).Padding(10).
		BackgroundStyle("regularMaterial").
		CornerRadius(8).
		Shadow(0, 0, 0, 0.15, 3, 0, 1)

	// Build context menu actions.
	var menuItems []swiftui.Viewable
	if col < 2 {
		targetCol := col + 1
		cardIdx := idx
		menuItems = append(menuItems,
			swiftui.Button(fmt.Sprintf("Move to %s →", columnNames[targetCol]), func() {
				moveCard(col, targetCol, cardIdx, versions)
			}),
		)
	}
	if col > 0 {
		targetCol := col - 1
		cardIdx := idx
		menuItems = append(menuItems,
			swiftui.Button(fmt.Sprintf("← Move to %s", columnNames[targetCol]), func() {
				moveCard(col, targetCol, cardIdx, versions)
			}),
		)
	}
	cardIdx := idx
	menuItems = append(menuItems,
		swiftui.Divider(),
		swiftui.Button("Delete", func() {
			mu.Lock()
			if cardIdx < len(columns[col]) {
				columns[col] = append(columns[col][:cardIdx], columns[col][cardIdx+1:]...)
			}
			mu.Unlock()
			versions[col].Set(versions[col].Get() + 1)
		}).ForegroundStyle(0.9, 0.3, 0.3, 1),
	)

	contextMenu := swiftui.VStack(menuItems...)

	if col == 2 {
		return cardView.Opacity(0.7).ContextMenu(contextMenu)
	}
	return cardView.ContextMenu(contextMenu)
}

func moveCard(from, to, idx int, versions [3]*swiftui.IntState) {
	mu.Lock()
	if idx >= len(columns[from]) {
		mu.Unlock()
		return
	}
	card := columns[from][idx]
	columns[from] = append(columns[from][:idx], columns[from][idx+1:]...)
	columns[to] = append(columns[to], card)
	mu.Unlock()
	versions[from].Set(versions[from].Get() + 1)
	versions[to].Set(versions[to].Get() + 1)
}

// Command a2ui-server is a demo HTTP server that streams A2UI component
// trees as Server-Sent Events.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/A2UI/agent_sdks/go/a2uibuild"
	"github.com/tmc/swiftui/a2ui"
)

const (
	surfaceID         = "task-tracker"
	catalogID         = a2ui.BasicCatalogID
	showcaseSurfaceID = "a2ui-showcase"
	showcaseCatalogID = a2ui.BasicCatalogID
)

type task struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type settings struct {
	SortOrder     string `json:"sortOrder"`
	FontSize      int    `json:"fontSize"`
	ShowCompleted bool   `json:"showCompleted"`
	Deadline      string `json:"deadline"`
}

type showcaseState struct {
	Name      string   `json:"name"`
	Count     string   `json:"count"`
	Secret    string   `json:"secret"`
	Notes     string   `json:"notes"`
	Enabled   bool     `json:"enabled"`
	Priority  string   `json:"priority"`
	Tags      []string `json:"tags"`
	Deadline  string   `json:"deadline"`
	Intensity int      `json:"intensity"`
	Status    string   `json:"status"`
	ImageURL  string   `json:"imageURL"`
	VideoURL  string   `json:"videoURL"`
	AudioURL  string   `json:"audioURL"`
	ContactEmail string `json:"contactEmail"`
}

type server struct {
	mu        sync.Mutex
	mode      string
	tasks     []task
	nextID    int
	inputText string
	settings  settings
	showcase  showcaseState
	clients   map[chan a2ui.ServerMessage]struct{}
}

func newServer(mode string) *server {
	return &server{
		mode: mode,
		tasks: []task{
			{ID: 1, Text: "Read the A2UI docs"},
			{ID: 2, Text: "Build a demo app"},
			{ID: 3, Text: "Ship it", Done: true},
		},
		nextID: 4,
		settings: settings{
			SortOrder:     "By Date",
			FontSize:      14,
			ShowCompleted: true,
		},
		showcase: showcaseState{
			Name:      "A2UI Showcase",
			Count:     "7",
			Secret:    "swordfish",
			Notes:     "This surface exercises the current macOS renderer coverage.",
			Enabled:   true,
			Priority:  "Medium",
			Tags:      []string{"Renderer", "Variants"},
			Deadline:  time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
			Intensity: 64,
			Status:    "Ready to render the full component catalog.",
			ImageURL:  "https://picsum.photos/320/200",
			VideoURL:  "file:///System/Library/Desktop%20Pictures/.wallpapers/Tahoe%20Day/Tahoe%20Day.mov",
			AudioURL:  "file:///System/Library/Sounds/Glass.aiff",
			ContactEmail: "demo@example.com",
		},
		clients: make(map[chan a2ui.ServerMessage]struct{}),
	}
}

func (s *server) addClient(ch chan a2ui.ServerMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[ch] = struct{}{}
}

func (s *server) removeClient(ch chan a2ui.ServerMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, ch)
	close(ch)
}

func (s *server) broadcast(msg a2ui.ServerMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *server) completionPct() int {
	if len(s.tasks) == 0 {
		return 0
	}
	done := 0
	for _, t := range s.tasks {
		if t.Done {
			done++
		}
	}
	return done * 100 / len(s.tasks)
}

func (s *server) statusText() string {
	if s.mode == "showcase" {
		return fmt.Sprintf("%s priority, intensity %d", s.showcase.Priority, s.showcase.Intensity)
	}
	done := 0
	for _, t := range s.tasks {
		if t.Done {
			done++
		}
	}
	return fmt.Sprintf("%d tasks, %d completed", len(s.tasks), done)
}

func (s *server) currentSurfaceID() string {
	if s.mode == "showcase" {
		return showcaseSurfaceID
	}
	return surfaceID
}

func (s *server) currentCatalogID() string {
	if s.mode == "showcase" {
		return showcaseCatalogID
	}
	return catalogID
}

func (s *server) buildComponents() []a2ui.Component {
	if s.mode == "showcase" {
		return s.buildShowcaseComponents()
	}
	return s.buildTaskComponents()
}

func (s *server) buildTaskComponents() []a2ui.Component {
	var components []a2ui.Component

	components = append(components, tabs("root",
		tab("Tasks", "tasks-tab"),
		tab("Settings", "settings-tab"),
	))

	components = append(components, column("tasks-tab",
		[]string{"title", "input-row", "divider-1", "task-list", "divider-2", "progress-card"},
		12, a2ui.LayoutAlignStart,
	))
	components[len(components)-1].Padding = float64Ptr(16)

	components = append(components, text("title", "Task Tracker", a2ui.TextVariantH2))

	components = append(components, row("input-row",
		[]string{"input-field", "add-btn"},
		8, a2ui.LayoutAlignCenter, a2ui.LayoutJustifyStart,
	))

	components = append(components,
		textField("input-field", "New task...", "/inputText", action("add_task", nil), ""),
		button("add-btn", "add-btn-label", action("add_task", nil), a2ui.ButtonVariantPrimary),
		text("add-btn-label", "Add", ""),
		divider("divider-1"),
	)

	taskChildren := make([]string, 0, len(s.tasks))
	for _, t := range s.tasks {
		rowID := fmt.Sprintf("task-%d", t.ID)
		checkID := fmt.Sprintf("task-%d-check", t.ID)
		textID := fmt.Sprintf("task-%d-text", t.ID)
		taskChildren = append(taskChildren, rowID)

		components = append(components,
			row(rowID, []string{checkID, textID}, 8, "", ""),
			checkBox(
				checkID,
				"",
				fmt.Sprintf("/tasks/%d/done", t.ID),
				action("toggle_task", map[string]a2ui.DynamicValue{
					"taskID": a2ui.ValueNumber(float64(t.ID)),
				}),
			),
			struckText(textID, t.Text, t.Done),
		)
	}

	components = append(components,
		list("task-list", taskChildren, a2ui.ListDirectionVertical),
		divider("divider-2"),
		card("progress-card", "progress-content"),
		column("progress-content", []string{"progress-bar", "status-text"}, 8, ""),
		a2ui.ProgressBar("progress-bar", a2ui.NumberBinding("/completionPct"), 100),
		boundText("status-text", "/status", a2ui.TextVariantCaption),
	)

	components = append(components, column("settings-tab",
		[]string{"settings-title", "sort-picker", "font-slider", "show-completed", "deadline-section", "about-modal"},
		12, a2ui.LayoutAlignStart,
	))
	components[len(components)-1].Padding = float64Ptr(16)

	components = append(components,
		text("settings-title", "Settings", a2ui.TextVariantH3),
		choicePicker(
			"sort-picker",
			"Sort Order",
			[]a2ui.ChoiceOption{
				option("By Date"),
				option("By Name"),
				option("By Status"),
			},
			"/settings/sortOrder",
			a2ui.ChoicePickerVariantMutuallyExclusive,
			a2ui.ChoicePickerDisplayStyleChips,
			action("sort_change", nil),
		),
		slider("font-slider", "Font Size", "/settings/fontSize", 10, 24, action("font_change", nil)),
		checkBox("show-completed", "Show Completed Tasks", "/settings/showCompleted", action("show_completed_change", nil)),
		column("deadline-section", []string{"deadline-label", "deadline-input"}, 4, a2ui.LayoutAlignStart),
		text("deadline-label", "Filter by Deadline", a2ui.TextVariantH5),
		dateTimeInput("deadline-input", "Before", "/settings/deadline", true, false, action("deadline_change", nil)),
		modal("about-modal", "about-content", "about-trigger"),
		button("about-trigger", "about-btn-label", action("about", nil), a2ui.ButtonVariantBorderless),
		row("about-btn-label", []string{"about-icon", "about-text"}, 4, a2ui.LayoutAlignCenter, ""),
		icon("about-icon", a2ui.IconInfo),
		text("about-text", "About", ""),
		column("about-content", []string{"about-title", "about-version", "about-desc"}, 8, a2ui.LayoutAlignStart),
		text("about-title", "A2UI Task Tracker", a2ui.TextVariantH2),
		text("about-version", "Version 0.9", a2ui.TextVariantCaption),
		text("about-desc", "A demo of the A2UI v0.9 protocol rendered natively on macOS.", ""),
	)

	return components
}

func (s *server) buildShowcaseComponents() []a2ui.Component {
	var components []a2ui.Component

	components = append(components, tabs("root",
		tab("Overview", "overview-tab"),
		tab("Typography", "typography-tab"),
		tab("Inputs", "inputs-tab"),
		tab("Functions", "functions-tab"),
		tab("Validation", "validation-tab"),
		tab("Templates", "templates-tab"),
		tab("Layout", "layout-tab"),
		tab("Media", "media-tab"),
		tab("Feedback", "feedback-tab"),
		tab("Actions", "actions-tab"),
		tab("Theme", "theme-tab"),
		tab("Dialogs", "dialogs-tab"),
	))

	components = append(components,
		column("overview-tab", []string{"showcase-title", "overview-card", "overview-divider", "support-grid", "catalog-card"}, 14, a2ui.LayoutAlignStart),
		column("typography-tab", []string{"typography-page-title", "typography-card", "copy-card"}, 14, a2ui.LayoutAlignStart),
		column("inputs-tab", []string{"form-title", "form-card"}, 10, a2ui.LayoutAlignStart),
		column("functions-tab", []string{"functions-title", "functions-card"}, 14, a2ui.LayoutAlignStart),
		column("validation-tab", []string{"validation-title", "validation-card"}, 14, a2ui.LayoutAlignStart),
		column("templates-tab", []string{"templates-title", "templates-card"}, 14, a2ui.LayoutAlignStart),
		column("layout-tab", []string{"layout-page-title", "layout-card", "actions-card"}, 14, a2ui.LayoutAlignStart),
		column("media-tab", []string{"media-title", "media-card", "image-variants-card", "progress-card", "spinner-card"}, 14, a2ui.LayoutAlignStart),
		column("feedback-tab", []string{"feedback-title", "feedback-card", "feedback-summary-card"}, 14, a2ui.LayoutAlignStart),
		column("actions-tab", []string{"client-actions-title", "client-actions-card"}, 14, a2ui.LayoutAlignStart),
		column("theme-tab", []string{"theme-title", "theme-card"}, 14, a2ui.LayoutAlignStart),
		column("dialogs-tab", []string{"dialogs-title", "actions-row", "dialogs-card"}, 14, a2ui.LayoutAlignStart),
	)
	for i := 1; i <= 11; i++ {
		components[len(components)-i].Padding = float64Ptr(16)
	}

	components = append(components,
		text("showcase-title", "A2UI Showcase", a2ui.TextVariantH2),
		card("overview-card", "overview-card-content"),
		column("overview-card-content", []string{"overview-header", "overview-body", "overview-status"}, 10, a2ui.LayoutAlignStart),
		row("overview-header", []string{"overview-icon", "overview-heading"}, 8, a2ui.LayoutAlignCenter, ""),
		icon("overview-icon", a2ui.IconInfo),
		text("overview-heading", "Renderer Coverage", a2ui.TextVariantH4),
		text("overview-body", "This surface intentionally exercises every component the current macOS renderer supports.", ""),
		boundText("overview-status", "/showcase/status", a2ui.TextVariantCaption),
		divider("overview-divider"),
		row("support-grid", []string{"support-column-left", "support-column-right"}, 12, a2ui.LayoutAlignStart, a2ui.LayoutJustifyStart),
		column("support-column-left", []string{"support-card-core", "support-card-inputs"}, 10, a2ui.LayoutAlignStretch),
		column("support-column-right", []string{"support-card-layout", "support-card-modal"}, 10, a2ui.LayoutAlignStretch),
		card("support-card-core", "support-card-core-content"),
		column("support-card-core-content", []string{"support-core-title", "support-core-body"}, 6, a2ui.LayoutAlignStart),
		text("support-core-title", "Core Components", a2ui.TextVariantH5),
		text("support-core-body", "Text, Icon, Divider, Row, Column, Card, List, and Tabs all render natively.", a2ui.TextVariantCaption),
		card("support-card-inputs", "support-card-inputs-content"),
		column("support-card-inputs-content", []string{"support-inputs-title", "support-inputs-body"}, 6, a2ui.LayoutAlignStart),
		text("support-inputs-title", "Inputs", a2ui.TextVariantH5),
		text("support-inputs-body", "TextField, CheckBox, Slider, ChoicePicker, and DateTimeInput round-trip through the server.", a2ui.TextVariantCaption),
		card("support-card-layout", "support-card-layout-content"),
		column("support-card-layout-content", []string{"support-layout-title", "support-layout-body"}, 6, a2ui.LayoutAlignStart),
		text("support-layout-title", "Layout and Feedback", a2ui.TextVariantH5),
		text("support-layout-body", "Spacing, padding, alignment, justify, progress, and spinner coverage live in focused tabs.", a2ui.TextVariantCaption),
		card("support-card-modal", "support-card-modal-content"),
		column("support-card-modal-content", []string{"support-modal-title", "support-modal-body"}, 6, a2ui.LayoutAlignStart),
		text("support-modal-title", "Modal and Extensions", a2ui.TextVariantH5),
		text("support-modal-body", "Modal chrome is renderer policy; local extensions add Action, Progress, Padding, Spacing, and Strikethrough.", a2ui.TextVariantCaption),
		card("catalog-card", "catalog-card-content"),
		column("catalog-card-content", []string{"catalog-title", "catalog-copy-1", "catalog-copy-2"}, 8, a2ui.LayoutAlignStart),
		text("catalog-title", "Showcase Map", a2ui.TextVariantH4),
		text("catalog-copy-1", "Overview summarizes protocol coverage and current renderer policy decisions.", ""),
		text("catalog-copy-2", "The remaining tabs isolate typography, inputs, layout, media, and modal/button behavior so each family can be tested cleanly.", a2ui.TextVariantCaption),
		text("typography-page-title", "Typography and Copy", a2ui.TextVariantH3),
		card("typography-card", "typography-card-content"),
		column("typography-card-content", []string{"typography-title", "text-h1", "text-h2", "text-h3", "text-h4", "text-h5", "text-body", "text-caption", "text-struck"}, 6, a2ui.LayoutAlignStart),
		text("typography-title", "Text Variants", a2ui.TextVariantH4),
		text("text-h1", "Heading One", a2ui.TextVariantH1),
		text("text-h2", "Heading Two", a2ui.TextVariantH2),
		text("text-h3", "Heading Three", a2ui.TextVariantH3),
		text("text-h4", "Heading Four", a2ui.TextVariantH4),
		text("text-h5", "Heading Five", a2ui.TextVariantH5),
		text("text-body", "Body copy uses the default readable weight.", a2ui.TextVariantBody),
		text("text-caption", "Caption copy is intentionally quieter.", a2ui.TextVariantCaption),
		struckText("text-struck", "Strikethrough is carried as a local extension.", true),
		card("copy-card", "copy-card-content"),
		column("copy-card-content", []string{"copy-title", "copy-body", "copy-caption"}, 8, a2ui.LayoutAlignStart),
		text("copy-title", "Presentation Notes", a2ui.TextVariantH4),
		text("copy-body", "Text labels should remain visible even after fields hold values, because A2UI labels are semantic labels, not transient placeholders.", ""),
		text("copy-caption", "The renderer now keeps that distinction explicit on macOS.", a2ui.TextVariantCaption),
		text("layout-page-title", "Layout and Actions", a2ui.TextVariantH3),
		card("layout-card", "layout-card-content"),
		column("layout-card-content", []string{"layout-title", "justify-row", "vertical-divider-row", "horizontal-list"}, 10, a2ui.LayoutAlignStretch),
		text("layout-title", "Layout Variants", a2ui.TextVariantH4),
		row("justify-row", []string{"justify-start", "justify-end", "justify-center"}, 12, a2ui.LayoutAlignCenter, a2ui.LayoutJustifySpaceBetween),
		text("justify-start", "Start", a2ui.TextVariantCaption),
		text("justify-end", "End", a2ui.TextVariantCaption),
		text("justify-center", "Space Between", a2ui.TextVariantCaption),
		row("vertical-divider-row", []string{"divider-left", "divider-vertical", "divider-right"}, 12, a2ui.LayoutAlignCenter, a2ui.LayoutJustifyStart),
		text("divider-left", "Vertical divider", a2ui.TextVariantCaption),
		dividerAxis("divider-vertical", a2ui.DividerAxisVertical),
		text("divider-right", "Padding and spacing applied at the component level", a2ui.TextVariantCaption),
		list("horizontal-list", []string{"horizontal-item-1", "horizontal-item-2", "horizontal-item-3"}, a2ui.ListDirectionHorizontal),
		card("horizontal-item-1", "horizontal-item-1-body"),
		text("horizontal-item-1-body", "Horizontal", a2ui.TextVariantCaption),
		card("horizontal-item-2", "horizontal-item-2-body"),
		text("horizontal-item-2-body", "List", a2ui.TextVariantCaption),
		card("horizontal-item-3", "horizontal-item-3-body"),
		text("horizontal-item-3-body", "Direction", a2ui.TextVariantCaption),
		card("actions-card", "actions-card-content"),
		column("actions-card-content", []string{"actions-title", "button-row", "action-caption"}, 8, a2ui.LayoutAlignStart),
		text("actions-title", "Button Variants", a2ui.TextVariantH4),
		row("button-row", []string{"layout-default-button", "layout-primary-button", "layout-borderless-button"}, 10, a2ui.LayoutAlignCenter, a2ui.LayoutJustifyStart),
		button("layout-default-button", "layout-default-button-label", action("showcase_secondary", nil), a2ui.ButtonVariantDefault),
		text("layout-default-button-label", "Default", ""),
		button("layout-primary-button", "layout-primary-button-label", action("showcase_apply", nil), a2ui.ButtonVariantPrimary),
		text("layout-primary-button-label", "Primary", ""),
		button("layout-borderless-button", "layout-borderless-button-label", action("showcase_secondary", nil), a2ui.ButtonVariantBorderless),
		text("layout-borderless-button-label", "Borderless", ""),
		text("action-caption", "Button variants share the same action transport but map to distinct native presentation.", a2ui.TextVariantCaption),
	)
	components[len(components)-12].Padding = float64Ptr(10)
	components[len(components)-10].Padding = float64Ptr(10)
	components[len(components)-8].Padding = float64Ptr(10)

	components = append(components,
		text("form-title", "Form Controls", a2ui.TextVariantH3),
		card("form-card", "form-card-content"),
		column("form-card-content", []string{
			"name-field", "count-field", "secret-field", "notes-field", "enabled-toggle", "intensity-slider", "priority-picker", "tags-picker", "deadline-input", "apply-button",
		}, 8, a2ui.LayoutAlignStart),
		textField("name-field", "Name", "/showcase/name", action("showcase_name_submit", nil), a2ui.TextFieldVariantShortText),
		textFieldRegex("count-field", "Count", "/showcase/count", action("showcase_count_submit", nil), a2ui.TextFieldVariantNumber, `^-?[0-9]+$`),
		textField("secret-field", "Secret", "/showcase/secret", action("showcase_secret_submit", nil), a2ui.TextFieldVariantObscured),
		textField("notes-field", "Notes", "/showcase/notes", action("showcase_notes_submit", nil), a2ui.TextFieldVariantLongText),
		checkBox("enabled-toggle", "Enable rich mode", "/showcase/enabled", action("showcase_enabled_change", nil)),
		slider("intensity-slider", "Intensity", "/showcase/intensity", 0, 100, action("showcase_intensity_change", nil)),
		choicePicker(
			"priority-picker",
			"Priority",
			[]a2ui.ChoiceOption{option("Low"), option("Medium"), option("High")},
			"/showcase/priority",
			a2ui.ChoicePickerVariantMutuallyExclusive,
			a2ui.ChoicePickerDisplayStyleChips,
			action("showcase_priority_change", nil),
		),
		filterableChoicePicker(
			"tags-picker",
			"Tags",
			[]a2ui.ChoiceOption{option("Renderer"), option("Variants"), option("Extensions")},
			"/showcase/tags",
			a2ui.ChoicePickerVariantMultipleSelection,
			action("showcase_tags_change", nil),
		),
		dateTimeInputBounded(
			"deadline-input",
			"Deadline",
			"/showcase/deadline",
			true,
			true,
			time.Now().Add(-24*time.Hour).UTC().Format(time.RFC3339),
			time.Now().Add(7*24*time.Hour).UTC().Format(time.RFC3339),
			action("showcase_deadline_change", nil),
		),
		button("apply-button", "apply-button-label", action("showcase_apply", nil), a2ui.ButtonVariantPrimary),
		text("apply-button-label", "Apply Snapshot", ""),
	)

	components = append(components,
		text("functions-title", "Dynamic Functions", a2ui.TextVariantH3),
		card("functions-card", "functions-card-content"),
		column("functions-card-content", []string{"functions-heading", "format-status", "format-number", "format-currency", "format-date", "format-plural"}, 8, a2ui.LayoutAlignStart),
		text("functions-heading", "Resolved at render time", a2ui.TextVariantH4),
		dynamicText("format-status", a2ui.FormatString(a2ui.StringLiteral("Status: ${/showcase/status}")), a2ui.TextVariantBody),
		dynamicText("format-number", a2ui.FormatNumber(a2ui.NumberLiteral(0), a2ui.BoolLiteral(true), a2ui.NumberBinding("/showcase/intensity")), a2ui.TextVariantCaption),
		dynamicText("format-currency", a2ui.FormatCurrency(a2ui.StringLiteral("$"), a2ui.NumberLiteral(2), a2ui.BoolLiteral(true), a2ui.NumberBinding("/showcase/budget")), a2ui.TextVariantCaption),
		dynamicText("format-date", a2ui.FormatDate(a2ui.StringLiteral("YYYY-MM-dd HH:mm"), a2ui.ValueBinding("/showcase/deadline")), a2ui.TextVariantCaption),
		dynamicText("format-plural", a2ui.Pluralize(a2ui.StringLiteral("a few items"), a2ui.StringLiteral("many items"), a2ui.StringLiteral("one item"), a2ui.StringLiteral("items"), a2ui.StringLiteral("two items"), a2ui.NumberBinding("/showcase/itemCount"), a2ui.StringLiteral("no items")), a2ui.TextVariantCaption),
	)

	components = append(components,
		text("validation-title", "Validation and Checks", a2ui.TextVariantH3),
		card("validation-card", "validation-card-content"),
		column("validation-card-content", []string{"validation-heading", "contact-email-field", "validation-caption"}, 8, a2ui.LayoutAlignStart),
		text("validation-heading", "Required and email checks", a2ui.TextVariantH4),
		checkedTextField("contact-email-field", "Contact Email", "/showcase/contactEmail", action("showcase_contact_submit", nil), a2ui.TextFieldVariantShortText,
			a2ui.CheckRule{Condition: a2ui.Required(a2ui.ValueBinding("/showcase/contactEmail")), Message: "Email is required"},
			a2ui.CheckRule{Condition: a2ui.Email(a2ui.StringBinding("/showcase/contactEmail")), Message: "Email must look valid"},
		),
		text("validation-caption", "The runtime aggregates A2UI checks instead of treating validation as a renderer-only regex feature.", a2ui.TextVariantCaption),
	)

	components = append(components,
		text("templates-title", "Templated Children", a2ui.TextVariantH3),
		card("templates-card", "templates-card-content"),
		column("templates-card-content", []string{"templates-heading", "template-list", "templates-caption"}, 8, a2ui.LayoutAlignStart),
		text("templates-heading", "Child templates expand from data-model objects", a2ui.TextVariantH4),
		templatedList("template-list", "template-item", "/showcase/templateItems"),
		text("templates-caption", "Each object in /showcase/templateItems overlays the template component before render.", a2ui.TextVariantCaption),
		dynamicText("template-item", a2ui.StringLiteral("Template item"), a2ui.TextVariantCaption),
	)

	components = append(components,
		text("media-title", "Media and Progress", a2ui.TextVariantH3),
		card("media-card", "media-card-content"),
		column("media-card-content", []string{"hero-image", "video-preview", "audio-preview", "media-note"}, 10, a2ui.LayoutAlignStart),
		image("hero-image", "/showcase/imageURL", a2ui.ImageVariantMediumFeature),
		video("video-preview", "/showcase/videoURL"),
		audioPlayer("audio-preview", "/showcase/audioURL", "Audio demo"),
		text("media-note", "Image fit and shell treatment are renderer policy choices tuned for compact desktop cards.", a2ui.TextVariantCaption),
		card("image-variants-card", "image-variants-content"),
		column("image-variants-content", []string{"image-variants-title", "image-variant-row-1", "image-variant-row-2"}, 8, a2ui.LayoutAlignStart),
		text("image-variants-title", "Image Variants", a2ui.TextVariantH5),
		row("image-variant-row-1", []string{"image-icon", "image-avatar", "image-small"}, 10, a2ui.LayoutAlignCenter, a2ui.LayoutJustifyStart),
		imageLiteralFit("image-icon", "https://picsum.photos/64", a2ui.ImageVariantIcon, a2ui.ImageFitContain),
		imageLiteralFit("image-avatar", "https://picsum.photos/96", a2ui.ImageVariantAvatar, a2ui.ImageFitCover),
		imageLiteralFit("image-small", "https://picsum.photos/120/90", a2ui.ImageVariantSmallFeature, a2ui.ImageFitFill),
		row("image-variant-row-2", []string{"image-medium", "image-large", "image-header"}, 10, a2ui.LayoutAlignCenter, a2ui.LayoutJustifyStart),
		imageLiteralFit("image-medium", "https://picsum.photos/180/120", a2ui.ImageVariantMediumFeature, a2ui.ImageFitScaleDown),
		imageLiteralFit("image-large", "https://picsum.photos/240/180", a2ui.ImageVariantLargeFeature, a2ui.ImageFitNone),
		imageLiteralFit("image-header", "https://picsum.photos/320/120", a2ui.ImageVariantHeader, a2ui.ImageFitCover),
		text("feedback-title", "Feedback and State", a2ui.TextVariantH3),
		card("feedback-card", "feedback-card-content"),
		column("feedback-card-content", []string{"progress-title", "progress-bar", "progress-caption", "spinner-title", "spinner-view"}, 8, a2ui.LayoutAlignStart),
		text("progress-title", "Progress", a2ui.TextVariantH5),
		a2ui.ProgressBar("progress-bar", a2ui.NumberBinding("/showcase/intensity"), 100),
		text("progress-caption", "Bound to the showcase intensity slider.", a2ui.TextVariantCaption),
		text("spinner-title", "Indeterminate Progress", a2ui.TextVariantH5),
		a2ui.Spinner("spinner-view"),
		card("feedback-summary-card", "feedback-summary-content"),
		column("feedback-summary-content", []string{"summary-title", "summary-name", "summary-count", "summary-priority", "summary-tags", "summary-notes"}, 8, a2ui.LayoutAlignStart),
		text("summary-title", "Server Snapshot", a2ui.TextVariantH5),
		boundText("summary-name", "/showcase/name", ""),
		boundText("summary-count", "/showcase/count", a2ui.TextVariantCaption),
		boundText("summary-priority", "/showcase/priority", a2ui.TextVariantCaption),
		boundText("summary-tags", "/showcase/tags/0", a2ui.TextVariantCaption),
		boundText("summary-notes", "/showcase/notes", a2ui.TextVariantCaption),
	)

	components = append(components,
		text("client-actions-title", "Client-side Actions", a2ui.TextVariantH3),
		card("client-actions-card", "client-actions-card-content"),
		column("client-actions-card-content", []string{"client-actions-heading", "open-docs-button", "client-actions-caption"}, 8, a2ui.LayoutAlignStart),
		text("client-actions-heading", "openUrl executes locally", a2ui.TextVariantH4),
		buttonFunction("open-docs-button", "open-docs-button-label", a2ui.OpenURL("https://a2ui.org"), a2ui.ButtonVariantPrimary),
		text("open-docs-button-label", "Open A2UI Docs", ""),
		text("client-actions-caption", "This button does not call the server; the runtime executes the client function directly.", a2ui.TextVariantCaption),
	)

	components = append(components,
		text("theme-title", "Theme and Accessibility", a2ui.TextVariantH3),
		card("theme-card", "theme-card-content"),
		column("theme-card-content", []string{"theme-heading", "theme-primary-button", "theme-caption", "accessible-note"}, 8, a2ui.LayoutAlignStart),
		text("theme-heading", "Theme colors and labels", a2ui.TextVariantH4),
		button("theme-primary-button", "theme-primary-button-label", action("showcase_secondary", nil), a2ui.ButtonVariantPrimary),
		text("theme-primary-button-label", "Theme-tinted Button", ""),
		text("theme-caption", "PrimaryColor from CreateSurface.Theme should tint prominent actions consistently.", a2ui.TextVariantCaption),
		accessibleText("accessible-note", "Accessible label and hint travel through the runtime.", "Accessibility note", "Read by assistive technologies"),
	)

	components = append(components,
		text("dialogs-title", "Dialogs and Buttons", a2ui.TextVariantH3),
		row("actions-row", []string{"secondary-button", "details-modal"}, 10, a2ui.LayoutAlignCenter, a2ui.LayoutJustifyStart),
		button("secondary-button", "secondary-button-label", action("showcase_secondary", nil), a2ui.ButtonVariantBorderless),
		text("secondary-button-label", "Ping Server", ""),
		modal("details-modal", "details-modal-content", "details-modal-trigger"),
		button("details-modal-trigger", "details-modal-trigger-label", action("showcase_modal_open", nil), a2ui.ButtonVariantPrimary),
		text("details-modal-trigger-label", "Open Feature Matrix", ""),
		column("details-modal-content", []string{"details-modal-title", "details-modal-version", "details-modal-body"}, 8, a2ui.LayoutAlignStart),
		text("details-modal-title", "A2UI Feature Matrix", a2ui.TextVariantH2),
		text("details-modal-version", "macOS renderer policy", a2ui.TextVariantCaption),
		text("details-modal-body", "Modal styling is renderer-defined because the core A2UI modal spec only references trigger and content IDs.", ""),
		card("dialogs-card", "dialogs-card-content"),
		column("dialogs-card-content", []string{"dialogs-card-title", "dialogs-card-copy-1", "dialogs-card-copy-2"}, 8, a2ui.LayoutAlignStart),
		text("dialogs-card-title", "Dialog Policy", a2ui.TextVariantH4),
		text("dialogs-card-copy-1", "macOS sheets need a presentation shell to feel native, because the protocol does not currently carry title, size, or footer actions.", ""),
		text("dialogs-card-copy-2", "This renderer injects that shell consistently so modal content reads like an intentional desktop surface rather than raw stacked text.", a2ui.TextVariantCaption),
	)

	return components
}

func (s *server) buildDataModel() map[string]any {
	if s.mode == "showcase" {
		return map[string]any{
			"showcase": map[string]any{
				"name":      s.showcase.Name,
				"count":     s.showcase.Count,
				"secret":    s.showcase.Secret,
				"notes":     s.showcase.Notes,
				"enabled":   s.showcase.Enabled,
				"priority":  s.showcase.Priority,
				"tags":      s.showcase.Tags,
				"deadline":  s.showcase.Deadline,
				"intensity": s.showcase.Intensity,
				"status":    s.showcase.Status,
				"budget":    12345.67,
				"itemCount": float64(3),
				"contactEmail": s.showcase.ContactEmail,
				"templateItems": []any{
					map[string]any{"text": "Templated renderer coverage", "variant": string(a2ui.TextVariantCaption)},
					map[string]any{"text": "Templated validation coverage", "variant": string(a2ui.TextVariantCaption), "padding": 6.0},
					map[string]any{"text": "Templated client action coverage", "variant": string(a2ui.TextVariantCaption)},
				},
				"imageURL":  s.showcase.ImageURL,
				"videoURL":  s.showcase.VideoURL,
				"audioURL":  s.showcase.AudioURL,
			},
		}
	}
	tasksMap := make(map[string]any, len(s.tasks))
	for _, t := range s.tasks {
		tasksMap[strconv.Itoa(t.ID)] = map[string]any{
			"id":   t.ID,
			"text": t.Text,
			"done": t.Done,
		}
	}
	return map[string]any{
		"tasks":         tasksMap,
		"inputText":     s.inputText,
		"status":        s.statusText(),
		"completionPct": s.completionPct(),
		"settings": map[string]any{
			"sortOrder":     s.settings.SortOrder,
			"fontSize":      s.settings.FontSize,
			"showCompleted": s.settings.ShowCompleted,
			"deadline":      s.settings.Deadline,
		},
	}
}

func (s *server) dataModelMessage() a2ui.ServerMessage {
	return s.dataModelPathMessage("", s.buildDataModel())
}

func (s *server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan a2ui.ServerMessage, 16)
	s.addClient(ch)
	defer s.removeClient(ch)

	if err := writeSSE(w, s.createSurfaceMessage()); err != nil {
		return
	}
	if err := writeSSE(w, s.updateComponentsMessage(
		column("root", []string{"loading-text", "loading-bar"}, 12, a2ui.LayoutAlignCenter),
		text("loading-text", loadingLabel(s.mode), a2ui.TextVariantH3),
		a2ui.Spinner("loading-bar"),
	)); err != nil {
		return
	}
	flusher.Flush()
	time.Sleep(600 * time.Millisecond)

	s.mu.Lock()
	components := s.buildComponents()
	dmMsg := s.dataModelMessage()
	s.mu.Unlock()

	if err := writeSSE(w, s.updateComponentsMessage(components...)); err != nil {
		return
	}
	if err := writeSSE(w, dmMsg); err != nil {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSE(w, msg); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			s.mu.Lock()
			msg := s.dataModelPathMessage(statusPath(s.mode), s.statusText())
			s.mu.Unlock()
			if err := writeSSE(w, msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, msg a2ui.ServerMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

func (s *server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var action a2ui.ClientAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	log.Printf("action: %s from %s context=%v", action.Name, action.SourceComponentID, action.Context)

	s.mu.Lock()
	switch action.Name {
	case "add_task":
		text := s.inputText
		if v, ok := action.Context["value"].(string); ok && v != "" {
			text = v
		} else if v, ok := action.Context["/inputText"].(string); ok && v != "" {
			text = v
		}
		if text == "" {
			s.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		s.tasks = append(s.tasks, task{ID: s.nextID, Text: text})
		s.nextID++
		s.inputText = ""
		components := s.buildComponents()
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(s.updateComponentsMessage(components...))
		s.broadcast(dmMsg)

	case "toggle_task":
		taskID, _ := taskIDFromContext(action.Context)
		for i := range s.tasks {
			if s.tasks[i].ID == taskID {
				s.tasks[i].Done = !s.tasks[i].Done
				break
			}
		}
		components := s.buildComponents()
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(s.updateComponentsMessage(components...))
		s.broadcast(dmMsg)

	case "input_change":
		if v, ok := action.Context["value"].(string); ok {
			s.inputText = v
		}
		s.mu.Unlock()

	case "sort_change":
		if v, ok := action.Context["value"].(string); ok {
			s.settings.SortOrder = v
		}
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "font_change":
		if v, ok := action.Context["value"].(float64); ok {
			s.settings.FontSize = int(v)
		}
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "show_completed_change":
		if v, ok := action.Context["value"].(bool); ok {
			s.settings.ShowCompleted = v
		}
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "deadline_change":
		if v, ok := action.Context["value"].(string); ok {
			s.settings.Deadline = v
		}
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_name_submit":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.Name = v
		}
		s.showcase.Status = "Updated the showcase name."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_secret_submit":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.Secret = v
		}
		s.showcase.Status = "Updated the showcase secret."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_count_submit":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.Count = v
		}
		s.showcase.Status = "Updated the showcase count."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_notes_submit":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.Notes = v
		}
		s.showcase.Status = "Updated the showcase notes."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_contact_submit":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.ContactEmail = v
			s.showcase.Status = fmt.Sprintf("Captured contact email %s.", v)
		}
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_enabled_change":
		if v, ok := action.Context["value"].(bool); ok {
			s.showcase.Enabled = v
		}
		s.showcase.Status = "Toggled the showcase feature flag."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_intensity_change":
		if v, ok := action.Context["value"].(float64); ok {
			s.showcase.Intensity = int(v)
		}
		s.showcase.Status = fmt.Sprintf("Intensity is now %d.", s.showcase.Intensity)
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_priority_change":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.Priority = v
		}
		s.showcase.Status = fmt.Sprintf("Priority set to %s.", s.showcase.Priority)
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_tags_change":
		switch v := action.Context["value"].(type) {
		case []any:
			tags := make([]string, 0, len(v))
			for _, item := range v {
				tags = append(tags, fmt.Sprintf("%v", item))
			}
			s.showcase.Tags = tags
		case []string:
			s.showcase.Tags = append([]string(nil), v...)
		}
		s.showcase.Status = fmt.Sprintf("Selected %d tags.", len(s.showcase.Tags))
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_deadline_change":
		if v, ok := action.Context["value"].(string); ok {
			s.showcase.Deadline = v
		}
		s.showcase.Status = "Scheduled the showcase deadline."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_apply":
		if v, ok := action.Context["/showcase/name"].(string); ok {
			s.showcase.Name = v
		}
		if v, ok := action.Context["/showcase/count"].(string); ok {
			s.showcase.Count = v
		}
		if v, ok := action.Context["/showcase/secret"].(string); ok {
			s.showcase.Secret = v
		}
		if v, ok := action.Context["/showcase/notes"].(string); ok {
			s.showcase.Notes = v
		}
		s.showcase.Status = "Applied the current local form snapshot."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_secondary":
		s.showcase.Status = fmt.Sprintf("Server ping at %s.", time.Now().Format("15:04:05"))
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	case "showcase_modal_open", "about":
		s.showcase.Status = "Opened a modal surface."
		dmMsg := s.dataModelMessage()
		s.mu.Unlock()
		s.broadcast(dmMsg)

	default:
		s.mu.Unlock()
		log.Printf("unknown action: %s", action.Name)
	}

	w.WriteHeader(http.StatusOK)
}

func taskIDFromContext(ctx map[string]any) (int, bool) {
	v, ok := ctx["taskID"]
	if !ok {
		return 0, false
	}
	switch id := v.(type) {
	case float64:
		return int(id), true
	case string:
		n, err := strconv.Atoi(id)
		return n, err == nil
	default:
		return 0, false
	}
}

func (s *server) createSurfaceMessage() a2ui.ServerMessage {
	return a2ui.ServerMessage{
		Version: a2ui.Version,
		CreateSurface: &a2ui.CreateSurface{
			SurfaceID: s.currentSurfaceID(),
			CatalogID: s.currentCatalogID(),
			Theme: &a2ui.Theme{
				PrimaryColor:     "#007AFF",
				AgentDisplayName: "A2UI Demo Agent",
			},
		},
	}
}

func (s *server) updateComponentsMessage(components ...a2ui.Component) a2ui.ServerMessage {
	return a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateComponents: &a2ui.UpdateComponents{
			SurfaceID:  s.currentSurfaceID(),
			Components: components,
		},
	}
}

func (s *server) dataModelPathMessage(path string, value any) a2ui.ServerMessage {
	return a2ui.ServerMessage{
		Version: a2ui.Version,
		UpdateDataModel: &a2ui.UpdateDataModel{
			SurfaceID: s.currentSurfaceID(),
			Path:      path,
			Value:     value,
		},
	}
}

func loadingLabel(mode string) string {
	if mode == "showcase" {
		return "Loading A2UI Showcase..."
	}
	return "Loading Task Tracker..."
}

func statusPath(mode string) string {
	if mode == "showcase" {
		return "/showcase/status"
	}
	return "/status"
}

func action(name string, ctx map[string]a2ui.DynamicValue) *a2ui.Action {
	return &a2ui.Action{
		Event: &a2ui.EventAction{
			Name:    name,
			Context: ctx,
		},
	}
}

func tabs(id string, defs ...a2ui.TabDef) a2ui.Component {
	return a2ui.FromSDK(a2uibuild.Tabs(id, defs))
}

func tab(title, child string) a2ui.TabDef {
	return a2ui.TabDef{
		Title: a2ui.StringLiteral(title),
		Child: child,
	}
}

func text(id, value string, variant a2ui.TextVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Text(id, a2ui.StringLiteral(value)))
	c.Text.Variant = variant
	return c
}

func dynamicText(id string, value a2ui.DynamicString, variant a2ui.TextVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Text(id, value))
	c.Text.Variant = variant
	return c
}

func boundText(id, path string, variant a2ui.TextVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Text(id, a2ui.StringBinding(path)))
	c.Text.Variant = variant
	return c
}

func struckText(id, value string, struck bool) a2ui.Component {
	c := text(id, value, "")
	c.Strikethrough = &struck
	return c
}

func row(id string, children []string, spacing float64, align a2ui.LayoutAlign, justify a2ui.LayoutJustify) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Row(id, a2ui.ChildList{IDs: children}))
	c.Row.Align = align
	c.Row.Justify = justify
	c.Spacing = &spacing
	return c
}

func column(id string, children []string, spacing float64, align a2ui.LayoutAlign) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Column(id, a2ui.ChildList{IDs: children}))
	c.Column.Align = align
	c.Spacing = &spacing
	return c
}

func list(id string, children []string, direction a2ui.ListDirection) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.List(id, a2ui.ChildList{IDs: children}))
	c.List.Direction = direction
	return c
}

func button(id, child string, action *a2ui.Action, variant a2ui.ButtonVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Button(id, derefAction(action), child))
	c.Button.Variant = variant
	return c
}

func card(id, child string) a2ui.Component {
	return a2ui.FromSDK(a2uibuild.Card(id, child))
}

func divider(id string) a2ui.Component {
	return a2ui.FromSDK(a2uibuild.Divider(id))
}

func dividerAxis(id string, axis a2ui.DividerAxis) a2ui.Component {
	c := divider(id)
	c.Divider.Axis = axis
	return c
}

func textField(id, label, path string, action *a2ui.Action, variant a2ui.TextFieldVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.TextField(id, a2ui.StringLiteral(label)))
	value := a2ui.StringBinding(path)
	c.TextField.Value = &value
	c.TextField.Variant = variant
	c.Action = action
	return c
}

func checkBox(id, label, path string, action *a2ui.Action) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.CheckBox(id, a2ui.StringLiteral(label), a2ui.BoolBinding(path)))
	c.Action = action
	return c
}

func slider(id, label, path string, min, max float64, action *a2ui.Action) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Slider(id, max, a2ui.NumberBinding(path)))
	c.Slider.Label = dynamicStringPtr(a2ui.StringLiteral(label))
	c.Slider.Min = &min
	c.Action = action
	return c
}

func choicePicker(id, label string, options []a2ui.ChoiceOption, path string, variant a2ui.ChoicePickerVariant, display a2ui.ChoicePickerDisplayStyle, action *a2ui.Action) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.ChoicePicker(id, options, a2ui.StringListBinding(path)))
	c.ChoicePicker.Label = dynamicStringPtr(a2ui.StringLiteral(label))
	c.ChoicePicker.Variant = variant
	c.ChoicePicker.DisplayStyle = display
	c.Action = action
	return c
}

func filterableChoicePicker(id, label string, options []a2ui.ChoiceOption, path string, variant a2ui.ChoicePickerVariant, action *a2ui.Action) a2ui.Component {
	c := choicePicker(id, label, options, path, variant, a2ui.ChoicePickerDisplayStyleCheckbox, action)
	filterable := true
	c.ChoicePicker.Filterable = &filterable
	return c
}

func option(label string) a2ui.ChoiceOption {
	return a2ui.ChoiceOption{
		Label: a2ui.StringLiteral(label),
		Value: label,
	}
}

func dateTimeInput(id, label, path string, enableDate, enableTime bool, action *a2ui.Action) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.DateTimeInput(id, a2ui.StringBinding(path)))
	c.DateTimeInput.Label = dynamicStringPtr(a2ui.StringLiteral(label))
	c.DateTimeInput.EnableDate = &enableDate
	c.DateTimeInput.EnableTime = &enableTime
	c.Action = action
	return c
}

func dateTimeInputBounded(id, label, path string, enableDate, enableTime bool, min, max string, action *a2ui.Action) a2ui.Component {
	c := dateTimeInput(id, label, path, enableDate, enableTime, action)
	if min != "" {
		v := a2ui.StringLiteral(min)
		c.DateTimeInput.Min = &v
	}
	if max != "" {
		v := a2ui.StringLiteral(max)
		c.DateTimeInput.Max = &v
	}
	return c
}

func modal(id, content, trigger string) a2ui.Component {
	return a2ui.FromSDK(a2uibuild.Modal(id, content, trigger))
}

func icon(id string, name a2ui.IconName) a2ui.Component {
	return a2ui.FromSDK(a2uibuild.Icon(id, a2ui.IconNameOrPath{Name: &name}))
}

func image(id, path string, variant a2ui.ImageVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Image(id, a2ui.StringBinding(path)))
	c.Image.Variant = variant
	return c
}

func imageLiteral(id, url string, variant a2ui.ImageVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Image(id, a2ui.StringLiteral(url)))
	c.Image.Variant = variant
	return c
}

func imageLiteralFit(id, url string, variant a2ui.ImageVariant, fit a2ui.ImageFit) a2ui.Component {
	c := imageLiteral(id, url, variant)
	c.Image.Fit = fit
	return c
}

func textFieldRegex(id, label, path string, action *a2ui.Action, variant a2ui.TextFieldVariant, validation string) a2ui.Component {
	c := textField(id, label, path, action, variant)
	c.TextField.ValidationRegexp = validation
	return c
}

func checkedTextField(id, label, path string, action *a2ui.Action, variant a2ui.TextFieldVariant, checks ...a2ui.CheckRule) a2ui.Component {
	c := textField(id, label, path, action, variant)
	c.Checks = append(c.Checks, checks...)
	return c
}

func templatedList(id, componentID, path string) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.List(id, a2ui.ChildList{
		Template: &a2ui.ChildTemplate{
			ComponentID: componentID,
			Path:        path,
		},
	}))
	return c
}

func buttonFunction(id, child string, action a2ui.Action, variant a2ui.ButtonVariant) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.Button(id, action, child))
	c.Button.Variant = variant
	return c
}

func accessibleText(id, value, label, description string) a2ui.Component {
	c := text(id, value, a2ui.TextVariantCaption)
	c.Accessibility = &a2ui.AccessibilityAttributes{
		Label:       dynamicStringPtr(a2ui.StringLiteral(label)),
		Description: dynamicStringPtr(a2ui.StringLiteral(description)),
	}
	return c
}

func video(id, path string) a2ui.Component {
	return a2ui.FromSDK(a2uibuild.Video(id, a2ui.StringBinding(path)))
}

func audioPlayer(id, path, description string) a2ui.Component {
	c := a2ui.FromSDK(a2uibuild.AudioPlayer(id, a2ui.StringBinding(path)))
	c.AudioPlayer.Description = dynamicStringPtr(a2ui.StringLiteral(description))
	return c
}

func derefAction(action *a2ui.Action) a2ui.Action {
	if action == nil {
		return a2ui.Action{}
	}
	return *action
}

func dynamicStringPtr(v a2ui.DynamicString) *a2ui.DynamicString {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func main() {
	modeFlag := flag.String("surface", "tasks", "demo surface to stream: tasks or showcase")
	flag.Parse()

	srv := newServer(*modeFlag)

	http.HandleFunc("/sse", srv.handleSSE)
	http.HandleFunc("/action", srv.handleAction)

	log.Printf("a2ui-server listening on http://localhost:8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatal(err)
	}
}

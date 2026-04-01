// Command a2ui-server is a demo HTTP server that streams A2UI component
// trees as Server-Sent Events. It implements a task tracker UI exercising
// all 19 A2UI v0.9 component types.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/tmc/swiftui/a2ui"
)

// task represents a single to-do item.
type task struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// settings holds user-configurable options.
type settings struct {
	SortOrder     string `json:"sortOrder"`
	FontSize      int    `json:"fontSize"`
	ShowCompleted bool   `json:"showCompleted"`
	Deadline      string `json:"deadline"`
}

// server holds the shared state for the task tracker.
type server struct {
	mu        sync.Mutex
	tasks     []task
	nextID    int
	inputText string
	settings  settings
	clients   map[chan a2ui.Envelope]struct{}
}

func newServer() *server {
	return &server{
		tasks: []task{
			{ID: 1, Text: "Read the A2UI docs", Done: false},
			{ID: 2, Text: "Build a demo app", Done: false},
			{ID: 3, Text: "Ship it", Done: true},
		},
		nextID: 4,
		settings: settings{
			SortOrder:     "By Date",
			FontSize:      14,
			ShowCompleted: true,
			Deadline:      "",
		},
		clients: make(map[chan a2ui.Envelope]struct{}),
	}
}

// addClient registers a new SSE client channel.
func (s *server) addClient(ch chan a2ui.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[ch] = struct{}{}
}

// removeClient unregisters an SSE client channel.
func (s *server) removeClient(ch chan a2ui.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, ch)
	close(ch)
}

// broadcast sends an envelope to all connected clients.
func (s *server) broadcast(env a2ui.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- env:
		default:
		}
	}
}

// completionPct returns percentage of tasks completed.
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
	done := 0
	for _, t := range s.tasks {
		if t.Done {
			done++
		}
	}
	return fmt.Sprintf("%d tasks, %d completed", len(s.tasks), done)
}

// buildComponents returns the full component tree for the current state.
func (s *server) buildComponents() []a2ui.Component {
	var components []a2ui.Component

	// Root: Tabs
	components = append(components, a2ui.Component{
		ID:   "root",
		Type: a2ui.ComponentTabs,
		Properties: map[string]any{
			"tabs": []any{
				map[string]any{"title": "Tasks", "child": "tasks-tab"},
				map[string]any{"title": "Settings", "child": "settings-tab"},
			},
		},
	})

	// === Tasks tab ===
	components = append(components, a2ui.Component{
		ID:       "tasks-tab",
		Type:     a2ui.ComponentColumn,
		Children: []string{"title", "input-row", "divider-1", "task-list", "divider-2", "progress-card"},
		Properties: map[string]any{
			"spacing": 12,
			"padding": 16,
		},
	})

	components = append(components, a2ui.Component{
		ID:   "title",
		Type: a2ui.ComponentText,
		Properties: map[string]any{
			"text":    "Task Tracker",
			"variant": "h2",
		},
	})

	// Input row: TextField + Button
	components = append(components, a2ui.Component{
		ID:       "input-row",
		Type:     a2ui.ComponentRow,
		Children: []string{"input-field", "add-btn"},
		Properties: map[string]any{
			"justify": "start",
			"align":   "center",
			"spacing": 8,
		},
	})

	components = append(components, a2ui.Component{
		ID:   "input-field",
		Type: a2ui.ComponentTextField,
		Properties: map[string]any{
			"label":  "New task...",
			"value":  map[string]any{"path": "/inputText"},
			"action": map[string]any{"event": map[string]any{"name": "add_task"}},
		},
	})

	components = append(components,
		a2ui.Component{
			ID:       "add-btn",
			Type:     a2ui.ComponentButton,
			Children: []string{"add-btn-label"},
			Properties: map[string]any{
				"action":  map[string]any{"event": map[string]any{"name": "add_task"}},
				"variant": "primary",
			},
		},
		a2ui.Component{
			ID:   "add-btn-label",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text": "Add",
			},
		},
	)

	components = append(components, a2ui.Component{
		ID:   "divider-1",
		Type: a2ui.ComponentDivider,
	})

	// Build task list with dynamic children.
	taskChildren := make([]string, 0, len(s.tasks))
	for _, t := range s.tasks {
		rowID := fmt.Sprintf("task-%d", t.ID)
		checkID := fmt.Sprintf("task-%d-check", t.ID)
		textID := fmt.Sprintf("task-%d-text", t.ID)
		taskChildren = append(taskChildren, rowID)

		components = append(components,
			a2ui.Component{
				ID:       rowID,
				Type:     a2ui.ComponentRow,
				Children: []string{checkID, textID},
				Properties: map[string]any{
					"spacing": 8,
				},
			},
			a2ui.Component{
				ID:   checkID,
				Type: a2ui.ComponentCheckBox,
				Properties: map[string]any{
					"label":  "",
					"value":  map[string]any{"path": fmt.Sprintf("/tasks/%d/done", t.ID)},
					"action": map[string]any{"event": map[string]any{"name": "toggle_task", "context": map[string]any{"taskID": t.ID}}},
				},
			},
			a2ui.Component{
				ID:   textID,
				Type: a2ui.ComponentText,
				Properties: map[string]any{
					"text":          t.Text,
					"strikethrough": t.Done,
				},
			},
		)
	}

	components = append(components, a2ui.Component{
		ID:       "task-list",
		Type:     a2ui.ComponentList,
		Children: taskChildren,
		Properties: map[string]any{
			"direction": "vertical",
		},
	})

	components = append(components, a2ui.Component{
		ID:   "divider-2",
		Type: a2ui.ComponentDivider,
	})

	// Progress card
	components = append(components,
		a2ui.Component{
			ID:       "progress-card",
			Type:     a2ui.ComponentCard,
			Children: []string{"progress-content"},
		},
		a2ui.Component{
			ID:       "progress-content",
			Type:     a2ui.ComponentColumn,
			Children: []string{"progress-bar", "status-text"},
			Properties: map[string]any{
				"spacing": 8,
			},
		},
		a2ui.Component{
			ID:   "progress-bar",
			Type: a2ui.ComponentProgress,
			Properties: map[string]any{
				"value": map[string]any{"path": "/completionPct"},
				"max":   100,
			},
		},
		a2ui.Component{
			ID:   "status-text",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text":    map[string]any{"path": "/status"},
				"variant": "caption",
			},
		},
	)

	// === Settings tab ===
	components = append(components, a2ui.Component{
		ID:       "settings-tab",
		Type:     a2ui.ComponentColumn,
		Children: []string{"settings-title", "sort-picker", "font-slider", "show-completed", "deadline-section", "about-modal"},
		Properties: map[string]any{
			"spacing": 12,
			"padding": 16,
		},
	})

	components = append(components, a2ui.Component{
		ID:   "settings-title",
		Type: a2ui.ComponentText,
		Properties: map[string]any{
			"text":    "Settings",
			"variant": "h3",
		},
	})

	// ChoicePicker for sort order
	components = append(components, a2ui.Component{
		ID:   "sort-picker",
		Type: a2ui.ComponentChoicePicker,
		Properties: map[string]any{
			"label":        "Sort Order",
			"options":      []any{"By Date", "By Name", "By Status"},
			"value":        map[string]any{"path": "/settings/sortOrder"},
			"variant":      "mutuallyExclusive",
			"displayStyle": "chips",
		},
	})

	// Slider for font size
	components = append(components, a2ui.Component{
		ID:   "font-slider",
		Type: a2ui.ComponentSlider,
		Properties: map[string]any{
			"label": "Font Size",
			"value": map[string]any{"path": "/settings/fontSize"},
			"min":   10,
			"max":   24,
		},
	})

	// CheckBox for show completed
	components = append(components, a2ui.Component{
		ID:   "show-completed",
		Type: a2ui.ComponentCheckBox,
		Properties: map[string]any{
			"label": "Show Completed Tasks",
			"value": map[string]any{"path": "/settings/showCompleted"},
		},
	})

	// Deadline section
	components = append(components,
		a2ui.Component{
			ID:       "deadline-section",
			Type:     a2ui.ComponentColumn,
			Children: []string{"deadline-label", "deadline-input"},
			Properties: map[string]any{
				"spacing": 4,
			},
		},
		a2ui.Component{
			ID:   "deadline-label",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text":    "Filter by Deadline",
				"variant": "h5",
			},
		},
		a2ui.Component{
			ID:   "deadline-input",
			Type: a2ui.ComponentDateTimeInput,
			Properties: map[string]any{
				"label":      "Before",
				"value":      map[string]any{"path": "/settings/deadline"},
				"enableDate": true,
				"enableTime": false,
			},
		},
	)

	// Modal: About
	components = append(components,
		a2ui.Component{
			ID:   "about-modal",
			Type: a2ui.ComponentModal,
			Properties: map[string]any{
				"trigger": "about-trigger",
				"content": "about-content",
			},
		},
		a2ui.Component{
			ID:       "about-trigger",
			Type:     a2ui.ComponentButton,
			Children: []string{"about-btn-label"},
			Properties: map[string]any{
				"action":  map[string]any{"event": map[string]any{"name": "about"}},
				"variant": "borderless",
			},
		},
		a2ui.Component{
			ID:       "about-btn-label",
			Type:     a2ui.ComponentRow,
			Children: []string{"about-icon", "about-text"},
			Properties: map[string]any{
				"spacing": 4,
			},
		},
		a2ui.Component{
			ID:   "about-icon",
			Type: a2ui.ComponentIcon,
			Properties: map[string]any{
				"name": "info.circle",
			},
		},
		a2ui.Component{
			ID:   "about-text",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text": "About",
			},
		},
		a2ui.Component{
			ID:       "about-content",
			Type:     a2ui.ComponentColumn,
			Children: []string{"about-title", "about-version", "about-desc"},
			Properties: map[string]any{
				"spacing": 8,
			},
		},
		a2ui.Component{
			ID:   "about-title",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text":    "A2UI Task Tracker",
				"variant": "h2",
			},
		},
		a2ui.Component{
			ID:   "about-version",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text":    "Version 0.9",
				"variant": "caption",
			},
		},
		a2ui.Component{
			ID:   "about-desc",
			Type: a2ui.ComponentText,
			Properties: map[string]any{
				"text": "A demo of the A2UI v0.9 protocol rendering all component types natively on macOS.",
			},
		},
	)

	return components
}

// buildDataModel returns the full data model as a single value for the root path.
func (s *server) buildDataModel() map[string]any {
	// Use task ID as key so checkbox bindings like /tasks/1/done work correctly.
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

// dataModelEnvelope returns an envelope that sets the full data model.
func (s *server) dataModelEnvelope() a2ui.Envelope {
	return mustEnvelope(a2ui.UpdateDataModel{
		SurfaceID: "task-tracker",
		Value:     s.buildDataModel(),
	})
}

func mustEnvelope(msg any) a2ui.Envelope {
	env, err := a2ui.NewEnvelope(msg)
	if err != nil {
		panic(err)
	}
	return env
}

// handleSSE streams Server-Sent Events to the client.
func (s *server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan a2ui.Envelope, 16)
	s.addClient(ch)
	defer s.removeClient(ch)

	// Stream the UI progressively: surface → loading → components → data.
	// This demonstrates SSE streaming with incremental UI construction.

	// 1. Create the surface.
	if err := writeSSE(w, mustEnvelope(a2ui.CreateSurface{
		SurfaceID: "task-tracker",
		Title:     "Task Tracker",
		Theme: &a2ui.Theme{
			PrimaryColor:     "#007AFF",
			AgentDisplayName: "A2UI Demo Agent",
		},
	})); err != nil {
		return
	}

	// 2. Show a loading indicator while "building" the UI.
	if err := writeSSE(w, mustEnvelope(a2ui.UpdateComponents{
		SurfaceID: "task-tracker",
		Components: []a2ui.Component{
			{ID: "root", Type: a2ui.ComponentColumn, Children: []string{"loading-text", "loading-bar"}, Properties: map[string]any{"spacing": 12, "align": "center"}},
			{ID: "loading-text", Type: a2ui.ComponentText, Properties: map[string]any{"text": "Loading Task Tracker...", "variant": "h3"}},
			{ID: "loading-bar", Type: a2ui.ComponentProgress},
		},
	})); err != nil {
		return
	}
	flusher.Flush()
	time.Sleep(600 * time.Millisecond)

	// 3. Stream the full component tree and data model.
	s.mu.Lock()
	components := s.buildComponents()
	dmEnv := s.dataModelEnvelope()
	s.mu.Unlock()

	if err := writeSSE(w, mustEnvelope(a2ui.UpdateComponents{
		SurfaceID:  "task-tracker",
		Components: components,
	})); err != nil {
		return
	}
	if err := writeSSE(w, dmEnv); err != nil {
		return
	}
	flusher.Flush()

	// Background ticker for periodic status updates.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case env, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSE(w, env); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			s.mu.Lock()
			env := mustEnvelope(a2ui.UpdateDataModel{
				SurfaceID: "task-tracker",
				Path:      "/status",
				Value:     s.statusText(),
			})
			s.mu.Unlock()
			if err := writeSSE(w, env); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// writeSSE writes a single SSE event to the writer.
func writeSSE(w http.ResponseWriter, env a2ui.Envelope) error {
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

// handleAction processes client actions posted as JSON.
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
		// Also accept value from TextField submit context.
		if v, ok := action.Context["value"].(string); ok && v != "" {
			text = v
		}
		if text == "" {
			s.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			return
		}
		s.tasks = append(s.tasks, task{
			ID:   s.nextID,
			Text: text,
			Done: false,
		})
		s.nextID++
		s.inputText = ""
		components := s.buildComponents()
		dmEnv := s.dataModelEnvelope()
		s.mu.Unlock()
		s.broadcast(mustEnvelope(a2ui.UpdateComponents{
			SurfaceID:  "task-tracker",
			Components: components,
		}))
		s.broadcast(dmEnv)

	case "toggle_task":
		taskID, _ := taskIDFromContext(action.Context)
		for i := range s.tasks {
			if s.tasks[i].ID == taskID {
				s.tasks[i].Done = !s.tasks[i].Done
				break
			}
		}
		components := s.buildComponents()
		dmEnv := s.dataModelEnvelope()
		s.mu.Unlock()
		s.broadcast(mustEnvelope(a2ui.UpdateComponents{
			SurfaceID:  "task-tracker",
			Components: components,
		}))
		s.broadcast(dmEnv)

	case "input_change":
		if v, ok := action.Context["text"].(string); ok {
			s.inputText = v
		}
		s.mu.Unlock()

	case "sort_change":
		if v, ok := action.Context["value"].(string); ok {
			s.settings.SortOrder = v
		}
		dmEnv := s.dataModelEnvelope()
		s.mu.Unlock()
		s.broadcast(dmEnv)

	case "font_change":
		if v, ok := action.Context["value"].(float64); ok {
			s.settings.FontSize = int(v)
		}
		dmEnv := s.dataModelEnvelope()
		s.mu.Unlock()
		s.broadcast(dmEnv)

	case "show_completed_change":
		if v, ok := action.Context["value"].(bool); ok {
			s.settings.ShowCompleted = v
		}
		dmEnv := s.dataModelEnvelope()
		s.mu.Unlock()
		s.broadcast(dmEnv)

	case "deadline_change":
		if v, ok := action.Context["value"].(string); ok {
			s.settings.Deadline = v
		}
		dmEnv := s.dataModelEnvelope()
		s.mu.Unlock()
		s.broadcast(dmEnv)

	default:
		s.mu.Unlock()
		log.Printf("unknown action: %s", action.Name)
	}

	w.WriteHeader(http.StatusOK)
}

// taskIDFromContext extracts the integer task ID from an action context.
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

func main() {
	srv := newServer()

	http.HandleFunc("/sse", srv.handleSSE)
	http.HandleFunc("/action", srv.handleAction)

	log.Printf("a2ui-server listening on http://localhost:8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		log.Fatal(err)
	}
}

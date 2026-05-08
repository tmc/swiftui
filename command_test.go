package swiftui

import (
	"encoding/json"
	"testing"
)

func TestCommandItemIsSeparator(t *testing.T) {
	tests := []struct {
		name string
		item CommandItem
		want bool
	}{
		{"empty title no children", CommandItem{}, true},
		{"has title", CommandItem{Title: "Undo"}, false},
		{"empty title with children", CommandItem{Children: []CommandItem{{Title: "Sub"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.IsSeparator(); got != tt.want {
				t.Errorf("IsSeparator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandItemIsSubmenu(t *testing.T) {
	tests := []struct {
		name string
		item CommandItem
		want bool
	}{
		{"no children", CommandItem{Title: "Undo"}, false},
		{"nil children", CommandItem{Title: "Edit", Children: nil}, false},
		{"empty children", CommandItem{Title: "Edit", Children: []CommandItem{}}, false},
		{"has children", CommandItem{Title: "Find", Children: []CommandItem{{Title: "Find..."}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.IsSubmenu(); got != tt.want {
				t.Errorf("IsSubmenu() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStandardEditMenu(t *testing.T) {
	menu := StandardEditMenu()
	if menu.Title != "Edit" {
		t.Errorf("Title = %q, want %q", menu.Title, "Edit")
	}
	if got := len(menu.Items); got != 7 {
		t.Fatalf("len(Items) = %d, want 7", got)
	}
	tests := []struct {
		index int
		title string
		isSep bool
	}{
		{0, "Undo", false},
		{1, "Redo", false},
		{2, "", true},
		{3, "Cut", false},
		{4, "Copy", false},
		{5, "Paste", false},
		{6, "Select All", false},
	}
	for _, tt := range tests {
		item := menu.Items[tt.index]
		if item.Title != tt.title {
			t.Errorf("Items[%d].Title = %q, want %q", tt.index, item.Title, tt.title)
		}
		if item.IsSeparator() != tt.isSep {
			t.Errorf("Items[%d].IsSeparator() = %v, want %v", tt.index, item.IsSeparator(), tt.isSep)
		}
	}
	// Verify shortcuts on Undo and Redo.
	undo := menu.Items[0]
	if undo.Shortcut.Key != "z" || undo.Shortcut.Modifiers != ModCommand {
		t.Errorf("Undo shortcut = %+v, want key=z mods=ModCommand", undo.Shortcut)
	}
	if undo.SystemAction != CommandSystemUndo {
		t.Errorf("Undo system action = %q, want %q", undo.SystemAction, CommandSystemUndo)
	}
	redo := menu.Items[1]
	if redo.Shortcut.Key != "z" || redo.Shortcut.Modifiers != ModCommand|ModShift {
		t.Errorf("Redo shortcut = %+v, want key=z mods=ModCommand|ModShift", redo.Shortcut)
	}
	if redo.SystemAction != CommandSystemRedo {
		t.Errorf("Redo system action = %q, want %q", redo.SystemAction, CommandSystemRedo)
	}
}

func TestStandardFileMenu(t *testing.T) {
	menu := StandardFileMenu(CommandItem{
		Title: "Open Alternate Document",
	})
	if menu.Title != "File" {
		t.Fatalf("Title = %q, want %q", menu.Title, "File")
	}
	if got := len(menu.Items); got != 3 {
		t.Fatalf("len(Items) = %d, want 3", got)
	}
	if menu.Items[1].Title != "" || !menu.Items[1].IsSeparator() {
		t.Fatalf("Items[1] = %+v, want separator", menu.Items[1])
	}
	closeItem := menu.Items[2]
	if closeItem.Title != "Close Window" {
		t.Fatalf("close item title = %q, want %q", closeItem.Title, "Close Window")
	}
	if closeItem.Shortcut.Key != "w" || closeItem.Shortcut.Modifiers != ModCommand {
		t.Fatalf("close item shortcut = %+v, want key=w mods=ModCommand", closeItem.Shortcut)
	}
	if closeItem.SystemAction != CommandSystemCloseWindow {
		t.Fatalf("close item system action = %q, want %q", closeItem.SystemAction, CommandSystemCloseWindow)
	}
}

func TestStandardFileMenuSeparatorRules(t *testing.T) {
	tests := []struct {
		name  string
		items []CommandItem
		want  int
	}{
		{
			name:  "empty menu only adds close window",
			items: nil,
			want:  1,
		},
		{
			name:  "existing trailing separator is reused",
			items: []CommandItem{{Title: "Export"}, Separator()},
			want:  3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menu := StandardFileMenu(tt.items...)
			if got := len(menu.Items); got != tt.want {
				t.Fatalf("len(Items) = %d, want %d", got, tt.want)
			}
			closeItem := menu.Items[len(menu.Items)-1]
			if closeItem.SystemAction != CommandSystemCloseWindow {
				t.Fatalf("close item system action = %q, want %q", closeItem.SystemAction, CommandSystemCloseWindow)
			}
			if tt.name == "empty menu only adds close window" && closeItem.IsSeparator() {
				t.Fatal("close item unexpectedly serialized as separator")
			}
			if tt.name == "existing trailing separator is reused" && !menu.Items[1].IsSeparator() {
				t.Fatalf("Items[1] = %+v, want separator", menu.Items[1])
			}
		})
	}
}

func TestOpenRecentDocumentsMenu(t *testing.T) {
	opened := ""
	menu := OpenRecentDocumentsMenu([]DocumentRecent{
		{DisplayName: "Quarterly Review", Path: "/tmp/review.md"},
		{Path: "/tmp/product-strategy.md"},
	}, func(path string) {
		opened = path
	})

	if menu.Title != "Open Recent" {
		t.Fatalf("Title = %q, want %q", menu.Title, "Open Recent")
	}
	if !menu.IsSubmenu() {
		t.Fatal("OpenRecentDocumentsMenu should return a submenu")
	}

	items := marshalCommandItems([]CommandItem{menu})
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if got, want := len(items[0].Children), 2; got != want {
		t.Fatalf("len(children) = %d, want %d", got, want)
	}
	if got, want := items[0].Children[0].Title, "Quarterly Review"; got != want {
		t.Fatalf("children[0].Title = %q, want %q", got, want)
	}
	if got, want := items[0].Children[1].Title, "product-strategy.md"; got != want {
		t.Fatalf("children[1].Title = %q, want %q", got, want)
	}
	if items[0].Children[0].ActionCallbackID == 0 || items[0].Children[1].ActionCallbackID == 0 {
		t.Fatal("recent document action callbacks should be registered")
	}

	fn := commandCallbacks.lookup(uintptr(items[0].Children[1].ActionCallbackID))
	if fn == nil {
		t.Fatal("recent document action callback not registered")
	}
	if got := fn(); got != 1 {
		t.Fatalf("recent document action callback returned %d, want 1", got)
	}
	if opened != "/tmp/product-strategy.md" {
		t.Fatalf("recent document open path = %q, want %q", opened, "/tmp/product-strategy.md")
	}
}

func TestOpenRecentDocumentsMenuEmpty(t *testing.T) {
	menu := OpenRecentDocumentsMenu(nil, nil)
	if menu.Title != "Open Recent" {
		t.Fatalf("Title = %q, want %q", menu.Title, "Open Recent")
	}
	if !menu.IsSubmenu() {
		t.Fatal("empty recent menu should still be a submenu shell")
	}
	if got, want := len(menu.Children), 1; got != want {
		t.Fatalf("len(children) = %d, want %d", got, want)
	}
	if got, want := menu.Children[0].Title, "No Recent Documents"; got != want {
		t.Fatalf("children[0].Title = %q, want %q", got, want)
	}
	if menu.Children[0].Enabled == nil {
		t.Fatal("placeholder item should carry a disabled predicate")
	}
}

func TestStandardWindowMenu(t *testing.T) {
	menu := StandardWindowMenu()
	if menu.Title != "Window" {
		t.Errorf("Title = %q, want %q", menu.Title, "Window")
	}
	if got := len(menu.Items); got != 4 {
		t.Fatalf("len(Items) = %d, want 4", got)
	}
	if !menu.Items[2].IsSeparator() {
		t.Error("Items[2] should be a separator")
	}
	if menu.Items[0].SystemAction != CommandSystemMinimizeWindow {
		t.Errorf("Items[0].SystemAction = %q, want %q", menu.Items[0].SystemAction, CommandSystemMinimizeWindow)
	}
	if menu.Items[1].SystemAction != CommandSystemZoomWindow {
		t.Errorf("Items[1].SystemAction = %q, want %q", menu.Items[1].SystemAction, CommandSystemZoomWindow)
	}
	if menu.Items[3].SystemAction != CommandSystemBringAllToFront {
		t.Errorf("Items[3].SystemAction = %q, want %q", menu.Items[3].SystemAction, CommandSystemBringAllToFront)
	}
}

func TestStandardHelpMenu(t *testing.T) {
	menu := StandardHelpMenu()
	if menu.Title != "Help" {
		t.Errorf("Title = %q, want %q", menu.Title, "Help")
	}
	if len(menu.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(menu.Items))
	}
}

func TestCommandsSerialization(t *testing.T) {
	actionCalled := false
	cmds := Commands(CommandMenu{
		Title: "Test",
		Items: []CommandItem{
			{
				Title:    "Do Thing",
				Shortcut: KeyboardShortcut{Key: "d", Modifiers: ModCommand},
				Action: func(CommandContext) {
					actionCalled = true
				},
			},
			Separator(),
			{
				Title: "Sub",
				Children: []CommandItem{
					{Title: "Child"},
				},
			},
		},
	})

	plan := scenePlan{commands: &cmds}
	run := sceneRunPlan{}
	for _, menu := range plan.commands.Menus {
		run.Commands = append(run.Commands, sceneRunPlanCommand{
			Title: menu.Title,
			Items: marshalCommandItems(menu.Items),
		})
	}

	if len(run.Commands) != 1 {
		t.Fatalf("len(Commands) = %d, want 1", len(run.Commands))
	}
	cmd := run.Commands[0]
	if cmd.Title != "Test" {
		t.Errorf("Title = %q, want %q", cmd.Title, "Test")
	}
	if len(cmd.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(cmd.Items))
	}

	// First item: regular action item.
	item0 := cmd.Items[0]
	if item0.Kind != "item" {
		t.Errorf("Items[0].Kind = %q, want %q", item0.Kind, "item")
	}
	if item0.Title != "Do Thing" {
		t.Errorf("Items[0].Title = %q, want %q", item0.Title, "Do Thing")
	}
	if item0.ShortcutKey != "d" {
		t.Errorf("Items[0].ShortcutKey = %q, want %q", item0.ShortcutKey, "d")
	}
	if item0.ShortcutModifiers != uint64(ModCommand) {
		t.Errorf("Items[0].ShortcutModifiers = %d, want %d", item0.ShortcutModifiers, ModCommand)
	}
	if item0.ActionCallbackID == 0 {
		t.Error("Items[0].ActionCallbackID should be non-zero")
	}
	if item0.SystemAction != "" {
		t.Errorf("Items[0].SystemAction = %q, want empty", item0.SystemAction)
	}

	// Second item: separator.
	item1 := cmd.Items[1]
	if item1.Kind != "separator" {
		t.Errorf("Items[1].Kind = %q, want %q", item1.Kind, "separator")
	}

	// Third item: submenu.
	item2 := cmd.Items[2]
	if item2.Kind != "item" {
		t.Errorf("Items[2].Kind = %q, want %q", item2.Kind, "item")
	}
	if len(item2.Children) != 1 {
		t.Fatalf("Items[2].Children length = %d, want 1", len(item2.Children))
	}
	if item2.Children[0].Title != "Child" {
		t.Errorf("Items[2].Children[0].Title = %q, want %q", item2.Children[0].Title, "Child")
	}

	// Verify the action callback was registered and is invocable.
	fn := commandCallbacks.lookup(uintptr(item0.ActionCallbackID))
	if fn == nil {
		t.Fatal("action callback not registered")
	}
	fn()
	if !actionCalled {
		t.Error("action callback did not fire")
	}

	// Verify JSON roundtrip.
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded sceneRunPlan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(decoded.Commands) != 1 {
		t.Fatalf("decoded commands length = %d, want 1", len(decoded.Commands))
	}
}

func TestEnabledPredicate(t *testing.T) {
	enabled := true
	items := marshalCommandItems([]CommandItem{
		{
			Title: "Toggle",
			Enabled: func() bool {
				return enabled
			},
		},
	})
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].EnabledCallbackID == 0 {
		t.Fatal("EnabledCallbackID should be non-zero")
	}

	fn := commandCallbacks.lookup(uintptr(items[0].EnabledCallbackID))
	if fn == nil {
		t.Fatal("enabled callback not registered")
	}
	if got := fn(); got != 1 {
		t.Errorf("enabled callback returned %d, want 1", got)
	}
	enabled = false
	if got := fn(); got != 0 {
		t.Errorf("enabled callback returned %d, want 0", got)
	}
}

func TestMarshalCommandItemsActionAndSystemAction(t *testing.T) {
	actionCalled := false
	items := marshalCommandItems([]CommandItem{
		{
			Title:        "Close Window",
			SystemAction: CommandSystemCloseWindow,
			Action: func(CommandContext) {
				actionCalled = true
			},
		},
	})
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ActionCallbackID == 0 {
		t.Fatal("ActionCallbackID should be non-zero when Action is present")
	}
	if got, want := items[0].SystemAction, string(CommandSystemCloseWindow); got != want {
		t.Fatalf("items[0].SystemAction = %q, want %q", got, want)
	}

	fn := commandCallbacks.lookup(uintptr(items[0].ActionCallbackID))
	if fn == nil {
		t.Fatal("action callback not registered")
	}
	if got := fn(); got != 1 {
		t.Fatalf("action callback returned %d, want 1", got)
	}
	if !actionCalled {
		t.Fatal("action callback did not fire")
	}
}

func TestSystemActionSerialization(t *testing.T) {
	items := marshalCommandItems([]CommandItem{
		{
			Title:        "Close Window",
			Shortcut:     KeyboardShortcut{Key: "w", Modifiers: ModCommand},
			SystemAction: CommandSystemCloseWindow,
		},
	})
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if got, want := items[0].SystemAction, string(CommandSystemCloseWindow); got != want {
		t.Fatalf("items[0].SystemAction = %q, want %q", got, want)
	}
	if items[0].ActionCallbackID != 0 {
		t.Fatalf("items[0].ActionCallbackID = %d, want 0 for system action", items[0].ActionCallbackID)
	}
}

package swiftui

import "path/filepath"

// CommandContext carries per-invocation context for a command action.
//
// Runtime surface.
//
// It is intentionally minimal and future-proof; additional fields (focused
// window, selection, undo manager) will be added in later tranches without
// breaking existing callers.
type CommandContext struct{}

// KeyModifiers is a bitmask of keyboard modifier keys.
//
// Runtime surface.
type KeyModifiers uint64

const (
	// ModCommand is the Command (⌘) key.
	ModCommand KeyModifiers = 1 << 20 // 1048576 — NSEvent.ModifierFlags.command.rawValue
	// ModShift is the Shift (⇧) key.
	ModShift KeyModifiers = 1 << 17 // 131072
	// ModOption is the Option (⌥) key.
	ModOption KeyModifiers = 1 << 19 // 524288
	// ModControl is the Control (⌃) key.
	ModControl KeyModifiers = 1 << 18 // 262144
)

// KeyboardShortcut describes a key equivalent with modifier keys.
//
// Runtime surface.
type KeyboardShortcut struct {
	Key       string
	Modifiers KeyModifiers
}

// CommandSystemAction identifies a standard AppKit-owned menu action.
//
// Runtime surface.
//
// These actions route through the current runner rather than a Go callback.
// They exist for standard macOS menu behavior such as Undo, Close Window, and
// Minimize. When both SystemAction and Action are set on a CommandItem, Action
// takes precedence.
type CommandSystemAction string

const (
	CommandSystemNone            CommandSystemAction = ""
	CommandSystemUndo            CommandSystemAction = "undo"
	CommandSystemRedo            CommandSystemAction = "redo"
	CommandSystemCut             CommandSystemAction = "cut"
	CommandSystemCopy            CommandSystemAction = "copy"
	CommandSystemPaste           CommandSystemAction = "paste"
	CommandSystemSelectAll       CommandSystemAction = "selectAll"
	CommandSystemCloseWindow     CommandSystemAction = "closeWindow"
	CommandSystemMinimizeWindow  CommandSystemAction = "minimizeWindow"
	CommandSystemZoomWindow      CommandSystemAction = "zoomWindow"
	CommandSystemBringAllToFront CommandSystemAction = "bringAllToFront"
)

// CommandItem describes a single menu item, separator, or submenu parent.
//
// Runtime surface.
//
// An empty Title produces a separator. Non-nil Children makes the item a
// submenu parent.
type CommandItem struct {
	Title        string
	Shortcut     KeyboardShortcut
	SystemAction CommandSystemAction
	Action       func(CommandContext)
	Enabled      func() bool
	Children     []CommandItem
}

// IsSeparator reports whether the item is a menu separator.
func (c CommandItem) IsSeparator() bool {
	return c.Title == "" && len(c.Children) == 0
}

// IsSubmenu reports whether the item is a submenu parent.
func (c CommandItem) IsSubmenu() bool {
	return len(c.Children) > 0
}

// Separator returns a separator CommandItem.
func Separator() CommandItem {
	return CommandItem{}
}

// CommandMenu describes a top-level menu bar header with its items.
//
// Runtime surface.
type CommandMenu struct {
	Title string
	Items []CommandItem
}

// AppCommands is a collection of command menus to install in the menu bar.
//
// Runtime surface.
type AppCommands struct {
	Menus []CommandMenu
}

// Commands creates an AppCommands from one or more command menus.
func Commands(menus ...CommandMenu) AppCommands {
	return AppCommands{Menus: menus}
}

// StandardFileMenu returns a File menu that ends with Close Window.
//
// Runtime surface.
//
// Any items passed before the standard Close Window item remain runner-owned
// Go commands. Close Window itself routes through the AppKit responder chain.
func StandardFileMenu(items ...CommandItem) CommandMenu {
	out := make([]CommandItem, 0, len(items)+2)
	out = append(out, items...)
	if len(out) > 0 && !out[len(out)-1].IsSeparator() {
		out = append(out, Separator())
	}
	out = append(out, CommandItem{
		Title:        "Close Window",
		Shortcut:     KeyboardShortcut{Key: "w", Modifiers: ModCommand},
		SystemAction: CommandSystemCloseWindow,
	})
	return CommandMenu{
		Title: "File",
		Items: out,
	}
}

// OpenRecentDocumentsMenu returns a submenu item that opens recent documents
// through a caller-provided path callback.
//
// Runtime surface.
//
// The submenu is always explicit. When no recent documents are available, it
// exposes a disabled placeholder item so the File menu still keeps a stable
// shape.
func OpenRecentDocumentsMenu(recent []DocumentRecent, open func(path string)) CommandItem {
	items := make([]CommandItem, 0, len(recent))
	for _, doc := range recent {
		if doc.Path == "" {
			continue
		}
		doc := doc
		title := doc.DisplayName
		if title == "" {
			title = filepath.Base(doc.Path)
		}
		if title == "" {
			title = doc.Path
		}
		item := CommandItem{Title: title}
		if open != nil {
			path := doc.Path
			item.Action = func(CommandContext) {
				open(path)
			}
		} else {
			item.Enabled = func() bool { return false }
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		items = append(items, CommandItem{
			Title:   "No Recent Documents",
			Enabled: func() bool { return false },
		})
	}
	return CommandItem{
		Title:    "Open Recent",
		Children: items,
	}
}

// StandardEditMenu returns an Edit menu with standard items: undo, redo,
// separator, cut, copy, paste, select all.
func StandardEditMenu() CommandMenu {
	return CommandMenu{
		Title: "Edit",
		Items: []CommandItem{
			{Title: "Undo", Shortcut: KeyboardShortcut{Key: "z", Modifiers: ModCommand}, SystemAction: CommandSystemUndo},
			{Title: "Redo", Shortcut: KeyboardShortcut{Key: "z", Modifiers: ModCommand | ModShift}, SystemAction: CommandSystemRedo},
			Separator(),
			{Title: "Cut", Shortcut: KeyboardShortcut{Key: "x", Modifiers: ModCommand}, SystemAction: CommandSystemCut},
			{Title: "Copy", Shortcut: KeyboardShortcut{Key: "c", Modifiers: ModCommand}, SystemAction: CommandSystemCopy},
			{Title: "Paste", Shortcut: KeyboardShortcut{Key: "v", Modifiers: ModCommand}, SystemAction: CommandSystemPaste},
			{Title: "Select All", Shortcut: KeyboardShortcut{Key: "a", Modifiers: ModCommand}, SystemAction: CommandSystemSelectAll},
		},
	}
}

// StandardWindowMenu returns a Window menu with standard items: minimize,
// zoom, separator, bring all to front.
func StandardWindowMenu() CommandMenu {
	return CommandMenu{
		Title: "Window",
		Items: []CommandItem{
			{Title: "Minimize", Shortcut: KeyboardShortcut{Key: "m", Modifiers: ModCommand}, SystemAction: CommandSystemMinimizeWindow},
			{Title: "Zoom", SystemAction: CommandSystemZoomWindow},
			Separator(),
			{Title: "Bring All to Front", SystemAction: CommandSystemBringAllToFront},
		},
	}
}

// StandardHelpMenu returns an empty Help menu shell.
func StandardHelpMenu() CommandMenu {
	return CommandMenu{
		Title: "Help",
		Items: nil,
	}
}

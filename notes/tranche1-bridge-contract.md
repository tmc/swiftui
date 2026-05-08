# Tranche 1 Bridge Contract: Commands, Menus, Lifecycle, AX Identifier

This document specifies the C-level bridge contract for Tranche 1. It covers
the JSON scene plan extensions and new C function signatures needed to support
declarative menus/commands, app lifecycle callbacks, and accessibility
identifiers.

Design principle: extend the JSON scene plan for declarative structures (menus,
lifecycle); add C functions only for imperative operations or per-view
modifiers.

---

## 1. Scene Plan JSON Extension: Commands

The top-level `SUIScenePlanPayload` gains a `"commands"` key alongside `"scenes"`.
Each command group maps to one top-level NSMenu header (Edit, View, Help, etc.).
The app menu (containing Quit, Settings, About) remains auto-generated; command
groups append after it.

```json
{
  "scenes": [ ... ],
  "commands": [
    {
      "title": "Edit",
      "items": [
        {
          "kind": "item",
          "title": "Undo",
          "shortcutKey": "z",
          "shortcutModifiers": 1048576,
          "actionCallbackID": 42,
          "enabledCallbackID": 43
        },
        {
          "kind": "item",
          "title": "Redo",
          "shortcutKey": "z",
          "shortcutModifiers": 1179648,
          "actionCallbackID": 44,
          "enabledCallbackID": 45
        },
        { "kind": "separator" },
        {
          "kind": "item",
          "title": "Find",
          "children": [
            {
              "kind": "item",
              "title": "Find...",
              "shortcutKey": "f",
              "shortcutModifiers": 1048576,
              "actionCallbackID": 50,
              "enabledCallbackID": 0
            },
            {
              "kind": "item",
              "title": "Find Next",
              "shortcutKey": "g",
              "shortcutModifiers": 1048576,
              "actionCallbackID": 51,
              "enabledCallbackID": 0
            }
          ]
        }
      ]
    },
    {
      "title": "Help",
      "items": [
        {
          "kind": "item",
          "title": "Release Notes",
          "shortcutKey": "",
          "shortcutModifiers": 0,
          "actionCallbackID": 60,
          "enabledCallbackID": 0
        }
      ]
    }
  ]
}
```

### Field reference

| Field | Type | Description |
|---|---|---|
| `commands` | `[CommandGroup]` | Top-level array. Each entry is one menu bar header. |
| `CommandGroup.title` | `string` | Menu bar header title (e.g. "Edit", "View"). |
| `CommandGroup.items` | `[CommandItem]` | Ordered list of menu items. |
| `CommandItem.kind` | `string` | `"item"` or `"separator"`. |
| `CommandItem.title` | `string` | Display title. Ignored for separators. |
| `CommandItem.shortcutKey` | `string` | Key equivalent (e.g. `"z"`, `"f"`, `","`). Empty string for no shortcut. |
| `CommandItem.shortcutModifiers` | `uint64` | Raw `NSEvent.ModifierFlags.rawValue`. Common values: Command = 1048576, Shift+Command = 1179648, Option+Command = 1572864. `0` means Command-only when shortcutKey is set. |
| `CommandItem.actionCallbackID` | `uint64` | Callback ID for the action. `0` means no action (parent of submenu). Uses the command callback trampoline (see section 5). |
| `CommandItem.enabledCallbackID` | `uint64` | Callback ID polled to determine enabled state. `0` means always enabled. Returns `int32`: 1 = enabled, 0 = disabled. Uses the bool-return trampoline (see section 5). |
| `CommandItem.children` | `[CommandItem]?` | If present and non-empty, this item becomes a submenu. Recursive. |

### Separator representation

A separator is simply `{"kind": "separator"}`. All other fields are ignored.

### Go-side structs (sketch, not implementation)

```
sceneRunPlan {
    Scenes   []sceneRunPlanScene   `json:"scenes"`
    Commands []sceneRunPlanCommand `json:"commands,omitempty"`
}

sceneRunPlanCommand {
    Title string                   `json:"title"`
    Items []sceneRunPlanCommandItem `json:"items"`
}

sceneRunPlanCommandItem {
    Kind              string                    `json:"kind"`
    Title             string                    `json:"title,omitempty"`
    ShortcutKey       string                    `json:"shortcutKey,omitempty"`
    ShortcutModifiers uint64                    `json:"shortcutModifiers,omitempty"`
    ActionCallbackID  uint64                    `json:"actionCallbackID,omitempty"`
    EnabledCallbackID uint64                    `json:"enabledCallbackID,omitempty"`
    Children          []sceneRunPlanCommandItem `json:"children,omitempty"`
}
```

---

## 2. New C Functions

Only two new C functions are needed. Everything else is JSON-driven.

### SUIAccessibilityIdentifier

```
SUIAccessibilityIdentifier(
    view: UnsafeRawPointer,
    id:   UnsafePointer<CChar>
) -> UnsafeRawPointer
```

Applies `.accessibilityIdentifier(id)` to the view and returns the modified
view. Standard view-modifier pattern: takes a view pointer, returns a new view
pointer.

Go declaration:

```
_SUIAccessibilityIdentifier func(uintptr, *byte) uintptr
```

### SUIUpdateMenuItemEnabled

```
SUIUpdateMenuItemEnabled(
    tag: Int32,
    enabled: Int32
) -> Void
```

Imperatively enables or disables a menu item by its tag after initial
construction. This is a fallback for cases where polling via
`enabledCallbackID` is insufficient (e.g. async state changes). Each menu item
built from the scene plan is assigned `tag = actionCallbackID` so it can be
found via `NSApp.mainMenu`.

Go declaration:

```
_SUIUpdateMenuItemEnabled func(int32, int32)
```

This function is optional for the initial implementation -- `enabledCallbackID`
polling covers the common case.

---

## 3. Lifecycle Callback IDs

App lifecycle events are delivered via the existing scene plan JSON, not via
new C functions. This keeps the contract declarative and consistent with how
`actionCallbackID` already works for scene visibility.

### JSON extension

Add a top-level `"lifecycle"` key to `SUIScenePlanPayload`:

```json
{
  "scenes": [ ... ],
  "commands": [ ... ],
  "lifecycle": {
    "didFinishLaunchingCallbackID": 100,
    "didBecomeActiveCallbackID": 101,
    "didResignActiveCallbackID": 102,
    "shouldTerminateCallbackID": 103,
    "willTerminateCallbackID": 104
  }
}
```

| Field | Type | Trampoline | Description |
|---|---|---|---|
| `didFinishLaunchingCallbackID` | `uint64` | button (fire-and-forget) | Called once after NSApplication finishes launching. |
| `didBecomeActiveCallbackID` | `uint64` | button (fire-and-forget) | Called on `applicationDidBecomeActive`. |
| `didResignActiveCallbackID` | `uint64` | button (fire-and-forget) | Called on `applicationWillResignActive`. |
| `shouldTerminateCallbackID` | `uint64` | command (returns int32) | Called on `applicationShouldTerminate`. Return 1 to allow, 0 to cancel. Uses the command callback trampoline. |
| `willTerminateCallbackID` | `uint64` | button (fire-and-forget) | Called on `applicationWillTerminate`. Last-chance cleanup. |

All callback IDs are optional; `0` means no callback registered.

### Swift-side behavior

The scene runner delegate (`SUISceneRunnerDelegate`) stores these callback IDs
after parsing `SUIScenePlanPayload` and invokes them from the corresponding
`NSApplicationDelegate` methods. For `shouldTerminate`, the delegate calls the
command callback trampoline and maps the int32 return to
`NSApplication.TerminateReply`.

### Go-side struct extension

```
sceneRunPlan {
    Scenes    []sceneRunPlanScene   `json:"scenes"`
    Commands  []sceneRunPlanCommand `json:"commands,omitempty"`
    Lifecycle *sceneRunPlanLifecycle `json:"lifecycle,omitempty"`
}

sceneRunPlanLifecycle {
    DidFinishLaunchingCallbackID uint64 `json:"didFinishLaunchingCallbackID,omitempty"`
    DidBecomeActiveCallbackID    uint64 `json:"didBecomeActiveCallbackID,omitempty"`
    DidResignActiveCallbackID    uint64 `json:"didResignActiveCallbackID,omitempty"`
    ShouldTerminateCallbackID    uint64 `json:"shouldTerminateCallbackID,omitempty"`
    WillTerminateCallbackID      uint64 `json:"willTerminateCallbackID,omitempty"`
}
```

---

## 4. Accessibility Identifier

See section 2 above for the C function signature. The Go-side API is a
standard view modifier:

```go
func (v View) AccessibilityIdentifier(id string) View
```

This is the only per-view accessibility primitive in Tranche 1. Additional
accessibility modifiers (label, hint, traits, rotor) are deferred to later
tranches.

---

## 5. Command Callback Trampoline

### Problem

The existing `buttonCallbackTrampoline` has signature `func(id uintptr)` --
fire-and-forget, no return value. Command callbacks need to return a status:

- Menu item actions may want to report whether they succeeded (for
  `shouldTerminate`, the return value determines whether the app quits).
- Enabled-check callbacks must return a bool (is this item enabled?).

### Solution: commandCallbackTrampoline

```
func commandCallbackTrampoline(id uintptr) int32
```

Go side:

```go
var (
    commandCallbackMap = map[uintptr]func() int32{}
)

func registerCommandCallback(fn func() int32) uintptr { ... }

func commandCallbackTrampoline(id uintptr) int32 {
    commandCallbackMu.Lock()
    fn := commandCallbackMap[id]
    commandCallbackMu.Unlock()
    if fn != nil {
        return fn()
    }
    return 1 // default: success / allow
}

var commandCallbackPtr = purego.NewCallback(commandCallbackTrampoline)
```

Swift side registers this trampoline via:

```
@_cdecl("SUIRegisterCommandCallback")
func SUIRegisterCommandCallback(_ fn: @convention(c) (UInt) -> Int32)
```

The function pointer is stored as `_SUICommandCallback` (analogous to
`_SUIStringCallback` and `_SUIButtonCallback`).

### Return value semantics

| Context | Return 1 | Return 0 |
|---|---|---|
| Menu item action | Success (no-op) | Failure (logged, no user-visible effect) |
| `shouldTerminate` | Allow termination | Cancel termination |
| Enabled check | Item enabled | Item disabled |

### Why not reuse existing trampolines?

- `buttonCallbackTrampoline(id)` returns void -- cannot carry status.
- `stringCallbackTrampoline(id, value)` takes a string parameter and returns
  int32, but command actions have no string argument.
- A dedicated `commandCallbackTrampoline(id) -> int32` is the minimal addition
  that covers all three use cases (action, shouldTerminate, enabled check)
  without overloading existing trampolines.

---

## Summary of bridge surface changes

| Category | Mechanism | Count |
|---|---|---|
| Menu commands | JSON `"commands"` in scene plan | 0 new C functions |
| Lifecycle | JSON `"lifecycle"` in scene plan | 0 new C functions |
| AX Identifier | `SUIAccessibilityIdentifier` C function | 1 new C function |
| Menu enable (optional) | `SUIUpdateMenuItemEnabled` C function | 1 new C function |
| Command trampoline | `SUIRegisterCommandCallback` C function | 1 new C function |
| **Total new C functions** | | **3** (1 optional) |
| **Total new callback maps** | `commandCallbackMap` | **1** |
| **Scene plan JSON keys added** | `commands`, `lifecycle` | **2** |

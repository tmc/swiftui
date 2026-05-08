//go:build darwin
// +build darwin

// Command scenes demonstrates the declarative scene helpers plus runner-owned
// app commands and menus.
//
// Usage:
//
//	go run .
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/tmc/swiftui"
)

func init() { runtime.LockOSThread() }

func main() {
	actionStatus := swiftui.NewStringState("No scene action invoked yet.")
	defer actionStatus.Release()
	closeCountState := swiftui.NewStringState("0")
	defer closeCountState.Release()
	refreshCount := swiftui.NewIntState(0)
	defer refreshCount.Release()
	sampleFolder, primaryDocument, secondaryDocument, notesDocument, err := prepareSceneDemoDocuments()
	if err != nil {
		log.Fatal(err)
	}
	initialBody, err := os.ReadFile(primaryDocument)
	if err != nil {
		log.Fatal(err)
	}
	documentBody := swiftui.NewStringState(string(initialBody))
	defer documentBody.Release()
	currentDocument := swiftui.NewStringState(primaryDocument)
	defer currentDocument.Release()
	currentDisplayName := swiftui.NewStringState(filepath.Base(primaryDocument))
	defer currentDisplayName.Release()
	lastRefresh := swiftui.NewStringState("never")
	defer lastRefresh.Release()
	currentSpace := swiftui.NewStringState("not opened")
	defer currentSpace.Release()
	lifecycleEvent := swiftui.NewStringState("launch pending")
	defer lifecycleEvent.Release()
	lifecycleCount := swiftui.NewIntState(0)
	defer lifecycleCount.Release()
	vetoQuit := swiftui.NewIntState(1)
	defer vetoQuit.Release()
	autoSave := swiftui.NewIntState(1)
	defer autoSave.Release()
	denseActivity := swiftui.NewIntState(0)
	defer denseActivity.Release()
	approvedCloseCount := 0

	menu := swiftui.VStackSpaced(10,
		swiftui.Text("MenuBarExtra").Font(swiftui.FontHeadline).FontWeight(swiftui.WeightSemibold),
		swiftui.Text("This scene plan lowers one document window, two auxiliary windows, one settings window, one preview surface, and one menu bar extra through RunScenes.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
		scenePlanRow("documents", "documents.primary", "primary document", "launch"),
		scenePlanRow("documents.inspector", "documents.inspector", "auxiliary inspector", "manual"),
		scenePlanRow("documents.activity", "documents.activity", "auxiliary activity", "launch"),
		scenePlanRow("settings", "settings", "settings window", "app menu"),
		scenePlanRow("demo.space", "demo.space", "auxiliary preview", "manual"),
		infoRowState("Current document", currentDisplayName),
		infoRow("Sample folder", sampleFolder),
		infoRowState("Last refresh", lastRefresh),
		infoRowState("Immersive preview", currentSpace),
		infoRowState("Lifecycle event", lifecycleEvent),
		infoRow("Lifecycle count", fmt.Sprintf("%d", lifecycleCount.Get())),
		infoRow("Quit veto", boolLabel(vetoQuit.Get() == 1)),
		swiftui.TextFromString(actionStatus).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView(),
	).Padding(14)

	var documentScene swiftui.DocumentGroupScene
	var inspectorScene swiftui.WindowGroupScene
	var activityScene swiftui.WindowGroupScene
	var immersivePreviewScene swiftui.WindowGroupScene

	focusScene := func(actions swiftui.SceneActions, id, label string) {
		err := actions.OpenWindow(id)
		if err == nil {
			actionStatus.Set("focused " + label)
			return
		}
		actionStatus.Set(renderActionResult(err))
	}

	openAnotherInspector := func() {
		err := inspectorScene.Actions().OpenWindow("documents.inspector")
		if err == nil {
			actionStatus.Set("opened another inspector window")
			return
		}
		actionStatus.Set(renderActionResult(err))
	}

	openDocument := func() {
		handle := documentScene.Handle()
		if handle.Open == nil {
			actionStatus.Set("document open unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.Open()))
	}

	openDocumentPath := func(path string) {
		handle := documentScene.Handle()
		if handle.OpenPath == nil {
			actionStatus.Set("document path-open unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.OpenPath(path)))
	}

	saveDocument := func() {
		handle := documentScene.Handle()
		if handle.Save == nil {
			actionStatus.Set("document save unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.Save()))
	}

	saveDocumentAs := func() {
		handle := documentScene.Handle()
		if handle.SaveAs == nil {
			actionStatus.Set("document save-as unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.SaveAs()))
	}

	exportDocument := func() {
		handle := documentScene.Handle()
		if handle.Export == nil {
			actionStatus.Set("document export unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.Export()))
	}

	importDocument := func() {
		handle := documentScene.Handle()
		if handle.Import == nil {
			actionStatus.Set("document import unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.Import()))
	}

	revertDocument := func() {
		handle := documentScene.Handle()
		if handle.Revert == nil {
			actionStatus.Set("document revert unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.Revert()))
	}

	closeDocument := func() {
		handle := documentScene.Handle()
		if handle.Close == nil {
			actionStatus.Set("document close unavailable")
			return
		}
		actionStatus.Set(renderActionResult(handle.Close()))
	}

	refreshDocument := func(actions swiftui.SceneActions) {
		err := actions.Refresh()
		if err == nil {
			nextCount := refreshCount.Get() + 1
			refreshCount.Set(nextCount)
			lastRefresh.Set(time.Now().Format("15:04:05"))
			actionStatus.Set(fmt.Sprintf("refreshed (%d)", nextCount))
			return
		}
		actionStatus.Set(renderActionResult(err))
	}

	openPreview := func(actions swiftui.SceneActions) {
		err := actions.OpenImmersiveSpace("demo.space")
		if err == nil {
			currentSpace.Set("demo.space")
			actionStatus.Set("opened demo.space")
			return
		}
		actionStatus.Set(renderActionResult(err))
	}

	documentScene = swiftui.DocumentGroupWithHandle("documents", swiftui.DocumentConfig{
		Title:  "Document Scene",
		Width:  620,
		Height: 560,
	}, func(handle swiftui.DocumentHandle, actions swiftui.SceneActions) swiftui.View {
		status := documentScene.Status()
		state := documentScene.RuntimeState()
		inspectorState := inspectorScene.RuntimeState()
		session := handle.Session
		path := session.Path
		if path == "" {
			path = "unsaved demo document"
		}
		return swiftui.VStackSpaced(12,
			swiftui.Text("DocumentGroup").Font(swiftui.FontTitle).FontWeight(swiftui.WeightBold),
			swiftui.Text("RunScenes now owns a real document session and file workflow here: the runner presents open/save panels, tracks dirty close policy, remembers recent documents, keeps session identity explicit in Go, and reports approved close paths back through the workflow.").
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary"),
			infoRow("Scene ID", status.ID),
			infoRow("Restoration ID", status.RestorationID),
			infoRow("Frame restore key", status.RestorationID),
			infoRow("Status active", boolLabel(status.Active)),
			infoRow("Restore visibility", boolLabel(status.RestoresVisibility != nil && *status.RestoresVisibility)),
			infoRow("Auxiliary policy", scenePolicyLabel(status.AuxiliaryPolicy)),
			infoRow("Lifecycle", string(state.Lifecycle)),
			infoRow("Live scene", boolLabel(state.Live)),
			infoRow("Session ID", session.ID),
			infoRow("Display name", session.DisplayName),
			infoRow("Path", path),
			infoRow("Dirty", boolLabel(handle.Dirty)),
			infoRow("Refresh count", fmt.Sprintf("%d", refreshCount.Get())),
			infoRow("Borrowed actions", boolLabel(state.ActionsAvailable())),
			infoRowState("Lifecycle event", lifecycleEvent),
			infoRow("Lifecycle count", fmt.Sprintf("%d", lifecycleCount.Get())),
			infoRow("Quit veto", boolLabel(vetoQuit.Get() == 1)),
			infoRow("Inspector instances", fmt.Sprintf("%d", inspectorState.WindowInstanceCount)),
			infoRow("Focused inspector", sceneWindowInstanceID(inspectorState.FocusedWindowInstanceID)),
			infoRow("Window action", boolLabel(actions.Window != nil)),
			infoRow("Handle open", boolLabel(handle.Open != nil)),
			infoRow("Handle open path", boolLabel(handle.OpenPath != nil)),
			infoRow("Handle save", boolLabel(handle.Save != nil)),
			infoRow("Handle revert", boolLabel(handle.Revert != nil)),
			infoRow("Handle import", boolLabel(handle.Import != nil)),
			infoRow("Handle close", boolLabel(handle.Close != nil)),
			infoRow("Refresh action", boolLabel(actions.RefreshScene != nil)),
			infoRow("Immersive action", boolLabel(actions.ImmersiveSpace != nil)),
			infoRow("Sample folder", sampleFolder),
			infoRow("Sample notes", notesDocument),
			infoRow("Sibling windows", "documents.inspector, documents.activity, demo.space"),
			infoRow("Declared auxiliary slots", "inspector/manual, activity/open on launch, preview/manual"),
			sceneWindowInstanceSection("Tracked inspector windows", inspectorState.WindowInstances),
			swiftui.HStackSpaced(8,
				swiftui.Button("Focus inspector", func() {
					focusScene(actions, "documents.inspector", "inspector")
				}),
				swiftui.Button("Focus activity", func() {
					focusScene(actions, "documents.activity", "activity")
				}),
				swiftui.Button("Open another inspector window", func() {
					openAnotherInspector()
				}),
				swiftui.Button("Open document…", func() {
					openDocument()
				}),
				swiftui.Button("Open strategy sample", func() {
					openDocumentPath(secondaryDocument)
				}),
				swiftui.Button("Save", func() {
					saveDocument()
				}),
				swiftui.Button("Revert", func() {
					revertDocument()
				}),
			),
			swiftui.HStackSpaced(8,
				swiftui.Button("Save as…", func() {
					saveDocumentAs()
				}),
				swiftui.Button("Export snapshot…", func() {
					exportDocument()
				}),
				swiftui.Button("Import notes…", func() {
					importDocument()
				}),
				swiftui.Button("Close via handle", func() {
					closeDocument()
				}),
				swiftui.Button("Refresh", func() {
					refreshDocument(actions)
				}),
				swiftui.Button("Missing window", func() {
					actionStatus.Set(renderActionResult(actions.OpenWindow("documents.missing")))
				}),
			),
			recentDocumentSection(handle, func(path string) {
				openDocumentPath(path)
			}),
			swiftui.TextEditorOnChange(documentBody, func() {
				if !documentScene.Handle().Dirty {
					documentScene.SetDirty(true)
					actionStatus.Set("edited current document")
				}
			}).
				Padding(6).
				Frame(580, 150).
				ScrollContentBackgroundHidden().
				BackgroundRoundedRect(0.14, 0.15, 0.18, 0.98, 12).
				ClipRoundedRect(12).
				AccessibilityIdentifier("scene-document-editor"),
			swiftui.HStackSpaced(8,
				swiftui.Button("Try immersive action", func() {
					openPreview(actions)
				}),
				swiftui.Text("On macOS, the current runner lowers immersive-space actions onto a named preview surface rather than native spatial presentation.").
					Font(swiftui.FontCaption).
					ForegroundStyleNamed("secondary").
					AsView(),
			),
			swiftui.TextFromString(actionStatus).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView(),
			swiftui.Text("Use File, View, Window, and Help menus above. File now routes through runner-owned document actions, including Open Recent, path-based open, revert, and a handle-driven close path that reports approved closes back to Go. App > Settings… reveals the concrete settings scene, WindowGroup can open additional inspector instances instead of reusing one window, and the runner restores each live inspector instance separately on relaunch so frame, visibility, focus, and instance identity stay explicit instead of hiding behind parity assumptions.").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
			swiftui.ShareLink("Open repository", "https://github.com/tmc/swiftui"),
		).Padding(18)
	}).WithHandle(swiftui.DocumentHandle{
		ID:          "documents",
		DisplayName: filepath.Base(primaryDocument),
		Path:        primaryDocument,
		Dirty:       false,
	}).WithDocumentWorkflow(swiftui.DocumentWorkflow{
		Open: func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			documentBody.Set(string(data))
			currentDocument.Set(path)
			currentDisplayName.Set(filepath.Base(path))
			actionStatus.Set("opened " + filepath.Base(path))
			return nil
		},
		Save: func(_ swiftui.DocumentSession, path string) error {
			if err := os.WriteFile(path, []byte(documentBody.Get()), 0o644); err != nil {
				return err
			}
			currentDocument.Set(path)
			currentDisplayName.Set(filepath.Base(path))
			actionStatus.Set("saved " + filepath.Base(path))
			return nil
		},
		Export: func(session swiftui.DocumentSession, path string) error {
			snapshot := fmt.Sprintf("Document: %s\nPath: %s\nDirty: %v\nLast refresh: %s\nPreview: %s\n\n%s",
				session.DisplayName,
				session.Path,
				documentScene.Handle().Dirty,
				lastRefresh.Get(),
				currentSpace.Get(),
				documentBody.Get(),
			)
			if err := os.WriteFile(path, []byte(snapshot), 0o644); err != nil {
				return err
			}
			actionStatus.Set("exported snapshot to " + filepath.Base(path))
			return nil
		},
		Import: func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			body := documentBody.Get()
			if body != "" {
				body += "\n\n"
			}
			documentBody.Set(body + string(data))
			documentScene.SetDirty(true)
			actionStatus.Set("imported " + filepath.Base(path))
			return nil
		},
		Close: func(session swiftui.DocumentSession) error {
			approvedCloseCount++
			closeCountState.Set(fmt.Sprintf("%d", approvedCloseCount))
			actionStatus.Set("approved close for " + session.DisplayName)
			return nil
		},
	}).WithRestorationID("documents.primary").WithVisibilityRestore(true)

	inspectorScene = swiftui.WindowGroupWithStatus("documents.inspector", swiftui.AppConfig{
		Title:  "Inspector",
		Width:  360,
		Height: 340,
	}, func(status swiftui.SceneStatus) swiftui.View {
		state := inspectorScene.RuntimeState()
		return swiftui.VStackSpaced(12,
			swiftui.Text("Inspector Window").Font(swiftui.FontTitle2).FontWeight(swiftui.WeightSemibold),
			swiftui.Text("This auxiliary window is declared alongside the document scene and owned by the same RunScenes plan.").
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary"),
			sceneStatusRows(status),
			infoRow("Window instances", fmt.Sprintf("%d", state.WindowInstanceCount)),
			infoRow("Focused window", sceneWindowInstanceID(state.FocusedWindowInstanceID)),
			sceneWindowInstanceSection("Tracked windows", state.WindowInstances),
			infoRow("Borrowed actions", boolLabel(state.ActionsAvailable())),
			swiftui.TextFromString(actionStatus).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView(),
			swiftui.HStackSpaced(8,
				swiftui.Button("Open another inspector window", func() {
					openAnotherInspector()
				}),
				swiftui.Button("Focus document", func() {
					focusScene(inspectorScene.Actions(), "documents", "document")
				}),
			),
			infoRowState("Tracked document", currentDisplayName),
			infoRowState("Path", currentDocument),
			swiftui.Text("Try the Focus inspector button in the document window to bring this window forward by scene ID.").
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		).Padding(18)
	}).
		WithRestorationID("documents.inspector").
		WithAuxiliaryWindowPolicy(swiftui.AuxiliaryWindowManual).
		WithVisibilityRestore(false)

	activityScene = swiftui.WindowGroupWithStatus("documents.activity", swiftui.AppConfig{
		Title:  "Activity",
		Width:  420,
		Height: 360,
	}, func(status swiftui.SceneStatus) swiftui.View {
		state := activityScene.RuntimeState()
		return swiftui.VStackSpaced(12,
			swiftui.Text("Activity Window").Font(swiftui.FontTitle2).FontWeight(swiftui.WeightSemibold),
			swiftui.Text("This sibling window proves the current runner can own more than one auxiliary window under one scene plan, and the runtime tracks each instance explicitly instead of flattening them into one borrowed scene flag.").
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary"),
			sceneStatusRows(status),
			infoRow("Window instances", fmt.Sprintf("%d", state.WindowInstanceCount)),
			infoRow("Focused window", sceneWindowInstanceID(state.FocusedWindowInstanceID)),
			sceneWindowInstanceSection("Tracked windows", state.WindowInstances),
			infoRow("Borrowed actions", boolLabel(state.ActionsAvailable())),
			infoRowState("Current document", currentDisplayName),
			infoRowState("Path", currentDocument),
			infoRowState("Last refresh", lastRefresh),
			infoRowState("Approved closes", closeCountState),
			swiftui.TextFromString(actionStatus).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView(),
		).Padding(18)
	}).
		WithRestorationID("documents.activity").
		WithAuxiliaryWindowPolicy(swiftui.AuxiliaryWindowOpenOnLaunch).
		WithVisibilityRestore(true)

	immersivePreviewScene = swiftui.WindowGroupWithStatus("demo.space", swiftui.AppConfig{
		Title:  "Immersive Preview",
		Width:  460,
		Height: 380,
	}, func(status swiftui.SceneStatus) swiftui.View {
		return swiftui.VStackSpaced(12,
			swiftui.Text("Immersive Preview").Font(swiftui.FontTitle2).FontWeight(swiftui.WeightSemibold),
			swiftui.Text("This macOS runner uses a named preview surface to exercise OpenImmersiveSpace without claiming native spatial-computing parity.").
				Font(swiftui.FontCallout).
				ForegroundStyleNamed("secondary"),
			sceneStatusRows(status),
			infoRowState("Current document", currentDisplayName),
			infoRowState("Path", currentDocument),
			infoRowState("Last refresh", lastRefresh),
			infoRowState("Space ID", currentSpace),
			swiftui.TextFromString(actionStatus).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary").
				AsView(),
		).Padding(18)
	}).
		WithRestorationID("demo.space").
		WithAuxiliaryWindowPolicy(swiftui.AuxiliaryWindowManual).
		WithVisibilityRestore(false)

	settingsScene := swiftui.Settings(swiftui.AppConfig{
		Title:  "Settings",
		Width:  520,
		Height: 340,
	}, swiftui.VStackSpaced(12,
		swiftui.Text("Settings").Font(swiftui.FontTitle2).FontWeight(swiftui.WeightSemibold),
		swiftui.Text("This concrete Settings scene is runner-owned. It opens from App > Settings… while the rest of the command surface stays in explicit Go runtime types.").
			Font(swiftui.FontCallout).
			ForegroundStyleNamed("secondary"),
		swiftui.Toggle("Auto-save scene-backed documents", autoSave, func() {
			actionStatus.Set("toggled auto-save preference")
		}),
		swiftui.Toggle("Dense activity timeline", denseActivity, func() {
			actionStatus.Set("toggled dense activity preference")
		}),
		swiftui.Toggle("Veto quit requests", vetoQuit, func() {
			actionStatus.Set("toggled quit veto preference")
		}),
		infoRowState("Tracked document", currentDisplayName),
		infoRowState("Last refresh", lastRefresh),
		infoRowState("Current preview", currentSpace),
		infoRowState("Lifecycle event", lifecycleEvent),
		infoRow("Lifecycle count", fmt.Sprintf("%d", lifecycleCount.Get())),
		infoRow("Quit veto", boolLabel(vetoQuit.Get() == 1)),
		swiftui.Text("Press ⌘, or choose App > Settings… to reveal this window.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"),
		swiftui.TextFromString(actionStatus).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary").
			AsView(),
	).Padding(18))

	commands := swiftui.Commands(
		swiftui.StandardFileMenu(
			swiftui.OpenRecentDocumentsMenu(documentScene.Handle().RecentDocuments, openDocumentPath),
			swiftui.CommandItem{
				Title:    "Open Document…",
				Shortcut: swiftui.KeyboardShortcut{Key: "o", Modifiers: swiftui.ModCommand},
				Action: func(swiftui.CommandContext) {
					openDocument()
				},
				Enabled: func() bool {
					return documentScene.Handle().Open != nil
				},
			},
			swiftui.CommandItem{
				Title: "Open Product Strategy",
				Action: func(swiftui.CommandContext) {
					openDocumentPath(secondaryDocument)
				},
				Enabled: func() bool {
					handle := documentScene.Handle()
					return handle.OpenPath != nil && handle.Path != secondaryDocument
				},
			},
			swiftui.CommandItem{
				Title:    "Save",
				Shortcut: swiftui.KeyboardShortcut{Key: "s", Modifiers: swiftui.ModCommand},
				Action: func(swiftui.CommandContext) {
					saveDocument()
				},
				Enabled: func() bool {
					handle := documentScene.Handle()
					return handle.Save != nil && handle.Dirty
				},
			},
			swiftui.CommandItem{
				Title:    "Save As…",
				Shortcut: swiftui.KeyboardShortcut{Key: "s", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					saveDocumentAs()
				},
				Enabled: func() bool {
					return documentScene.Handle().SaveAs != nil
				},
			},
			swiftui.CommandItem{
				Title:    "Revert to Saved",
				Shortcut: swiftui.KeyboardShortcut{Key: "r", Modifiers: swiftui.ModCommand},
				Action: func(swiftui.CommandContext) {
					revertDocument()
				},
				Enabled: func() bool {
					handle := documentScene.Handle()
					return handle.Revert != nil && handle.Dirty
				},
			},
			swiftui.CommandItem{
				Title:    "Import Notes…",
				Shortcut: swiftui.KeyboardShortcut{Key: "i", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					importDocument()
				},
				Enabled: func() bool {
					return documentScene.Handle().Import != nil
				},
			},
			swiftui.CommandItem{
				Title:    "Export Snapshot…",
				Shortcut: swiftui.KeyboardShortcut{Key: "e", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					exportDocument()
				},
				Enabled: func() bool {
					return documentScene.Handle().Export != nil
				},
			},
			swiftui.CommandItem{
				Title:    "Close Via Handle",
				Shortcut: swiftui.KeyboardShortcut{Key: "w", Modifiers: swiftui.ModCommand | swiftui.ModShift},
				Action: func(swiftui.CommandContext) {
					closeDocument()
				},
				Enabled: func() bool {
					return documentScene.Handle().Close != nil
				},
			},
		),
		swiftui.StandardEditMenu(),
		swiftui.CommandMenu{
			Title: "View",
			Items: []swiftui.CommandItem{
				{
					Title:    "Show Inspector",
					Shortcut: swiftui.KeyboardShortcut{Key: "1", Modifiers: swiftui.ModCommand},
					Action: func(swiftui.CommandContext) {
						focusScene(documentScene.Actions(), "documents.inspector", "inspector")
					},
					Enabled: func() bool {
						return documentScene.Actions().Window != nil && !inspectorScene.Status().Active
					},
				},
				{
					Title:    "Show Activity",
					Shortcut: swiftui.KeyboardShortcut{Key: "2", Modifiers: swiftui.ModCommand},
					Action: func(swiftui.CommandContext) {
						focusScene(documentScene.Actions(), "documents.activity", "activity")
					},
					Enabled: func() bool {
						return documentScene.Actions().Window != nil && !activityScene.Status().Active
					},
				},
				{
					Title:    "Open Preview Surface",
					Shortcut: swiftui.KeyboardShortcut{Key: "p", Modifiers: swiftui.ModCommand | swiftui.ModShift},
					Action: func(swiftui.CommandContext) {
						openPreview(documentScene.Actions())
					},
					Enabled: func() bool {
						return documentScene.Actions().ImmersiveSpace != nil && !immersivePreviewScene.Status().Active
					},
				},
			},
		},
		swiftui.StandardWindowMenu(),
		swiftui.CommandMenu{
			Title: "Help",
			Items: []swiftui.CommandItem{
				{
					Title: "Explain Scene Runner",
					Action: func(swiftui.CommandContext) {
						actionStatus.Set("RunScenes owns the app shell; scenes stay declarative specs")
					},
				},
			},
		},
	)

	err = swiftui.RunScenes(
		documentScene,
		inspectorScene,
		activityScene,
		settingsScene,
		immersivePreviewScene,
		swiftui.MenuBarExtra(swiftui.MenuBarConfig{
			Label:        "Scenes",
			SystemImage:  "rectangle.3.group",
			Width:        320,
			Height:       180,
			OpenOnLaunch: false,
		}, menu),
		swiftui.WithCommands(commands),
		swiftui.WithLifecycle(swiftui.AppLifecycle{
			OnLaunched: func() {
				lifecycleCount.Set(lifecycleCount.Get() + 1)
				lifecycleEvent.Set("launched")
			},
			OnActivate: func() {
				lifecycleCount.Set(lifecycleCount.Get() + 1)
				lifecycleEvent.Set("activated")
			},
			OnResignActive: func() {
				lifecycleCount.Set(lifecycleCount.Get() + 1)
				lifecycleEvent.Set("resigned active")
			},
			ShouldTerminate: func() bool {
				lifecycleCount.Set(lifecycleCount.Get() + 1)
				lifecycleEvent.Set("terminate requested")
				if vetoQuit.Get() == 1 {
					actionStatus.Set("terminate request observed and vetoed")
					return false
				}
				actionStatus.Set("terminate request approved")
				return true
			},
			OnTerminate: func() {
				lifecycleCount.Set(lifecycleCount.Get() + 1)
				lifecycleEvent.Set("terminated")
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func infoRow(label, value string) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).FontWeight(swiftui.WeightSemibold),
		swiftui.Spacer(),
		swiftui.Text(value).Font(swiftui.FontCaption),
	)
}

func infoRowState(label string, value *swiftui.StringState) swiftui.View {
	return swiftui.HStack(
		swiftui.Text(label).FontWeight(swiftui.WeightSemibold),
		swiftui.Spacer(),
		swiftui.TextFromString(value).Font(swiftui.FontCaption).AsView(),
	)
}

func sceneStatusRows(status swiftui.SceneStatus) swiftui.View {
	return swiftui.VStackSpaced(6,
		infoRow("Scene ID", status.ID),
		infoRow("Restoration ID", status.RestorationID),
		infoRow("Status active", boolLabel(status.Active)),
		infoRow("Restore visibility", boolLabel(status.RestoresVisibility != nil && *status.RestoresVisibility)),
		infoRow("Launch policy", scenePolicyLabel(status.AuxiliaryPolicy)),
	)
}

func scenePlanRow(id, restorationID, role, policy string) swiftui.View {
	return swiftui.VStackSpaced(4,
		infoRow("Scene", id),
		infoRow("Restore", restorationID),
		infoRow("Role", role),
		infoRow("Launch", policy),
	).
		Padding(10).
		Background(0.12, 0.14, 0.18, 0.92).
		CornerRadius(10)
}

func renderActionResult(err error) string {
	if err != nil {
		return err.Error()
	}
	return "ok"
}

func boolLabel(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func scenePolicyLabel(policy swiftui.AuxiliaryWindowPolicy) string {
	if policy == "" {
		return "n/a"
	}
	return string(policy)
}

func sceneWindowInstanceID(id string) string {
	if id == "" {
		return "none"
	}
	return id
}

func sceneWindowInstanceSection(label string, instances []swiftui.SceneWindowInstanceState) swiftui.View {
	rows := []swiftui.Viewable{
		swiftui.Text(label).Font(swiftui.FontHeadline),
	}
	if len(instances) == 0 {
		rows = append(rows, swiftui.Text("No live window instances are currently tracked.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"))
		return swiftui.VStackSpaced(6, rows...).Padding(10).Background(0.12, 0.14, 0.18, 0.92).CornerRadius(10)
	}
	for _, instance := range instances {
		rows = append(rows,
			swiftui.Text(instance.ID).Font(swiftui.FontCaption),
			swiftui.Text(sceneWindowInstanceDetail(instance)).
				Font(swiftui.FontCaption).
				ForegroundStyleNamed("secondary"),
		)
	}
	return swiftui.VStackSpaced(6, rows...).Padding(10).Background(0.12, 0.14, 0.18, 0.92).CornerRadius(10)
}

func sceneWindowInstanceDetail(instance swiftui.SceneWindowInstanceState) string {
	status := "visible"
	if !instance.Visible {
		status = "hidden"
	}
	if instance.Focused {
		status += ", focused"
	}
	return status
}

func recentDocumentSection(handle swiftui.DocumentHandle, openDocument func(string)) swiftui.View {
	rows := []swiftui.Viewable{
		swiftui.Text("Recent documents").Font(swiftui.FontHeadline),
	}
	if len(handle.RecentDocuments) == 0 {
		rows = append(rows, swiftui.Text("No recent documents yet. Open or save another file to populate the runner-owned list; the list persists by scene and restores on relaunch.").
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"))
		return swiftui.VStackSpaced(6, rows...).Padding(10).Background(0.12, 0.14, 0.18, 0.92).CornerRadius(10)
	}
	for _, recent := range handle.RecentDocuments {
		recent := recent
		title := recent.DisplayName
		if title == "" {
			title = filepath.Base(recent.Path)
		}
		rows = append(rows, swiftui.Button(title, func() {
			openDocument(recent.Path)
		}).ButtonStyle(swiftui.ButtonStylePlain))
		rows = append(rows, swiftui.Text(recent.Path).
			Font(swiftui.FontCaption).
			ForegroundStyleNamed("secondary"))
	}
	return swiftui.VStackSpaced(6, rows...).Padding(10).Background(0.12, 0.14, 0.18, 0.92).CornerRadius(10)
}

func prepareSceneDemoDocuments() (dir, primary, secondary, notes string, err error) {
	dir, err = os.MkdirTemp("", "swiftui-scenes-*")
	if err != nil {
		return "", "", "", "", err
	}
	primary = filepath.Join(dir, "quarterly-review.md")
	secondary = filepath.Join(dir, "product-strategy.md")
	notes = filepath.Join(dir, "research-notes.txt")
	files := map[string]string{
		primary:   "# Quarterly Review\n\n- Revenue beat plan by 8%\n- Focus the next sprint on reliability work\n",
		secondary: "# Product Strategy\n\n1. Tighten the document runtime\n2. Improve accessibility and testing hooks\n",
		notes:     "Imported notes:\n- Verify dirty-close behavior\n- Export a snapshot before filing the review\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return "", "", "", "", err
		}
	}
	return dir, primary, secondary, notes, nil
}

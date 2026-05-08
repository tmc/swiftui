import AppKit
import SwiftUI

// Scene plan runtime: wire types, window management, delegate, and entry points.

// MARK: - Global state

nonisolated(unsafe) var _sceneRunnerDelegate: SUISceneRunnerDelegate?
nonisolated(unsafe) var _sceneWindows: [String: NSWindow] = [:]
nonisolated(unsafe) var _sceneWindowSpecs: [String: SUIScenePlanScene] = [:]
nonisolated(unsafe) var _sceneWindowViews: [String: AnyView] = [:]
nonisolated(unsafe) var _sceneWindowFamiliesByInstanceID: [String: String] = [:]
nonisolated(unsafe) var _sceneWindowInstanceIDsByFamilyID: [String: [String]] = [:]
nonisolated(unsafe) var _sceneWindowNextInstanceIndexByFamilyID: [String: Int] = [:]
nonisolated(unsafe) var _sceneDocumentStates: [String: SUISceneDocumentState] = [:]
nonisolated(unsafe) var _sceneSettingsID: String?
nonisolated(unsafe) var _sceneFocusedWindowID: String?
nonisolated(unsafe) var _sceneWindowDelegates: [String: SUISceneWindowDelegate] = [:]
nonisolated(unsafe) var _sceneApprovedCloseIDs: Set<String> = []

@inline(__always)
func suiOnMainSync<T>(_ body: () -> T) -> T {
  if Thread.isMainThread {
    return body()
  }
  return DispatchQueue.main.sync(execute: body)
}

// MARK: - Wire types

struct SUIScenePlanPayload: Decodable {
  let scenes: [SUIScenePlanScene]
  let commands: [SUICommandGroup]?
  let lifecycle: SUILifecycleCallbacks?
}

struct SUICommandGroup: Decodable {
  let title: String
  let items: [SUICommandItem]
}

struct SUICommandItem: Decodable {
  let kind: String?
  let title: String?
  let shortcutKey: String?
  let shortcutModifiers: UInt64?
  let systemAction: String?
  let actionCallbackID: UInt64?
  let enabledCallbackID: UInt64?
  let children: [SUICommandItem]?
}

struct SUILifecycleCallbacks: Decodable {
  let didFinishLaunchingCallbackID: UInt64?
  let didBecomeActiveCallbackID: UInt64?
  let didResignActiveCallbackID: UInt64?
  let shouldTerminateCallbackID: UInt64?
  let willTerminateCallbackID: UInt64?
}

struct SUISceneDocumentState {
  var displayName: String
  var path: String
  var dirty: Bool
}

struct SUIScenePlanScene: Decodable {
  let kind: String
  let id: String?
  let restorationID: String?
  let title: String?
  let width: Double?
  let height: Double?
  let label: String?
  let systemImage: String?
  let openOnLaunch: Bool?
  let restoreVisibility: Bool?
  let multipleInstances: Bool?
  let actionCallbackID: UInt64?
  let documentDisplayName: String?
  let documentPath: String?
  let documentDirty: Bool?
  let documentOpenCallbackID: UInt64?
  let documentSaveCallbackID: UInt64?
  let documentExportCallbackID: UInt64?
  let documentImportCallbackID: UInt64?
  let documentCloseCallbackID: UInt64?
  let documentDirtyCallbackID: UInt64?
  let viewIndex: Int
}

// MARK: - Scene runner delegate

class SUISceneRunnerDelegate: NSObject, NSApplicationDelegate {
  var terminateAfterLastWindowClosedValue = true
  var didFinishLaunchingCallbackID: UInt64 = 0
  var didBecomeActiveCallbackID: UInt64 = 0
  var didResignActiveCallbackID: UInt64 = 0
  var shouldTerminateCallbackID: UInt64 = 0
  var willTerminateCallbackID: UInt64 = 0

  func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return terminateAfterLastWindowClosedValue
  }

  func applicationDidFinishLaunching(_ notification: Notification) {
    guard didFinishLaunchingCallbackID != 0 else { return }
    _SUIButtonCallback?(UInt(didFinishLaunchingCallbackID))
  }

  func applicationDidBecomeActive(_ notification: Notification) {
    if didBecomeActiveCallbackID != 0 {
      _SUIButtonCallback?(UInt(didBecomeActiveCallbackID))
    }
    SUIInvalidateCommandMenus()
  }

  func applicationWillResignActive(_ notification: Notification) {
    if didResignActiveCallbackID != 0 {
      _SUIButtonCallback?(UInt(didResignActiveCallbackID))
    }
    SUIInvalidateCommandMenus()
  }

  func applicationWillTerminate(_ notification: Notification) {
    guard willTerminateCallbackID != 0 else { return }
    _SUIButtonCallback?(UInt(willTerminateCallbackID))
  }

  func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
    if shouldTerminateCallbackID != 0, let fn = _SUICommandCallback {
      let result = fn(UInt(shouldTerminateCallbackID))
      if result == 0 {
        return .terminateCancel
      }
    }
    return SUIConfirmTerminateDocumentWindows() ? .terminateNow : .terminateCancel
  }

  @MainActor @objc func togglePopover(_ sender: AnyObject?) {
    guard let popover = _popover, let button = _statusItem?.button else { return }
    if popover.isShown {
      popover.performClose(sender)
    } else {
      popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
      NSApp.activate()
    }
  }

  @MainActor func openPopoverOnLaunch() {
    guard let popover = _popover, let button = _statusItem?.button else { return }
    if !popover.isShown {
      popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)
      NSApp.activate()
    }
  }

  @MainActor @objc func openSettings(_ sender: AnyObject?) {
    guard let sceneID = _sceneSettingsID else { return }
    _ = SUIRevealSceneWindow(sceneID)
  }
}

@MainActor
private func suiSceneWindowAllowsMultipleInstances(_ scene: SUIScenePlanScene) -> Bool {
  scene.multipleInstances ?? false
}

@MainActor
private func suiSceneWindowInstancesStoreKey(_ scene: SUIScenePlanScene) -> String {
  "swiftui.scene.window.instances." + suiScenePersistenceKey(scene)
}

@MainActor
private func suiSceneWindowVisibleStoreKey(_ scene: SUIScenePlanScene, _ instanceID: String) -> String {
  if suiSceneWindowAllowsMultipleInstances(scene) {
    return "swiftui.scene.visible." + instanceID
  }
  return suiSceneVisibleKey(scene)
}

@MainActor
private func suiSceneWindowFamilyInstanceCount(_ familyID: String) -> Int {
  _sceneWindowInstanceIDsByFamilyID[familyID]?.count ?? 0
}

@MainActor
private func suiSceneWindowNextInstanceID(_ familyID: String) -> String {
  let next = (_sceneWindowNextInstanceIndexByFamilyID[familyID] ?? 0) + 1
  _sceneWindowNextInstanceIndexByFamilyID[familyID] = next
  return "\(familyID).instance.\(next)"
}

@MainActor
private func suiSceneRegisterWindowInstance(_ familyID: String, _ instanceID: String) {
  _sceneWindowFamiliesByInstanceID[instanceID] = familyID
  var instances = _sceneWindowInstanceIDsByFamilyID[familyID] ?? []
  if !instances.contains(instanceID) {
    instances.append(instanceID)
  }
  _sceneWindowInstanceIDsByFamilyID[familyID] = instances
  if let suffix = suiSceneWindowInstanceSuffix(instanceID) {
    let next = max(_sceneWindowNextInstanceIndexByFamilyID[familyID] ?? 0, suffix)
    _sceneWindowNextInstanceIndexByFamilyID[familyID] = next
  }
  if let scene = _sceneWindowSpecs[familyID] {
    let key = suiSceneWindowInstancesStoreKey(scene)
    UserDefaults.standard.set(instances, forKey: key)
  }
}

@MainActor
private func suiSceneUnregisterWindowInstance(_ instanceID: String) -> String? {
  guard let familyID = _sceneWindowFamiliesByInstanceID.removeValue(forKey: instanceID) else {
    return nil
  }
  if var instances = _sceneWindowInstanceIDsByFamilyID[familyID] {
    instances.removeAll { $0 == instanceID }
    if instances.isEmpty {
      _sceneWindowInstanceIDsByFamilyID.removeValue(forKey: familyID)
    } else {
      _sceneWindowInstanceIDsByFamilyID[familyID] = instances
    }
    if let scene = _sceneWindowSpecs[familyID] {
      let key = suiSceneWindowInstancesStoreKey(scene)
      if instances.isEmpty {
        UserDefaults.standard.removeObject(forKey: key)
      } else {
        UserDefaults.standard.set(instances, forKey: key)
      }
    }
  }
  _sceneWindowDelegates.removeValue(forKey: instanceID)
  _sceneApprovedCloseIDs.remove(instanceID)
  return familyID
}

@MainActor
private func suiSceneWindowInstanceSuffix(_ instanceID: String) -> Int? {
  guard let range = instanceID.range(of: ".instance.") else { return nil }
  return Int(instanceID[range.upperBound...])
}

@MainActor
private func suiSceneRestoreWindowInstanceIDs(_ scene: SUIScenePlanScene) -> [String] {
  let key = suiSceneWindowInstancesStoreKey(scene)
  let defaults = UserDefaults.standard
  guard let stored = defaults.array(forKey: key) as? [String] else {
    return []
  }
  return stored.filter { !$0.isEmpty }
}

@MainActor
private func suiSceneWindowInstanceVisibility(_ scene: SUIScenePlanScene, _ instanceID: String) -> Bool {
  let key = suiSceneWindowVisibleStoreKey(scene, instanceID)
  let defaults = UserDefaults.standard
  if defaults.object(forKey: key) != nil {
    return defaults.bool(forKey: key)
  }
  return true
}

@MainActor
private func suiSceneWindowKeyIdentifier() -> String? {
  NSApp.keyWindow?.identifier?.rawValue
}

@MainActor
private func suiSceneWindowFamilyHasKeyWindow(_ familyID: String) -> Bool {
  guard let keyID = suiSceneWindowKeyIdentifier(),
    let keyFamily = _sceneWindowFamiliesByInstanceID[keyID]
  else {
    return false
  }
  return keyFamily == familyID
}

@MainActor
private func suiSceneWindowActionEvent(
  familyID: String, kind: String, instanceID: String? = nil, visible: Bool? = nil
) -> String {
  let count = suiSceneWindowFamilyInstanceCount(familyID)
  var components: [String] = []
  if kind != "unavailable" {
    components.append("window,document,refresh,immersive")
  }
  components.append("count=\(count)")
  if let instanceID, !instanceID.isEmpty {
    components.append("instance=\(instanceID)")
  }
  if let visible {
    components.append("visible=\(visible ? 1 : 0)")
  }
  return "\(kind):" + components.joined(separator: ";")
}

// MARK: - Window delegate

final class SUISceneWindowDelegate: NSObject, NSWindowDelegate {
  let sceneID: String
  let instanceID: String

  init(sceneID: String, instanceID: String) {
    self.sceneID = sceneID
    self.instanceID = instanceID
  }

  func windowDidMove(_ notification: Notification) {
    guard let window = notification.object as? NSWindow else { return }
    guard let scene = _sceneWindowSpecs[sceneID] else { return }
    let persistenceKey = suiSceneWindowPersistenceKey(scene, instanceID)
    window.saveFrame(usingName: persistenceKey)
  }

  func windowDidResize(_ notification: Notification) {
    guard let window = notification.object as? NSWindow else { return }
    guard let scene = _sceneWindowSpecs[sceneID] else { return }
    let persistenceKey = suiSceneWindowPersistenceKey(scene, instanceID)
    window.saveFrame(usingName: persistenceKey)
  }

  func windowShouldClose(_ sender: NSWindow) -> Bool {
    SUIConfirmCloseSceneWindow(sceneID, instanceID, sender)
  }

  func windowWillClose(_ notification: Notification) {
    guard let window = notification.object as? NSWindow else { return }
    guard let scene = _sceneWindowSpecs[sceneID] else { return }
    window.saveFrame(usingName: suiSceneWindowPersistenceKey(scene, instanceID))
    let familyID = suiSceneUnregisterWindowInstance(instanceID) ?? sceneID
    if _sceneFocusedWindowID == instanceID {
      _sceneFocusedWindowID = nil
    }
    if suiSceneShouldRestoreVisibility(scene) {
      UserDefaults.standard.set(false, forKey: suiSceneWindowVisibleStoreKey(scene, instanceID))
    } else {
      UserDefaults.standard.removeObject(forKey: suiSceneWindowVisibleStoreKey(scene, instanceID))
    }
    let remainingCount = suiSceneWindowFamilyInstanceCount(familyID)
    let eventKind = remainingCount == 0
      ? "unavailable"
      : (suiSceneWindowFamilyHasKeyWindow(familyID) ? "focused" : "blurred")
    let event = suiSceneWindowActionEvent(
      familyID: familyID, kind: eventKind, instanceID: instanceID, visible: false)
    SUIInvokeSceneActionCallback(scene.actionCallbackID, event)
    SUIInvalidateCommandMenus()
  }

  func windowDidBecomeKey(_ notification: Notification) {
    _sceneFocusedWindowID = instanceID
    guard let scene = _sceneWindowSpecs[sceneID] else { return }
    let event = suiSceneWindowActionEvent(
      familyID: sceneID, kind: "focused", instanceID: instanceID, visible: true)
    SUIInvokeSceneActionCallback(scene.actionCallbackID, event)
    SUIInvalidateCommandMenus()
  }

  func windowDidResignKey(_ notification: Notification) {
    if _sceneFocusedWindowID == instanceID {
      _sceneFocusedWindowID = nil
    }
    guard let scene = _sceneWindowSpecs[sceneID] else { return }
    let kind = suiSceneWindowFamilyHasKeyWindow(sceneID) ? "focused" : "blurred"
    let event = suiSceneWindowActionEvent(
      familyID: sceneID, kind: kind, instanceID: instanceID, visible: true)
    SUIInvokeSceneActionCallback(scene.actionCallbackID, event)
    SUIInvalidateCommandMenus()
  }
}

// MARK: - Scene action bridge view

@MainActor
private func SUIInvokeSceneActionCallback(_ callbackID: UInt64?, _ event: String) {
  guard let callbackID, callbackID != 0, let fn = _SUIStringCallback else { return }
  event.withCString { cstr in
    _ = fn(UInt(callbackID), cstr)
  }
}

struct SUISceneActionBridgeView: View {
  let callbackID: UInt64?
  let content: AnyView

  var body: some View {
    content
  }
}

// MARK: - Document helpers

private let suiDocumentActionSuccess: Int32 = 1
private let suiDocumentActionFailed: Int32 = 0
private let suiDocumentActionCanceled: Int32 = -1

@MainActor
private func suiSceneDocumentState(_ scene: SUIScenePlanScene) -> SUISceneDocumentState {
  let fallbackDisplayName: String
  if let documentDisplayName = scene.documentDisplayName, !documentDisplayName.isEmpty {
    fallbackDisplayName = documentDisplayName
  } else if let title = scene.title, !title.isEmpty {
    fallbackDisplayName = title
  } else {
    fallbackDisplayName = "Untitled"
  }
  let fallback = SUISceneDocumentState(
    displayName: fallbackDisplayName,
    path: scene.documentPath ?? "",
    dirty: scene.documentDirty ?? false
  )
  guard let id = scene.id else { return fallback }
  return _sceneDocumentStates[id] ?? fallback
}

@MainActor
private func suiSetSceneDocumentState(
  _ sceneID: String, _ scene: SUIScenePlanScene?, _ state: SUISceneDocumentState
) {
  _sceneDocumentStates[sceneID] = state
  if let window = _sceneWindows[sceneID] {
    if !state.displayName.isEmpty {
      window.title = state.displayName
    } else if let scene, let title = scene.title, !title.isEmpty {
      window.title = title
    }
  }
  SUIInvalidateCommandMenus()
}

@MainActor
private func suiScenePersistenceKey(_ scene: SUIScenePlanScene) -> String {
  if let restorationID = scene.restorationID?.trimmingCharacters(in: .whitespacesAndNewlines),
    !restorationID.isEmpty
  {
    return restorationID
  }
  if let id = scene.id?.trimmingCharacters(in: .whitespacesAndNewlines), !id.isEmpty {
    return id
  }
  return ""
}

@MainActor
private func suiScenePersistenceKey(_ sceneID: String) -> String {
  guard let scene = _sceneWindowSpecs[sceneID] else { return sceneID }
  return suiScenePersistenceKey(scene)
}

@MainActor
private func suiSceneDocumentRestoreKey(_ scene: SUIScenePlanScene) -> String {
  "swiftui.scene.document.path." + suiScenePersistenceKey(scene)
}

@MainActor
private func suiPersistSceneDocumentState(
  _ scene: SUIScenePlanScene, _ state: SUISceneDocumentState
) {
  let defaults = UserDefaults.standard
  let key = suiSceneDocumentRestoreKey(scene)
  if state.path.isEmpty {
    defaults.removeObject(forKey: key)
    return
  }
  defaults.set(state.path, forKey: key)
}

@MainActor
private func suiNoteRecentDocumentURL(_ url: URL) {
  guard url.isFileURL else { return }
  NSDocumentController.shared.noteNewRecentDocumentURL(url)
}

@MainActor
private func suiRestoreSceneDocumentState(_ scene: SUIScenePlanScene)
  -> SUISceneDocumentState?
{
  guard let callbackID = scene.documentOpenCallbackID, callbackID != 0 else { return nil }
  let defaults = UserDefaults.standard
  let key = suiSceneDocumentRestoreKey(scene)
  guard let path = defaults.string(forKey: key), !path.isEmpty else {
    return nil
  }
  guard suiInvokeDocumentPathCallback(callbackID, path) else {
    defaults.removeObject(forKey: key)
    return nil
  }
  let restored = SUISceneDocumentState(
    displayName: URL(fileURLWithPath: path).lastPathComponent,
    path: path,
    dirty: false
  )
  return restored
}

@MainActor
private func suiInvokeDocumentPathCallback(_ callbackID: UInt64?, _ path: String) -> Bool {
  guard let callbackID, callbackID != 0, let fn = _SUIStringCallback else { return false }
  return path.withCString { cstr in
    fn(UInt(callbackID), cstr) != 0
  }
}

@MainActor
private func suiDocumentIsDirty(_ scene: SUIScenePlanScene) -> Bool {
  if let callbackID = scene.documentDirtyCallbackID, callbackID != 0, let fn = _SUICommandCallback {
    return fn(UInt(callbackID)) != 0
  }
  return suiSceneDocumentState(scene).dirty
}

@MainActor
private func suiSuggestedDocumentName(_ scene: SUIScenePlanScene, _ state: SUISceneDocumentState)
  -> String
{
  if !state.displayName.isEmpty {
    return state.displayName
  }
  if !state.path.isEmpty {
    return URL(fileURLWithPath: state.path).lastPathComponent
  }
  if let title = scene.title, !title.isEmpty {
    return title
  }
  return "Untitled"
}

@MainActor
private func suiPresentDocumentFailureAlert(_ scene: SUIScenePlanScene, _ operation: String) {
  let alert = NSAlert()
  alert.alertStyle = .warning
  alert.messageText = "Could not \(operation)"
  alert.informativeText =
    "The runner-owned document action did not complete. The current document session was left unchanged."
  alert.addButton(withTitle: "OK")
  if let window = scene.id.flatMap({ _sceneWindows[$0] }) {
    alert.beginSheetModal(for: window)
  } else {
    alert.runModal()
  }
}

@MainActor
private func suiApplySceneDocumentOpen(
  _ sceneID: String,
  _ scene: SUIScenePlanScene,
  _ path: String,
  failureOperation: String
) -> Int32 {
  let trimmedPath = path.trimmingCharacters(in: .whitespacesAndNewlines)
  guard !trimmedPath.isEmpty else { return suiDocumentActionFailed }
  guard let callbackID = scene.documentOpenCallbackID, callbackID != 0 else {
    return suiDocumentActionFailed
  }
  guard suiInvokeDocumentPathCallback(callbackID, trimmedPath) else {
    suiPresentDocumentFailureAlert(scene, failureOperation)
    return suiDocumentActionFailed
  }
  let updated = SUISceneDocumentState(
    displayName: URL(fileURLWithPath: trimmedPath).lastPathComponent,
    path: trimmedPath,
    dirty: false
  )
  suiSetSceneDocumentState(sceneID, scene, updated)
  suiPersistSceneDocumentState(scene, updated)
  suiNoteRecentDocumentURL(URL(fileURLWithPath: trimmedPath))
  return suiDocumentActionSuccess
}

@MainActor
private func suiRunSavePanel(
  _ scene: SUIScenePlanScene, _ state: SUISceneDocumentState, operation: String
) -> URL? {
  let panel = NSSavePanel()
  if !state.path.isEmpty {
    let currentURL = URL(fileURLWithPath: state.path)
    panel.directoryURL = currentURL.deletingLastPathComponent()
    panel.nameFieldStringValue = currentURL.lastPathComponent
  } else {
    panel.nameFieldStringValue = suiSuggestedDocumentName(scene, state)
  }
  panel.canCreateDirectories = true
  panel.title = operation
  guard panel.runModal() == .OK else { return nil }
  return panel.url
}

@MainActor
private func suiPerformSceneDocumentSave(
  _ sceneID: String, _ scene: SUIScenePlanScene, forcePanel: Bool
) -> Int32 {
  let state = suiSceneDocumentState(scene)
  guard let callbackID = scene.documentSaveCallbackID, callbackID != 0 else {
    return suiDocumentActionFailed
  }
  let targetPath: String
  if forcePanel || state.path.isEmpty {
    guard
      let targetURL = suiRunSavePanel(
        scene, state, operation: forcePanel ? "Save Document As" : "Save Document")
    else {
      return suiDocumentActionCanceled
    }
    targetPath = targetURL.path
  } else {
    targetPath = state.path
  }
  if !suiInvokeDocumentPathCallback(callbackID, targetPath) {
    suiPresentDocumentFailureAlert(
      scene, forcePanel || state.path.isEmpty ? "save the document" : "save changes")
    return suiDocumentActionFailed
  }
  let updated = SUISceneDocumentState(
    displayName: URL(fileURLWithPath: targetPath).lastPathComponent,
    path: targetPath,
    dirty: false
  )
  suiSetSceneDocumentState(sceneID, scene, updated)
  suiPersistSceneDocumentState(scene, updated)
  suiNoteRecentDocumentURL(URL(fileURLWithPath: targetPath))
  return suiDocumentActionSuccess
}

@MainActor
private func suiConfirmUnsavedChanges(_ sceneID: String, _ scene: SUIScenePlanScene, intent: String)
  -> Int32
{
  guard suiDocumentIsDirty(scene) else { return suiDocumentActionSuccess }

  let alert = NSAlert()
  alert.alertStyle = .warning
  alert.messageText =
    "Do you want to save changes to \(suiSuggestedDocumentName(scene, suiSceneDocumentState(scene)))?"
  alert.informativeText = "Your changes will be lost if you \(intent) without saving."

  if (scene.documentSaveCallbackID ?? 0) != 0 {
    alert.addButton(withTitle: "Save")
    alert.addButton(withTitle: "Don't Save")
    alert.addButton(withTitle: "Cancel")
    switch alert.runModal() {
    case .alertFirstButtonReturn:
      return suiPerformSceneDocumentSave(sceneID, scene, forcePanel: false)
    case .alertSecondButtonReturn:
      return suiDocumentActionSuccess
    default:
      return suiDocumentActionCanceled
    }
  }

  alert.addButton(withTitle: "Close Without Saving")
  alert.addButton(withTitle: "Cancel")
  switch alert.runModal() {
  case .alertFirstButtonReturn:
    return suiDocumentActionSuccess
  default:
    return suiDocumentActionCanceled
  }
}

@MainActor
private func suiPrepareSceneWindowClose(_ sceneID: String, _ scene: SUIScenePlanScene, intent: String)
  -> Int32
{
  let unsaved = suiConfirmUnsavedChanges(sceneID, scene, intent: intent)
  guard unsaved == suiDocumentActionSuccess else { return unsaved }
  guard let callbackID = scene.documentCloseCallbackID, callbackID != 0 else {
    return suiDocumentActionSuccess
  }
  guard let fn = _SUICommandCallback, fn(UInt(callbackID)) != 0 else {
    suiPresentDocumentFailureAlert(scene, "close the document")
    return suiDocumentActionFailed
  }
  return suiDocumentActionSuccess
}

@MainActor
private func suiPerformSceneDocumentOperation(_ sceneID: String, _ operation: String) -> Int32 {
  guard let scene = _sceneWindowSpecs[sceneID] else { return suiDocumentActionFailed }
  switch operation {
  case "open":
    let unsaved = suiConfirmUnsavedChanges(sceneID, scene, intent: "open another document")
    guard unsaved == suiDocumentActionSuccess else { return unsaved }
    guard (scene.documentOpenCallbackID ?? 0) != 0 else {
      return suiDocumentActionFailed
    }
    let panel = NSOpenPanel()
    let state = suiSceneDocumentState(scene)
    if !state.path.isEmpty {
      panel.directoryURL = URL(fileURLWithPath: state.path).deletingLastPathComponent()
    }
    panel.canChooseFiles = true
    panel.canChooseDirectories = false
    panel.allowsMultipleSelection = false
    panel.title = "Open Document"
    guard panel.runModal() == .OK, let url = panel.url else {
      return suiDocumentActionCanceled
    }
    return suiApplySceneDocumentOpen(
      sceneID, scene, url.path, failureOperation: "open the document")
  case "save":
    return suiPerformSceneDocumentSave(sceneID, scene, forcePanel: false)
  case "saveAs":
    return suiPerformSceneDocumentSave(sceneID, scene, forcePanel: true)
  case "revert":
    let state = suiSceneDocumentState(scene)
    guard !state.path.isEmpty else { return suiDocumentActionFailed }
    let unsaved = suiConfirmUnsavedChanges(sceneID, scene, intent: "revert your changes")
    guard unsaved == suiDocumentActionSuccess else { return unsaved }
    return suiApplySceneDocumentOpen(
      sceneID, scene, state.path, failureOperation: "revert the document")
  case "export":
    guard let callbackID = scene.documentExportCallbackID, callbackID != 0 else {
      return suiDocumentActionFailed
    }
    let state = suiSceneDocumentState(scene)
    guard let url = suiRunSavePanel(scene, state, operation: "Export Document") else {
      return suiDocumentActionCanceled
    }
    guard suiInvokeDocumentPathCallback(callbackID, url.path) else {
      suiPresentDocumentFailureAlert(scene, "export the document")
      return suiDocumentActionFailed
    }
    return suiDocumentActionSuccess
  case "import":
    guard let callbackID = scene.documentImportCallbackID, callbackID != 0 else {
      return suiDocumentActionFailed
    }
    let panel = NSOpenPanel()
    let state = suiSceneDocumentState(scene)
    if !state.path.isEmpty {
      panel.directoryURL = URL(fileURLWithPath: state.path).deletingLastPathComponent()
    }
    panel.canChooseFiles = true
    panel.canChooseDirectories = false
    panel.allowsMultipleSelection = false
    panel.title = "Import Into Document"
    guard panel.runModal() == .OK, let url = panel.url else {
      return suiDocumentActionCanceled
    }
    guard suiInvokeDocumentPathCallback(callbackID, url.path) else {
      suiPresentDocumentFailureAlert(scene, "import into the document")
      return suiDocumentActionFailed
    }
    return suiDocumentActionSuccess
  case "close":
    guard let window = _sceneWindows[sceneID] else { return suiDocumentActionFailed }
    let result = suiPrepareSceneWindowClose(sceneID, scene, intent: "close this window")
    guard result == suiDocumentActionSuccess else { return result }
    _sceneApprovedCloseIDs.insert(sceneID)
    window.performClose(nil)
    return suiDocumentActionSuccess
  default:
    return suiDocumentActionFailed
  }
}

@MainActor
private func SUIConfirmCloseSceneWindow(
  _ sceneID: String, _ instanceID: String, _ window: NSWindow
) -> Bool {
  if _sceneApprovedCloseIDs.remove(instanceID) != nil {
    return true
  }
  guard let scene = _sceneWindowSpecs[sceneID], scene.kind == "document" else {
    return true
  }
  let result = suiPrepareSceneWindowClose(sceneID, scene, intent: "close this window")
  return result == suiDocumentActionSuccess
}

@MainActor
private func SUIConfirmTerminateDocumentWindows() -> Bool {
  var approved: [String] = []
  let documentScenes =
    _sceneWindowSpecs
    .sorted { $0.key < $1.key }
    .filter { $0.value.kind == "document" }
  for (sceneID, scene) in documentScenes {
    guard _sceneWindows[sceneID] != nil else { continue }
    let result = suiPrepareSceneWindowClose(sceneID, scene, intent: "quit")
    if result != suiDocumentActionSuccess {
      for approvedSceneID in approved {
        _sceneApprovedCloseIDs.remove(approvedSceneID)
      }
      return false
    }
    _sceneApprovedCloseIDs.insert(sceneID)
    approved.append(sceneID)
  }
  return true
}

// MARK: - Scene helpers

@MainActor
private func suiSceneShouldRestoreVisibility(_ scene: SUIScenePlanScene) -> Bool {
  if let restoreVisibility = scene.restoreVisibility {
    return restoreVisibility
  }
  return scene.kind != "settings"
}

@MainActor
private func suiSceneVisibleKey(_ scene: SUIScenePlanScene) -> String {
  "swiftui.scene.visible." + suiScenePersistenceKey(scene)
}

@MainActor
private func suiSceneShouldOpenOnLaunch(_ scene: SUIScenePlanScene) -> Bool {
  let defaultOpenOnLaunch = scene.kind == "settings" ? false : true
  if scene.openOnLaunch ?? defaultOpenOnLaunch {
    return true
  }
  if !suiSceneShouldRestoreVisibility(scene) {
    return scene.openOnLaunch ?? defaultOpenOnLaunch
  }
  let key = suiSceneVisibleKey(scene)
  if UserDefaults.standard.object(forKey: key) != nil {
    return UserDefaults.standard.bool(forKey: key)
  }
  return scene.openOnLaunch ?? defaultOpenOnLaunch
}

@MainActor
private func suiSceneWindowPersistenceKey(_ scene: SUIScenePlanScene, _ instanceID: String) -> String {
  if suiSceneWindowAllowsMultipleInstances(scene) {
    return instanceID
  }
  return suiScenePersistenceKey(scene)
}

// MARK: - Window management

@MainActor
private func SUIRevealSceneWindow(_ sceneID: String) -> Bool {
  guard let scene = _sceneWindowSpecs[sceneID], let view = _sceneWindowViews[sceneID] else {
    return false
  }
  _ = SUIInstallSceneWindow(scene, view, reveal: true)
  return true
}

@MainActor
private func SUIInstallSceneWindow(
  _ scene: SUIScenePlanScene, _ view: AnyView, reveal: Bool, instanceIDOverride: String? = nil
)
  -> String?
{
  guard let id = scene.id else { return nil }
  let isSettings = scene.kind == "settings"
  let width = scene.width ?? 720
  let height = scene.height ?? 480
  let allowsMultipleInstances = suiSceneWindowAllowsMultipleInstances(scene)
  let instanceID = instanceIDOverride ?? (allowsMultipleInstances ? suiSceneWindowNextInstanceID(id) : id)
  let persistenceKey = suiSceneWindowPersistenceKey(scene, instanceID)
  let title: String
  if scene.kind == "document" {
    title = suiSuggestedDocumentName(scene, suiSceneDocumentState(scene))
  } else {
    title = scene.title ?? id
  }
  let bridged = AnyView(SUISceneActionBridgeView(callbackID: scene.actionCallbackID, content: view))
  let sized = AnyView(bridged.frame(minWidth: width, minHeight: height))

  if !allowsMultipleInstances, let window = _sceneWindows[id] {
    if let hc = window.contentViewController as? NSHostingController<AnyView> {
      hc.rootView = sized
    } else {
      let hc = NSHostingController(rootView: sized)
      hc.sizingOptions = []
      window.contentViewController = hc
    }
    window.title = title
    window.setContentSize(NSSize(width: width, height: height))
    if reveal {
      let visibleKey = suiSceneWindowVisibleStoreKey(scene, instanceID)
      if suiSceneShouldRestoreVisibility(scene) {
        UserDefaults.standard.set(true, forKey: visibleKey)
      } else {
        UserDefaults.standard.removeObject(forKey: visibleKey)
      }
      _sceneFocusedWindowID = id
      window.makeKeyAndOrderFront(nil)
      NSApp.activate()
    } else {
      let visibleKey = suiSceneWindowVisibleStoreKey(scene, instanceID)
      if suiSceneShouldRestoreVisibility(scene) {
        UserDefaults.standard.set(false, forKey: visibleKey)
      } else {
        UserDefaults.standard.removeObject(forKey: visibleKey)
      }
      if _sceneFocusedWindowID == id {
        _sceneFocusedWindowID = nil
      }
      window.orderOut(nil)
    }
    return id
  }

  let hc = NSHostingController(rootView: sized)
  hc.sizingOptions = []
  let window = NSWindow(
    contentRect: NSRect(x: 0, y: 0, width: width, height: height),
    styleMask: isSettings
      ? [.titled, .closable, .miniaturizable] : [.titled, .closable, .resizable, .miniaturizable],
    backing: .buffered, defer: false
  )
  window.identifier = NSUserInterfaceItemIdentifier(instanceID)
  window.isReleasedWhenClosed = false
  window.title = title
  window.contentViewController = hc
  window.setContentSize(NSSize(width: width, height: height))
  window.minSize = NSSize(width: 300, height: 200)
  let delegate = SUISceneWindowDelegate(sceneID: id, instanceID: instanceID)
  _sceneWindowDelegates[instanceID] = delegate
  window.delegate = delegate
  if !window.setFrameUsingName(persistenceKey) {
    window.center()
  }
  _sceneWindows[instanceID] = window
  suiSceneRegisterWindowInstance(id, instanceID)
  let availableEvent = suiSceneWindowActionEvent(
    familyID: id, kind: "available", instanceID: instanceID, visible: reveal)
  SUIInvokeSceneActionCallback(scene.actionCallbackID, availableEvent)
  if reveal {
    let visibleKey = suiSceneWindowVisibleStoreKey(scene, instanceID)
    if suiSceneShouldRestoreVisibility(scene) {
      UserDefaults.standard.set(true, forKey: visibleKey)
    } else {
      UserDefaults.standard.removeObject(forKey: visibleKey)
    }
    _sceneFocusedWindowID = instanceID
    window.makeKeyAndOrderFront(nil)
    NSApp.activate()
  } else {
    let visibleKey = suiSceneWindowVisibleStoreKey(scene, instanceID)
    if suiSceneShouldRestoreVisibility(scene) {
      UserDefaults.standard.set(false, forKey: visibleKey)
    } else {
      UserDefaults.standard.removeObject(forKey: visibleKey)
    }
    if _sceneFocusedWindowID == instanceID {
      _sceneFocusedWindowID = nil
    }
    window.orderOut(nil)
  }
  return instanceID
}

// MARK: - Menu bar scene

@MainActor
private func SUIConfigureSceneMenuBar(
  _ scene: SUIScenePlanScene, _ view: AnyView, _ delegate: SUISceneRunnerDelegate
) {
  let labelStr = scene.label ?? "App"
  let imageStr = scene.systemImage ?? "square.grid.2x2"
  let width = scene.width ?? 320
  let height = scene.height ?? 220

  let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
  if let img = NSImage(systemSymbolName: imageStr, accessibilityDescription: labelStr) {
    item.button?.image = img
  }
  item.button?.title = " " + labelStr
  item.button?.target = delegate
  item.button?.action = #selector(SUISceneRunnerDelegate.togglePopover(_:))
  _statusItem = item

  let popover = NSPopover()
  popover.contentSize = NSSize(width: width, height: height)
  popover.behavior = .transient
  let hc = NSHostingController(rootView: AnyView(view.frame(minWidth: width, minHeight: height)))
  popover.contentViewController = hc
  _popover = popover

  if scene.openOnLaunch ?? false {
    DispatchQueue.main.async {
      delegate.openPopoverOnLaunch()
    }
  }
}

// MARK: - Quit menu

// Compatibility wrapper for menu-bar-only surfaces that only need the app menu.
@MainActor
func SUIInstallQuitMenu() {
  SUIInstallDefaultMenus(
    includeSettings: false,
    settingsTarget: nil,
    settingsAction: nil,
    includeWindowMenu: false
  )
}

// MARK: - App menu (with optional Settings)

@MainActor
private func SUIInstallAppMenu(_ delegate: SUISceneRunnerDelegate, includeSettings: Bool, includeWindowMenu: Bool) {
  SUIInstallDefaultMenus(
    includeSettings: includeSettings,
    settingsTarget: delegate,
    settingsAction: #selector(SUISceneRunnerDelegate.openSettings(_:)),
    includeWindowMenu: includeWindowMenu
  )
}

// MARK: - Entry points

@_cdecl("SUIRunScenePlan")
@MainActor
public func SUIRunScenePlan(
  _ planJSON: UnsafePointer<CChar>,
  _ viewRefs: UnsafePointer<UnsafeMutableRawPointer>?,
  _ viewCount: Int32
) {
  suiOnMainSync {
    let data = Data(String(cString: planJSON).utf8)
    guard let plan = try? JSONDecoder().decode(SUIScenePlanPayload.self, from: data) else {
      return
    }
    let refs = viewRefs.map { Array(UnsafeBufferPointer(start: $0, count: Int(viewCount))) } ?? []
    let views = refs.map { ref in
      Unmanaged<Box<AnyView>>.fromOpaque(ref).takeUnretainedValue().value
    }

    _sceneWindows = [:]
    _sceneWindowSpecs = [:]
    _sceneWindowViews = [:]
    _sceneWindowFamiliesByInstanceID = [:]
    _sceneWindowInstanceIDsByFamilyID = [:]
    _sceneWindowNextInstanceIndexByFamilyID = [:]
    _sceneDocumentStates = [:]
    _sceneSettingsID = nil
    _sceneFocusedWindowID = nil
    _sceneWindowDelegates = [:]
    _sceneApprovedCloseIDs = []

    let app = NSApplication.shared
    let hasWindow = plan.scenes.contains {
      $0.kind == "window" || $0.kind == "document" || $0.kind == "settings"
    }
    let hasMenuBar = plan.scenes.contains { $0.kind == "menuBar" }
    let hasSettings = plan.scenes.contains { $0.kind == "settings" }
    app.setActivationPolicy(hasWindow ? .regular : .accessory)

    let delegate = SUISceneRunnerDelegate()
    delegate.terminateAfterLastWindowClosedValue = !hasMenuBar
    if let lc = plan.lifecycle {
      delegate.didFinishLaunchingCallbackID = lc.didFinishLaunchingCallbackID ?? 0
      delegate.didBecomeActiveCallbackID = lc.didBecomeActiveCallbackID ?? 0
      delegate.didResignActiveCallbackID = lc.didResignActiveCallbackID ?? 0
      delegate.shouldTerminateCallbackID = lc.shouldTerminateCallbackID ?? 0
      delegate.willTerminateCallbackID = lc.willTerminateCallbackID ?? 0
    }
    _sceneRunnerDelegate = delegate
    app.delegate = delegate
    _sceneSettingsID = plan.scenes.first(where: { $0.kind == "settings" })?.id
    let commands = plan.commands ?? []
    if commands.isEmpty {
      if hasWindow {
        SUIInstallAppMenu(delegate, includeSettings: hasSettings, includeWindowMenu: hasWindow)
      }
    } else {
      SUIInstallCommandMenus(delegate, includeSettings: hasSettings, commands: commands, includeWindowMenu: hasWindow)
    }

    for scene in plan.scenes {
      guard scene.viewIndex >= 0 && scene.viewIndex < views.count else { continue }
      let view = views[scene.viewIndex]
      switch scene.kind {
      case "window", "document", "settings":
        if let id = scene.id {
          _sceneWindowSpecs[id] = scene
          _sceneWindowViews[id] = view
          if scene.kind == "document" {
            let fallback = SUISceneDocumentState(
              displayName: scene.documentDisplayName ?? scene.title ?? "Untitled",
              path: scene.documentPath ?? "",
              dirty: scene.documentDirty ?? false
            )
            let restored = suiRestoreSceneDocumentState(scene)
            let state = restored ?? fallback
            _sceneDocumentStates[id] = state
            if !state.path.isEmpty {
              suiPersistSceneDocumentState(scene, state)
              suiNoteRecentDocumentURL(URL(fileURLWithPath: state.path))
            }
          }
          if scene.kind == "window", suiSceneWindowAllowsMultipleInstances(scene) {
            let restoredIDs = suiSceneRestoreWindowInstanceIDs(scene)
            if !restoredIDs.isEmpty {
              for instanceID in restoredIDs {
                let reveal = suiSceneWindowInstanceVisibility(scene, instanceID)
                _ = SUIInstallSceneWindow(
                  scene, view, reveal: reveal, instanceIDOverride: instanceID)
              }
              continue
            }
          }
        }
        if suiSceneShouldOpenOnLaunch(scene) {
          _ = SUIInstallSceneWindow(scene, view, reveal: true)
        }
      case "menuBar":
        SUIConfigureSceneMenuBar(scene, view, delegate)
      default:
        continue
      }
    }

    app.activate()
    app.run()
  }
}

@_cdecl("SUIOpenSceneWindow")
@MainActor
public func SUIOpenSceneWindow(_ id: UnsafePointer<CChar>) -> Int32 {
  suiOnMainSync {
    let sceneID = String(cString: id)
    return SUIRevealSceneWindow(sceneID) ? 1 : 0
  }
}

@_cdecl("SUIRunSceneDocumentOperation")
@MainActor
public func SUIRunSceneDocumentOperation(
  _ id: UnsafePointer<CChar>,
  _ operation: UnsafePointer<CChar>
) -> Int32 {
  suiOnMainSync {
    let sceneID = String(cString: id)
    let operationName = String(cString: operation)
    return suiPerformSceneDocumentOperation(sceneID, operationName)
  }
}

@_cdecl("SUIRunSceneDocumentPathOperation")
@MainActor
public func SUIRunSceneDocumentPathOperation(
  _ id: UnsafePointer<CChar>,
  _ operation: UnsafePointer<CChar>,
  _ path: UnsafePointer<CChar>
) -> Int32 {
  suiOnMainSync {
    let sceneID = String(cString: id)
    let operationName = String(cString: operation)
    let pathValue = String(cString: path).trimmingCharacters(in: .whitespacesAndNewlines)
    guard let scene = _sceneWindowSpecs[sceneID] else { return suiDocumentActionFailed }
    switch operationName {
    case "openPath":
      let unsaved = suiConfirmUnsavedChanges(sceneID, scene, intent: "open another document")
      guard unsaved == suiDocumentActionSuccess else { return unsaved }
      return suiApplySceneDocumentOpen(
        sceneID, scene, pathValue, failureOperation: "open the document")
    default:
      return suiDocumentActionFailed
    }
  }
}

@_cdecl("SUIUpdateSceneDocumentState")
@MainActor
public func SUIUpdateSceneDocumentState(
  _ id: UnsafePointer<CChar>,
  _ displayName: UnsafePointer<CChar>,
  _ path: UnsafePointer<CChar>,
  _ dirty: Int32
) {
  suiOnMainSync {
    let sceneID = String(cString: id)
    guard let scene = _sceneWindowSpecs[sceneID] else { return }
    let displayNameValue = String(cString: displayName)
    let pathValue = String(cString: path)
    let current = suiSceneDocumentState(scene)
    let nextDisplayName: String
    if !displayNameValue.isEmpty {
      nextDisplayName = displayNameValue
    } else if !pathValue.isEmpty {
      nextDisplayName = URL(fileURLWithPath: pathValue).lastPathComponent
    } else {
      nextDisplayName = current.displayName
    }
    let updated = SUISceneDocumentState(
      displayName: nextDisplayName,
      path: pathValue,
      dirty: dirty != 0
    )
    suiSetSceneDocumentState(sceneID, scene, updated)
    suiPersistSceneDocumentState(scene, updated)
  }
}

import SwiftUI
import AppKit

// Command menu building, menu item management, and callback registration.

// MARK: - Callback registration

nonisolated(unsafe) var _SUIStringCallback: (@convention(c) (UInt, UnsafePointer<CChar>) -> Int32)?
nonisolated(unsafe) var _SUICommandCallback: (@convention(c) (UInt) -> Int32)?

@_cdecl("SUISetStringCallback")
public func SUISetStringCallback(_ fn: @convention(c) (UInt, UnsafePointer<CChar>) -> Int32) {
    _SUIStringCallback = fn
}

@_cdecl("SUIRegisterCommandCallback")
public func SUIRegisterCommandCallback(_ fn: @convention(c) (UInt) -> Int32) {
    _SUICommandCallback = fn
}

// MARK: - Command coordinator

nonisolated(unsafe) var _commandCoordinator: SUICommandCoordinator?

class SUICommandCoordinator: NSObject, NSMenuDelegate {
    private var actionCallbacks: [Int: UInt64] = [:]
    private var enabledCallbacks: [Int: UInt64] = [:]

    func registerActionItem(_ item: NSMenuItem, actionCallbackID: UInt64, enabledCallbackID: UInt64) {
        let tag = Int(actionCallbackID)
        item.tag = tag
        item.target = self
        item.action = #selector(handleAction(_:))
        actionCallbacks[tag] = actionCallbackID
        registerEnabledItem(item, enabledCallbackID: enabledCallbackID)
    }

    func registerEnabledItem(_ item: NSMenuItem, enabledCallbackID: UInt64) {
        guard enabledCallbackID != 0 else { return }
        if item.tag == 0 {
            item.tag = Int(enabledCallbackID)
        }
        enabledCallbacks[item.tag] = enabledCallbackID
    }

    @objc func handleAction(_ sender: NSMenuItem) {
        guard let cbID = actionCallbacks[sender.tag], cbID != 0 else { return }
        _ = _SUICommandCallback?(UInt(cbID))
    }

    @objc func validateMenuItem(_ item: NSMenuItem) -> Bool {
        guard let cbID = enabledCallbacks[item.tag], cbID != 0, let fn = _SUICommandCallback else {
            return true
        }
        return fn(UInt(cbID)) != 0
    }

    func menuNeedsUpdate(_ menu: NSMenu) {
        for item in menu.items {
            if let cbID = enabledCallbacks[item.tag], cbID != 0, let fn = _SUICommandCallback {
                item.isEnabled = fn(UInt(cbID)) != 0
            }
        }
    }

    @MainActor @objc func closeKeyWindow(_ sender: AnyObject?) {
        suiCurrentCommandWindow()?.performClose(sender)
    }

    @MainActor @objc func minimizeKeyWindow(_ sender: AnyObject?) {
        suiCurrentCommandWindow()?.performMiniaturize(sender)
    }

    @MainActor @objc func zoomKeyWindow(_ sender: AnyObject?) {
        suiCurrentCommandWindow()?.performZoom(sender)
    }

    @MainActor @objc func bringAllToFront(_ sender: AnyObject?) {
        NSApp.arrangeInFront(sender)
    }
}

@MainActor
private func suiCurrentCommandWindow() -> NSWindow? {
    if let sceneID = _sceneFocusedWindowID, let focused = _sceneWindows[sceneID] {
        return focused
    }
    if let key = NSApp.keyWindow {
        return key
    }
    if let main = NSApp.mainWindow {
        return main
    }
    if let visible = _sceneWindows.values.first(where: { $0.isVisible }) {
        return visible
    }
    return NSApp.windows.first(where: { $0.isVisible })
}

@MainActor
private func suiTopLevelMenu(title: String, submenu: NSMenu) -> NSMenuItem {
    let item = NSMenuItem()
    submenu.title = title
    item.submenu = submenu
    return item
}

@MainActor
private func suiBuildAppMenuItem(appName: String, includeSettings: Bool, settingsTarget: AnyObject?, settingsAction: Selector?) -> NSMenuItem {
    let appMenu = NSMenu(title: appName)
    if includeSettings, let settingsAction {
        let settingsItem = NSMenuItem(title: "Settings\u{2026}", action: settingsAction, keyEquivalent: ",")
        settingsItem.target = settingsTarget
        appMenu.addItem(settingsItem)
        appMenu.addItem(.separator())
    }
    let quitItem = NSMenuItem(title: "Quit " + appName, action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
    appMenu.addItem(quitItem)
    return suiTopLevelMenu(title: appName, submenu: appMenu)
}

@MainActor
private func suiBuildDefaultEditMenuItem() -> NSMenuItem {
    let editMenu = NSMenu(title: "Edit")

    let undoItem = NSMenuItem(title: "Undo", action: #selector(UndoManager.undo), keyEquivalent: "z")
    undoItem.target = nil
    editMenu.addItem(undoItem)

    let redoItem = NSMenuItem(title: "Redo", action: #selector(UndoManager.redo), keyEquivalent: "z")
    redoItem.target = nil
    redoItem.keyEquivalentModifierMask = [.command, .shift]
    editMenu.addItem(redoItem)

    editMenu.addItem(.separator())

    let cutItem = NSMenuItem(title: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x")
    cutItem.target = nil
    editMenu.addItem(cutItem)

    let copyItem = NSMenuItem(title: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
    copyItem.target = nil
    editMenu.addItem(copyItem)

    let pasteItem = NSMenuItem(title: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
    pasteItem.target = nil
    editMenu.addItem(pasteItem)

    let selectAllItem = NSMenuItem(title: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
    selectAllItem.target = nil
    editMenu.addItem(selectAllItem)

    return suiTopLevelMenu(title: "Edit", submenu: editMenu)
}

@MainActor
private func suiBuildDefaultWindowMenuItem(coordinator: SUICommandCoordinator?) -> NSMenuItem {
    let windowMenu = NSMenu(title: "Window")

    let minimizeItem = NSMenuItem(title: "Minimize", action: nil, keyEquivalent: "m")
    if let coordinator {
        minimizeItem.target = coordinator
        minimizeItem.action = #selector(SUICommandCoordinator.minimizeKeyWindow(_:))
    } else {
        minimizeItem.target = nil
        minimizeItem.action = #selector(NSWindow.performMiniaturize(_:))
    }
    windowMenu.addItem(minimizeItem)

    let zoomItem = NSMenuItem(title: "Zoom", action: nil, keyEquivalent: "")
    if let coordinator {
        zoomItem.target = coordinator
        zoomItem.action = #selector(SUICommandCoordinator.zoomKeyWindow(_:))
    } else {
        zoomItem.target = nil
        zoomItem.action = #selector(NSWindow.performZoom(_:))
    }
    windowMenu.addItem(zoomItem)

    windowMenu.addItem(.separator())

    let bringAllItem = NSMenuItem(title: "Bring All to Front", action: nil, keyEquivalent: "")
    if let coordinator {
        bringAllItem.target = coordinator
        bringAllItem.action = #selector(SUICommandCoordinator.bringAllToFront(_:))
    } else {
        bringAllItem.target = nil
        bringAllItem.action = #selector(NSApplication.arrangeInFront(_:))
    }
    windowMenu.addItem(bringAllItem)

    NSApp.windowsMenu = windowMenu
    return suiTopLevelMenu(title: "Window", submenu: windowMenu)
}

private func suiHasTopLevelMenu(_ mainMenu: NSMenu, title: String) -> Bool {
    mainMenu.items.contains {
        $0.submenu?.title.compare(title, options: [.caseInsensitive, .diacriticInsensitive]) == .orderedSame
    }
}

@MainActor
private func suiAppendDefaultSystemMenus(_ mainMenu: NSMenu, coordinator: SUICommandCoordinator?, includeWindowMenu: Bool) {
    if !suiHasTopLevelMenu(mainMenu, title: "Edit") {
        mainMenu.addItem(suiBuildDefaultEditMenuItem())
    }
    if includeWindowMenu && !suiHasTopLevelMenu(mainMenu, title: "Window") {
        mainMenu.addItem(suiBuildDefaultWindowMenuItem(coordinator: coordinator))
    }
}

@MainActor
func SUIInstallDefaultMenus(includeSettings: Bool, settingsTarget: AnyObject?, settingsAction: Selector?, includeWindowMenu: Bool) {
    let appName = ProcessInfo.processInfo.processName
    let mainMenu = NSMenu(title: appName)
    mainMenu.addItem(suiBuildAppMenuItem(
        appName: appName,
        includeSettings: includeSettings,
        settingsTarget: settingsTarget,
        settingsAction: settingsAction,
    ))
    suiAppendDefaultSystemMenus(mainMenu, coordinator: nil, includeWindowMenu: includeWindowMenu)
    NSApp.mainMenu = mainMenu
}

// MARK: - System action helpers

@MainActor
private func suiSystemSelector(_ action: String) -> Selector? {
    switch action {
    case "undo":
        return #selector(UndoManager.undo)
    case "redo":
        return #selector(UndoManager.redo)
    case "cut":
        return #selector(NSText.cut(_:))
    case "copy":
        return #selector(NSText.copy(_:))
    case "paste":
        return #selector(NSText.paste(_:))
    case "selectAll":
        return #selector(NSText.selectAll(_:))
    case "closeWindow":
        return #selector(NSWindow.performClose(_:))
    case "minimizeWindow":
        return #selector(NSWindow.performMiniaturize(_:))
    case "zoomWindow":
        return #selector(NSWindow.performZoom(_:))
    case "bringAllToFront":
        return #selector(NSApplication.arrangeInFront(_:))
    default:
        return nil
    }
}

@MainActor
private func suiBindSystemAction(_ item: NSMenuItem, action: String, coordinator: SUICommandCoordinator) {
    switch action {
    case "closeWindow":
        item.target = coordinator
        item.action = #selector(SUICommandCoordinator.closeKeyWindow(_:))
    case "minimizeWindow":
        item.target = coordinator
        item.action = #selector(SUICommandCoordinator.minimizeKeyWindow(_:))
    case "zoomWindow":
        item.target = coordinator
        item.action = #selector(SUICommandCoordinator.zoomKeyWindow(_:))
    case "bringAllToFront":
        item.target = coordinator
        item.action = #selector(SUICommandCoordinator.bringAllToFront(_:))
    default:
        guard let selector = suiSystemSelector(action) else { return }
        item.action = selector
        item.target = nil
    }
}

// MARK: - Menu building

@MainActor
private func SUIBuildMenuItems(_ items: [SUICommandItem], coordinator: SUICommandCoordinator) -> [NSMenuItem] {
    var result: [NSMenuItem] = []
    for item in items {
        if item.kind == "separator" || (item.title ?? "").isEmpty {
            result.append(.separator())
            continue
        }
        let title = item.title ?? ""
        let menuItem = NSMenuItem(title: title, action: nil, keyEquivalent: "")

        if let children = item.children, !children.isEmpty {
            let submenu = NSMenu(title: title)
            submenu.delegate = coordinator
            for child in SUIBuildMenuItems(children, coordinator: coordinator) {
                submenu.addItem(child)
            }
            menuItem.submenu = submenu
        } else if let actionID = item.actionCallbackID, actionID != 0 {
            coordinator.registerActionItem(
                menuItem,
                actionCallbackID: actionID,
                enabledCallbackID: item.enabledCallbackID ?? 0
            )
        } else if let systemAction = item.systemAction, !systemAction.isEmpty {
            suiBindSystemAction(menuItem, action: systemAction, coordinator: coordinator)
            coordinator.registerEnabledItem(menuItem, enabledCallbackID: item.enabledCallbackID ?? 0)
        } else if let enabledCallbackID = item.enabledCallbackID, enabledCallbackID != 0 {
            coordinator.registerEnabledItem(menuItem, enabledCallbackID: enabledCallbackID)
        }

        if let key = item.shortcutKey, !key.isEmpty {
            menuItem.keyEquivalent = key
            let rawMods = item.shortcutModifiers ?? 0
            if rawMods != 0 {
                menuItem.keyEquivalentModifierMask = NSEvent.ModifierFlags(rawValue: UInt(rawMods))
            } else {
                menuItem.keyEquivalentModifierMask = .command
            }
        }

        result.append(menuItem)
    }
    return result
}

@MainActor
func SUIInstallCommandMenus(_ delegate: SUISceneRunnerDelegate, includeSettings: Bool, commands: [SUICommandGroup], includeWindowMenu: Bool) {
    let coordinator = SUICommandCoordinator()
    _commandCoordinator = coordinator

    let appName = ProcessInfo.processInfo.processName
    let mainMenu = NSMenu(title: appName)

    mainMenu.addItem(suiBuildAppMenuItem(
        appName: appName,
        includeSettings: includeSettings,
        settingsTarget: delegate,
        settingsAction: #selector(SUISceneRunnerDelegate.openSettings(_:)),
    ))

    // Command menus from the scene plan.
    for group in commands {
        let headerItem = NSMenuItem()
        let submenu = NSMenu(title: group.title)
        submenu.delegate = coordinator
        for child in SUIBuildMenuItems(group.items, coordinator: coordinator) {
            submenu.addItem(child)
        }
        headerItem.submenu = submenu
        mainMenu.addItem(headerItem)
    }

    suiAppendDefaultSystemMenus(mainMenu, coordinator: coordinator, includeWindowMenu: includeWindowMenu)
    NSApp.mainMenu = mainMenu
    SUIInvalidateCommandMenus()
}

// MARK: - Menu item update

@_cdecl("SUIUpdateMenuItemEnabled")
@MainActor
public func SUIUpdateMenuItemEnabled(_ tag: Int32, _ enabled: Int32) {
    guard let menu = NSApp.mainMenu else { return }
    guard let item = menu.item(withTag: Int(tag)) ?? suiFindMenuItem(in: menu, tag: Int(tag)) else { return }
    item.isEnabled = enabled != 0
}

@MainActor
private func suiFindMenuItem(in menu: NSMenu, tag: Int) -> NSMenuItem? {
    for item in menu.items {
        if item.tag == tag { return item }
        if let sub = item.submenu, let found = suiFindMenuItem(in: sub, tag: tag) {
            return found
        }
    }
    return nil
}

@MainActor
func SUIInvalidateCommandMenus() {
    guard let menu = NSApp.mainMenu else { return }
    suiRefreshMenu(menu)
}

@MainActor
private func suiRefreshMenu(_ menu: NSMenu) {
    menu.update()
    for item in menu.items {
        if let submenu = item.submenu {
            suiRefreshMenu(submenu)
        }
    }
}

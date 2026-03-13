import SwiftUI
import AppKit


class SUIAppDelegate: NSObject, NSApplicationDelegate {
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }
}

nonisolated(unsafe) var _appDelegate: SUIAppDelegate?

@_cdecl("SUIRun")
@MainActor
public func SUIRun(_ rootView: UnsafeMutableRawPointer, _ title: UnsafePointer<CChar>,
                    _ width: Double, _ height: Double) {
    let view = Unmanaged<Box<AnyView>>.fromOpaque(rootView).takeUnretainedValue().value
    let t = String(cString: title)

    let app = NSApplication.shared
    app.setActivationPolicy(.regular)

    let delegate = SUIAppDelegate()
    _appDelegate = delegate
    app.delegate = delegate

    let sized = AnyView(
        view.frame(minWidth: width, minHeight: height)
    )

    let hc = NSHostingController(rootView: sized)
    hc.sizingOptions = []

    let window = NSWindow(
        contentRect: NSRect(x: 0, y: 0, width: width, height: height),
        styleMask: [.titled, .closable, .resizable, .miniaturizable],
        backing: .buffered, defer: false
    )
    window.title = t
    window.contentViewController = hc
    window.setContentSize(NSSize(width: width, height: height))
    window.minSize = NSSize(width: 300, height: 200)
    window.center()
    window.makeKeyAndOrderFront(nil)
    app.activate()
    app.run()
}

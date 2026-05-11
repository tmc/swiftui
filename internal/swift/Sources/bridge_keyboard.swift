import AppKit
import SwiftUI

@_cdecl("SUIKeyboardCapture")
@MainActor
public func SUIKeyboardCapture(_ callbackID: UInt) -> UnsafeMutableRawPointer {
    let view = AnyView(KeyboardCaptureRepresentable(callbackID: callbackID))
    return Unmanaged.passRetained(Box(view)).toOpaque()
}

struct KeyboardCaptureRepresentable: NSViewRepresentable {
    let callbackID: UInt

    func makeNSView(context: Context) -> KeyboardCaptureNSView {
        let view = KeyboardCaptureNSView()
        view.callbackID = callbackID
        DispatchQueue.main.async {
            view.window?.makeFirstResponder(view)
        }
        return view
    }

    func updateNSView(_ nsView: KeyboardCaptureNSView, context: Context) {
        nsView.callbackID = callbackID
    }
}

final class KeyboardCaptureNSView: NSView {
    var callbackID: UInt = 0
    private var requestedInitialFocus = false

    override var acceptsFirstResponder: Bool { true }
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool { true }
    override var intrinsicContentSize: NSSize {
        NSSize(width: NSView.noIntrinsicMetric, height: NSView.noIntrinsicMetric)
    }

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()
        if !requestedInitialFocus {
            requestedInitialFocus = true
            DispatchQueue.main.async { [weak self] in
                self?.window?.makeFirstResponder(self)
            }
        }
    }

    override func mouseDown(with event: NSEvent) {
        window?.makeFirstResponder(self)
        super.mouseDown(with: event)
    }

    override func keyDown(with event: NSEvent) {
        guard callbackID != 0 else { return }
        let text = encoded(event: event)
        guard !text.isEmpty else { return }
        text.withCString { ptr in
            _ = _SUIStringCallback?(callbackID, ptr)
        }
    }

    private func encoded(event: NSEvent) -> String {
        switch event.keyCode {
        case 36: return "\r"
        case 48: return "\t"
        case 51: return "\u{7f}"
        case 53: return "\u{1b}"
        case 123: return "\u{1b}[D"
        case 124: return "\u{1b}[C"
        case 125: return "\u{1b}[B"
        case 126: return "\u{1b}[A"
        case 115: return "\u{1b}[H"
        case 119: return "\u{1b}[F"
        case 116: return "\u{1b}[5~"
        case 121: return "\u{1b}[6~"
        default:
            if event.modifierFlags.contains(.control),
               let scalar = event.charactersIgnoringModifiers?.unicodeScalars.first {
                let v = scalar.value
                if v >= 64 && v <= 95 {
                    return String(UnicodeScalar(v - 64)!)
                }
                if v >= 97 && v <= 122 {
                    return String(UnicodeScalar(v - 96)!)
                }
            }
            return event.characters ?? ""
        }
    }
}

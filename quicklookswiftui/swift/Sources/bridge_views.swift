import QuickLook
import SwiftUI


// MARK: - QuickLookPreview

@_cdecl("QLS_QuickLookPreview")
@MainActor
public func QLS_QuickLookPreview(_ view: UnsafeMutableRawPointer, _ urlStr: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let v = Unmanaged<Box<AnyView>>.fromOpaque(view).takeUnretainedValue().value
    let url = URL(fileURLWithPath: String(cString: urlStr))
    let modified = AnyView(v.quickLookPreview(Binding.constant(url)))
    return Unmanaged.passRetained(Box(modified)).toOpaque()
}

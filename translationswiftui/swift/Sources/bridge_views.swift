import Translation
import SwiftUI


// MARK: - TranslationPresentation

@_cdecl("TRS_TranslationPresentation")
@MainActor
public func TRS_TranslationPresentation(_ view: UnsafeMutableRawPointer, _ text: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let v = Unmanaged<Box<AnyView>>.fromOpaque(view).takeUnretainedValue().value
    let t = String(cString: text)
    let modified = AnyView(v.translationPresentation(isPresented: .constant(true), text: t))
    return Unmanaged.passRetained(Box(modified)).toOpaque()
}

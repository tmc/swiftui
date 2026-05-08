import Foundation
import SwiftUI

func suiAnimationForKind(_ kind: Int32, duration: Double) -> Animation {
    let d = duration.isFinite && duration > 0 ? duration : 0
    if d == 0 {
        return animationForKind(kind)
    }
    switch kind {
    case 1:
        return .easeIn(duration: d)
    case 2:
        return .easeOut(duration: d)
    default:
        return .easeInOut(duration: d)
    }
}

private func suiInlineMarkdown(_ source: String) -> AttributedString {
    let trimmed = source.trimmingCharacters(in: .newlines)
    let options = AttributedString.MarkdownParsingOptions(
        interpretedSyntax: .inlineOnlyPreservingWhitespace
    )
    if let attr = try? AttributedString(markdown: trimmed, options: options) {
        return attr
    }
    return AttributedString(trimmed)
}

@_cdecl("SUISelectableText")
public func SUISelectableText(_ textPtr: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let text = String(cString: textPtr)
    let view = AnyView(
        Text(verbatim: text)
            .textSelection(.enabled)
            .fixedSize(horizontal: false, vertical: true)
    )
    return retainView(view)
}

@_cdecl("SUIInlineMarkdownText")
public func SUIInlineMarkdownText(_ textPtr: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let source = String(cString: textPtr)
    let view = AnyView(
        Text(suiInlineMarkdown(source))
            .textSelection(.enabled)
            .fixedSize(horizontal: false, vertical: true)
    )
    return retainView(view)
}

@_cdecl("SUIViewEnvironmentOpenURL")
@MainActor
public func SUIViewEnvironmentOpenURL(
    _ viewRef: UnsafeMutableRawPointer,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    guard callbackID != 0 else {
        return retainDerivedView(from: viewRef, base)
    }
    let id = callbackID
    let view = AnyView(base.environment(\.openURL, OpenURLAction { url in
        url.absoluteString.withCString { ptr in
            _SUIStringCallback?(id, ptr) != 0 ? .handled : .systemAction
        }
    }))
    return retainDerivedView(from: viewRef, view)
}

@_cdecl("SUIViewOnTapGestureCount")
public func SUIViewOnTapGestureCount(
    _ viewRef: UnsafeMutableRawPointer,
    _ count: Int32,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    let taps = max(1, Int(count))
    let id = callbackID
    let view = AnyView(base.onTapGesture(count: taps) {
        _SUIButtonCallback?(id)
    })
    return retainDerivedView(from: viewRef, view)
}

@_cdecl("SUIViewScaleEffectAnchor")
public func SUIViewScaleEffectAnchor(
    _ viewRef: UnsafeMutableRawPointer,
    _ scale: Double,
    _ anchorX: Double,
    _ anchorY: Double
) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    let view = AnyView(base.scaleEffect(scale, anchor: UnitPoint(x: anchorX, y: anchorY)))
    return retainDerivedView(from: viewRef, view)
}

@_cdecl("SUIViewAnimationWithDuration")
public func SUIViewAnimationWithDuration(
    _ viewRef: UnsafeMutableRawPointer,
    _ kind: Int32,
    _ duration: Double
) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    let view = AnyView(base.animation(suiAnimationForKind(kind, duration: duration), value: true))
    return retainDerivedView(from: viewRef, view)
}

@_cdecl("SUIStateSetIntAnimatedWithDuration")
public func SUIStateSetIntAnimatedWithDuration(
    _ ref: UnsafeMutableRawPointer,
    _ value: Int32,
    _ kind: Int32,
    _ duration: Double
) {
    let state = Unmanaged<BridgedIntState>.fromOpaque(ref).takeUnretainedValue()
    let v = Int(value)
    if Thread.isMainThread {
        BridgeCommandQueue.shared.applyInline(kind: kind, duration: duration) {
            state.setAndBump(v)
        }
    } else {
        BridgeCommandQueue.shared.enqueue(kind: kind, duration: duration) {
            state.setAndBump(v)
        }
    }
}

@_cdecl("SUIStateSetFloatAnimatedWithDuration")
public func SUIStateSetFloatAnimatedWithDuration(
    _ ref: UnsafeMutableRawPointer,
    _ value: Double,
    _ kind: Int32,
    _ duration: Double
) {
    let state = Unmanaged<BridgedFloatState>.fromOpaque(ref).takeUnretainedValue()
    if Thread.isMainThread {
        BridgeCommandQueue.shared.applyInline(kind: kind, duration: duration) {
            state.setAndBump(value)
        }
    } else {
        BridgeCommandQueue.shared.enqueue(kind: kind, duration: duration) {
            state.setAndBump(value)
        }
    }
}

@_cdecl("SUIStateSetBoolAnimatedWithDuration")
public func SUIStateSetBoolAnimatedWithDuration(
    _ ref: UnsafeMutableRawPointer,
    _ value: Int32,
    _ kind: Int32,
    _ duration: Double
) {
    let state = Unmanaged<BridgedBoolState>.fromOpaque(ref).takeUnretainedValue()
    let v = value != 0
    if Thread.isMainThread {
        BridgeCommandQueue.shared.applyInline(kind: kind, duration: duration) {
            state.setAndBump(v)
        }
    } else {
        BridgeCommandQueue.shared.enqueue(kind: kind, duration: duration) {
            state.setAndBump(v)
        }
    }
}

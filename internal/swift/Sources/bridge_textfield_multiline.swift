import SwiftUI

@MainActor
private struct BridgedMultilineTextFieldCallbacks: View {
    let placeholder: String
    let state: BridgedStringState
    let minLines: Int
    let maxLines: Int
    let onChangeID: UInt
    let onSubmitID: UInt

    @State private var draft: String
    @State private var suppressDraftSync = false

    init(
        placeholder: String,
        state: BridgedStringState,
        minLines: Int,
        maxLines: Int,
        onChangeID: UInt,
        onSubmitID: UInt
    ) {
        self.placeholder = placeholder
        self.state = state
        self.minLines = minLines
        self.maxLines = maxLines
        self.onChangeID = onChangeID
        self.onSubmitID = onSubmitID
        _draft = State(initialValue: state.value)
    }

    private var lineRange: ClosedRange<Int> {
        let lower = max(1, minLines)
        let upper = max(lower, maxLines)
        return lower...upper
    }

    var body: some View {
        TextField(placeholder, text: $draft, axis: .vertical)
            .lineLimit(lineRange)
            .onChange(of: draft) { _, newValue in
                if suppressDraftSync {
                    suppressDraftSync = false
                    return
                }
                if state.value != newValue {
                    state.setAndBump(newValue)
                }
                if onChangeID != 0 {
                    _SUIButtonCallback?(onChangeID)
                }
            }
            .onChange(of: state.value) { _, newValue in
                if draft == newValue {
                    return
                }
                suppressDraftSync = true
                draft = newValue
            }
            .onSubmit {
                if state.value != draft {
                    state.setAndBump(draft)
                }
                if onSubmitID != 0 {
                    _SUIButtonCallback?(onSubmitID)
                }
            }
    }
}

@_cdecl("SUIMultilineTextFieldCallbacks")
@MainActor
public func SUIMultilineTextFieldCallbacks(
    _ placeholderPtr: UnsafePointer<CChar>,
    _ stateRef: UnsafeMutableRawPointer,
    _ minLines: Int32,
    _ maxLines: Int32,
    _ onChangeID: UInt,
    _ onSubmitID: UInt
) -> UnsafeMutableRawPointer {
    let placeholder = String(cString: placeholderPtr)
    let state = Unmanaged<BridgedStringState>.fromOpaque(stateRef).takeUnretainedValue()
    let view = AnyView(
        BridgedMultilineTextFieldCallbacks(
            placeholder: placeholder,
            state: state,
            minLines: Int(minLines),
            maxLines: Int(maxLines),
            onChangeID: onChangeID,
            onSubmitID: onSubmitID
        )
    )
    return retainView(view)
}

import AppKit
import Foundation
import PhotosUI
import SwiftUI
import UniformTypeIdentifiers

@Observable
final class BridgedStringListState: @unchecked Sendable {
    var value: [String]

    init(_ value: [String]) {
        self.value = value
    }
}

private struct SearchableChoiceOption: Codable, Hashable {
    let label: String
    let value: String
}

private struct BridgedPhotosPickerSelectionItem: Codable, Hashable {
    let id: String
    let filename: String
    let utType: String
    let mediaKind: String
    let order: Int
}

private func suiStringList(from json: String) -> [String] {
    guard let data = json.data(using: .utf8),
          let value = try? JSONDecoder().decode([String].self, from: data) else {
        return []
    }
    return value
}

private func suiStringListJSON(_ value: [String]) -> UnsafeMutablePointer<CChar>? {
    guard let data = try? JSONEncoder().encode(value),
          let string = String(data: data, encoding: .utf8) else {
        return strdup("[]")
    }
    return strdup(string)
}

private func suiChoiceOptions(from json: String) -> [SearchableChoiceOption] {
    guard let data = json.data(using: .utf8),
          let value = try? JSONDecoder().decode([SearchableChoiceOption].self, from: data) else {
        return []
    }
    return value
}

private func suiChoiceOptions(packed ptr: UnsafePointer<UInt8>?, length: Int) -> [SearchableChoiceOption] {
    suiDecodePackedChoiceOptions(ptr, length).map {
        SearchableChoiceOption(label: $0.label, value: $0.value)
    }
}

private func suiPhotosPickerFilename(item: PhotosPickerItem, fallbackID: String, utType: String) -> String {
    if let identifier = item.itemIdentifier, !identifier.isEmpty {
        let value = URL(fileURLWithPath: identifier).lastPathComponent
        if !value.isEmpty {
            return value
        }
        return identifier
    }
    if let ext = UTType(utType)?.preferredFilenameExtension, !ext.isEmpty {
        return fallbackID + "." + ext
    }
    return fallbackID
}

private func suiPhotosPickerSelectionJSON(_ value: [PhotosPickerItem]) -> String {
    let normalized = value.enumerated().map { index, item in
        let utType = item.supportedContentTypes.first?.identifier ?? ""
        let id = item.itemIdentifier ?? "photo-\(index)"
        let mediaKind: String
        if item.supportedContentTypes.first?.conforms(to: .image) == true {
            mediaKind = "image"
        } else if item.supportedContentTypes.first?.conforms(to: .movie) == true ||
                    item.supportedContentTypes.first?.conforms(to: .audiovisualContent) == true {
            mediaKind = "video"
        } else {
            mediaKind = ""
        }
        return BridgedPhotosPickerSelectionItem(
            id: id,
            filename: suiPhotosPickerFilename(item: item, fallbackID: id, utType: utType),
            utType: utType,
            mediaKind: mediaKind,
            order: index
        )
    }
    guard let data = try? JSONEncoder().encode(normalized),
          let string = String(data: data, encoding: .utf8) else {
        return "[]"
    }
    return string
}

private func suiPhotosPickerFilter(_ matching: Int32) -> PHPickerFilter? {
    switch matching {
    case 1:
        return .images
    case 2:
        return .videos
    default:
        return nil
    }
}

private func suiAlignment(_ horizontal: Int32, _ vertical: Int32) -> Alignment {
    let h: HorizontalAlignment
    switch horizontal {
    case 0:
        h = .leading
    case 2:
        h = .trailing
    default:
        h = .center
    }
    let v: VerticalAlignment
    switch vertical {
    case 0:
        v = .top
    case 2:
        v = .bottom
    default:
        v = .center
    }
    switch (h, v) {
    case (.leading, .top):
        return .topLeading
    case (.leading, .bottom):
        return .bottomLeading
    case (.trailing, .top):
        return .topTrailing
    case (.trailing, .bottom):
        return .bottomTrailing
    case (.leading, _):
        return .leading
    case (.trailing, _):
        return .trailing
    case (_, .top):
        return .top
    case (_, .bottom):
        return .bottom
    default:
        return .center
    }
}

@MainActor
enum SUIRegexMatcher {
    private enum Entry {
        case invalid
        case regex(NSRegularExpression)
    }

    private static var cache: [String: Entry] = [:]

    static func matches(_ pattern: String, value: String) -> Bool {
        if pattern.isEmpty {
            return true
        }
        let entry: Entry
        if let cached = cache[pattern] {
            entry = cached
        } else if let regex = try? NSRegularExpression(pattern: pattern) {
            let compiled = Entry.regex(regex)
            cache[pattern] = compiled
            entry = compiled
        } else {
            cache[pattern] = .invalid
            entry = .invalid
        }
        guard case let .regex(regex) = entry else {
            return true
        }
        let range = NSRange(location: 0, length: value.utf16.count)
        guard let match = regex.firstMatch(in: value, options: [], range: range) else {
            return false
        }
        return NSEqualRanges(match.range, range)
    }

    static func resetForTests() {
        cache.removeAll()
    }

    static var cachedPatternCount: Int {
        cache.count
    }
}

@MainActor
private func suiRegexMatches(_ pattern: String, value: String) -> Bool {
    SUIRegexMatcher.matches(pattern, value: value)
}

@MainActor
private func suiSetValidState(_ validState: BridgedBoolState?, value: String, pattern: String) {
    guard let validState else {
        return
    }
    let valid = suiRegexMatches(pattern, value: value)
    if validState.value != valid {
        validState.value = valid
    }
}

@_cdecl("SUIStateCreateStringList")
public func SUIStateCreateStringList(_ jsonPtr: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let value = suiStringList(from: String(cString: jsonPtr))
    return Unmanaged.passRetained(BridgedStringListState(value)).toOpaque()
}

@_cdecl("SUIStateGetStringListJSON")
public func SUIStateGetStringListJSON(_ ref: UnsafeMutableRawPointer) -> UnsafeMutablePointer<CChar>? {
    let state = Unmanaged<BridgedStringListState>.fromOpaque(ref).takeUnretainedValue()
    return suiStringListJSON(state.value)
}

@_cdecl("SUIStateSetStringListJSON")
public func SUIStateSetStringListJSON(_ ref: UnsafeMutableRawPointer, _ jsonPtr: UnsafePointer<CChar>) {
    let state = Unmanaged<BridgedStringListState>.fromOpaque(ref).takeUnretainedValue()
    state.value = suiStringList(from: String(cString: jsonPtr))
}

// Packed-wire variants (P4). These replace the JSON entry points on the hot
// path: Go encodes the slice into a length-prefixed buffer (see
// wire_packed.go) and passes pointer+length rather than a NUL-terminated JSON
// blob. JSON entry points above remain for debug and persistence.
@_cdecl("SUIStateCreateStringListPacked")
public func SUIStateCreateStringListPacked(
    _ ptr: UnsafePointer<UInt8>?,
    _ length: Int32
) -> UnsafeMutableRawPointer {
    let values = suiDecodePackedStringSlice(ptr, Int(length))
    return Unmanaged.passRetained(BridgedStringListState(values)).toOpaque()
}

@_cdecl("SUIStateGetStringListPacked")
public func SUIStateGetStringListPacked(
    _ ref: UnsafeMutableRawPointer,
    _ outLen: UnsafeMutablePointer<Int32>?
) -> UnsafeMutablePointer<UInt8>? {
    let state = Unmanaged<BridgedStringListState>.fromOpaque(ref).takeUnretainedValue()
    let (buf, total) = suiEncodePackedStringSlice(state.value)
    outLen?.pointee = Int32(total)
    return buf
}

@_cdecl("SUIStateSetStringListPacked")
public func SUIStateSetStringListPacked(
    _ ref: UnsafeMutableRawPointer,
    _ ptr: UnsafePointer<UInt8>?,
    _ length: Int32
) {
    let state = Unmanaged<BridgedStringListState>.fromOpaque(ref).takeUnretainedValue()
    state.value = suiDecodePackedStringSlice(ptr, Int(length))
}

@MainActor
private final class RemoteImageModel: ObservableObject {
    @Published var image: NSImage?
    @Published var failed = false

    init(url: URL) {
        load(url: url)
    }

    private func load(url: URL) {
        URLSession.shared.dataTask(with: url) { [weak self] data, _, error in
            let image = data.flatMap(NSImage.init(data:))
            DispatchQueue.main.async {
                guard let self else {
                    return
                }
                self.image = image
                self.failed = error != nil || image == nil
            }
        }.resume()
    }
}

@MainActor
private struct BridgedPhotosPickerView: View {
    let label: String
    let matching: Int32
    let maxSelectionCount: Int32
    let callbackID: UInt

    @State private var selection: [PhotosPickerItem] = []

    var body: some View {
        PhotosPicker(
            selection: $selection,
            maxSelectionCount: maxSelectionCount > 0 ? Int(maxSelectionCount) : nil,
            matching: suiPhotosPickerFilter(matching)
        ) {
            Label(label, systemImage: "photo.on.rectangle")
        }
        .onAppear {
            pushSelection(selection)
        }
        .onChange(of: selection) { _, newValue in
            pushSelection(newValue)
        }
    }

    private func pushSelection(_ items: [PhotosPickerItem]) {
        guard callbackID != 0 else {
            return
        }
        let payload = suiPhotosPickerSelectionJSON(items)
        payload.withCString { ptr in
            _ = _SUIStringCallback?(callbackID, ptr)
        }
    }
}

@MainActor
private struct BridgedOpenPanelButton: View {
    let label: String
    let callbackID: UInt

    var body: some View {
        Button(label) {
            let panel = NSOpenPanel()
            panel.canChooseFiles = true
            panel.canChooseDirectories = false
            panel.allowsMultipleSelection = false
            panel.title = label.isEmpty ? "Open File" : label
            guard panel.runModal() == .OK, let url = panel.url else {
                return
            }
            guard callbackID != 0 else {
                return
            }
            url.path.withCString { ptr in
                _ = _SUIStringCallback?(callbackID, ptr)
            }
        }
    }
}

@_cdecl("SUIPhotosPicker")
@MainActor
public func SUIPhotosPicker(
    _ labelPtr: UnsafePointer<CChar>,
    _ matching: Int32,
    _ maxSelectionCount: Int32,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let label = String(cString: labelPtr)
    let view = AnyView(BridgedPhotosPickerView(
        label: label,
        matching: matching,
        maxSelectionCount: maxSelectionCount,
        callbackID: callbackID
    ))
    return retainView(view)
}

@_cdecl("SUIOpenPanel")
@MainActor
public func SUIOpenPanel(
    _ labelPtr: UnsafePointer<CChar>,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let label = String(cString: labelPtr)
    let view = AnyView(BridgedOpenPanelButton(label: label, callbackID: callbackID))
    return retainView(view)
}

private struct ScaleDownImageView: View {
    let image: NSImage

    var body: some View {
        GeometryReader { proxy in
            let intrinsic = image.size
            if intrinsic.width > 0,
               intrinsic.height > 0,
               (intrinsic.width > proxy.size.width || intrinsic.height > proxy.size.height) {
                Image(nsImage: image)
                    .resizable()
                    .aspectRatio(contentMode: .fit)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
            } else {
                Image(nsImage: image)
                    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
            }
        }
    }
}

@MainActor
private struct BridgedAsyncImageFitView: View {
    let fit: Int32
    @StateObject private var model: RemoteImageModel

    init(url: URL, fit: Int32) {
        self.fit = fit
        _model = StateObject(wrappedValue: RemoteImageModel(url: url))
    }

    @ViewBuilder
    private func fitted(_ image: NSImage) -> some View {
        switch fit {
        case 1:
            Image(nsImage: image)
                .resizable()
                .aspectRatio(contentMode: .fill)
                .clipped()
        case 2:
            Image(nsImage: image)
                .resizable()
        case 3:
            Image(nsImage: image)
        case 4:
            ScaleDownImageView(image: image)
        default:
            Image(nsImage: image)
                .resizable()
                .aspectRatio(contentMode: .fit)
        }
    }

    var body: some View {
        Group {
            if let image = model.image {
                fitted(image)
            } else if model.failed {
                Image(systemName: "exclamationmark.triangle")
            } else {
                ProgressView()
            }
        }
    }
}

@_cdecl("SUIAsyncImageFit")
@MainActor
public func SUIAsyncImageFit(_ urlPtr: UnsafePointer<CChar>, _ fit: Int32) -> UnsafeMutableRawPointer {
    let raw = String(cString: urlPtr)
    guard let url = URL(string: raw) else {
        return retainView(AnyView(Image(systemName: "photo")))
    }
    return retainView(AnyView(BridgedAsyncImageFitView(url: url, fit: fit)))
}

@_cdecl("SUIDatePickerBounded")
@MainActor
public func SUIDatePickerBounded(
    _ labelPtr: UnsafePointer<CChar>,
    _ stateRef: UnsafeMutableRawPointer,
    _ hasMin: Int32,
    _ minValue: Double,
    _ hasMax: Int32,
    _ maxValue: Double,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let label = String(cString: labelPtr)
    let state = Unmanaged<BridgedDateState>.fromOpaque(stateRef).takeUnretainedValue()
    let minDate = hasMin != 0 ? Date(timeIntervalSince1970: minValue) : nil
    let maxDate = hasMax != 0 ? Date(timeIntervalSince1970: maxValue) : nil
    let view = AnyView(BridgedBoundedDatePicker(label: label, state: state, minDate: minDate, maxDate: maxDate, mode: 2, callbackID: callbackID))
    return retainView(view)
}

@MainActor
private struct BridgedBoundedDatePicker: View {
    let label: String
    var state: BridgedDateState
    let minDate: Date?
    let maxDate: Date?
    let mode: Int32
    let callbackID: UInt

    var body: some View {
        let binding = Binding(
            get: { state.value },
            set: { newValue in
                state.setAndBump(newValue)
                _SUIButtonCallback?(callbackID)
            }
        )
        Group {
            if let minDate, let maxDate {
                picker(binding, in: minDate...maxDate)
            } else if let minDate {
                picker(binding, in: minDate...)
            } else if let maxDate {
                picker(binding, in: ...maxDate)
            } else {
                picker(binding)
            }
        }
    }

    @ViewBuilder
    private func picker(_ binding: Binding<Date>) -> some View {
        switch mode {
        case 0:
            DatePicker(label, selection: binding, displayedComponents: [.date])
        case 1:
            DatePicker(label, selection: binding, displayedComponents: [.hourAndMinute])
        default:
            DatePicker(label, selection: binding, displayedComponents: [.date, .hourAndMinute])
        }
    }

    @ViewBuilder
    private func picker(_ binding: Binding<Date>, in range: ClosedRange<Date>) -> some View {
        switch mode {
        case 0:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.date])
        case 1:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.hourAndMinute])
        default:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.date, .hourAndMinute])
        }
    }

    @ViewBuilder
    private func picker(_ binding: Binding<Date>, in range: PartialRangeFrom<Date>) -> some View {
        switch mode {
        case 0:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.date])
        case 1:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.hourAndMinute])
        default:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.date, .hourAndMinute])
        }
    }

    @ViewBuilder
    private func picker(_ binding: Binding<Date>, in range: PartialRangeThrough<Date>) -> some View {
        switch mode {
        case 0:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.date])
        case 1:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.hourAndMinute])
        default:
            DatePicker(label, selection: binding, in: range, displayedComponents: [.date, .hourAndMinute])
        }
    }
}

@_cdecl("SUIDatePickerBoundedMode")
@MainActor
public func SUIDatePickerBoundedMode(
    _ labelPtr: UnsafePointer<CChar>,
    _ stateRef: UnsafeMutableRawPointer,
    _ hasMin: Int32,
    _ minValue: Double,
    _ hasMax: Int32,
    _ maxValue: Double,
    _ mode: Int32,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let label = String(cString: labelPtr)
    let state = Unmanaged<BridgedDateState>.fromOpaque(stateRef).takeUnretainedValue()
    let minDate = hasMin != 0 ? Date(timeIntervalSince1970: minValue) : nil
    let maxDate = hasMax != 0 ? Date(timeIntervalSince1970: maxValue) : nil
    let view = AnyView(BridgedBoundedDatePicker(label: label, state: state, minDate: minDate, maxDate: maxDate, mode: mode, callbackID: callbackID))
    return retainView(view)
}

@MainActor
private struct BridgedPolicyTextField: View {
    let placeholder: String
    let state: BridgedStringState
    let allowedPattern: String
    let validationPattern: String
    let validState: BridgedBoolState?
    let onChangeID: UInt
    let onSubmitID: UInt

    @State private var draft: String
    @State private var suppressDraftSync = false

    init(
        placeholder: String,
        state: BridgedStringState,
        allowedPattern: String,
        validationPattern: String,
        validState: BridgedBoolState?,
        onChangeID: UInt,
        onSubmitID: UInt
    ) {
        self.placeholder = placeholder
        self.state = state
        self.allowedPattern = allowedPattern
        self.validationPattern = validationPattern
        self.validState = validState
        self.onChangeID = onChangeID
        self.onSubmitID = onSubmitID
        _draft = State(initialValue: state.value)
    }

    var body: some View {
        TextField(placeholder, text: $draft)
            .onAppear {
                suiSetValidState(validState, value: draft, pattern: validationPattern)
            }
            .onChange(of: draft) { _, newValue in
                if suppressDraftSync {
                    suppressDraftSync = false
                    return
                }
                guard suiRegexMatches(allowedPattern, value: newValue) else {
                    suppressDraftSync = true
                    draft = state.value
                    suiSetValidState(validState, value: state.value, pattern: validationPattern)
                    return
                }
                if state.value != newValue {
                    state.setAndBump(newValue)
                }
                suiSetValidState(validState, value: newValue, pattern: validationPattern)
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
                suiSetValidState(validState, value: newValue, pattern: validationPattern)
            }
            .onSubmit {
                if state.value != draft {
                    state.setAndBump(draft)
                }
                suiSetValidState(validState, value: draft, pattern: validationPattern)
                if onSubmitID != 0 {
                    _SUIButtonCallback?(onSubmitID)
                }
            }
    }
}

@MainActor
private struct BridgedPolicySecureField: View {
    let placeholder: String
    let state: BridgedStringState
    let allowedPattern: String
    let validationPattern: String
    let validState: BridgedBoolState?
    let onChangeID: UInt
    let onSubmitID: UInt

    @State private var draft: String
    @State private var suppressDraftSync = false

    init(
        placeholder: String,
        state: BridgedStringState,
        allowedPattern: String,
        validationPattern: String,
        validState: BridgedBoolState?,
        onChangeID: UInt,
        onSubmitID: UInt
    ) {
        self.placeholder = placeholder
        self.state = state
        self.allowedPattern = allowedPattern
        self.validationPattern = validationPattern
        self.validState = validState
        self.onChangeID = onChangeID
        self.onSubmitID = onSubmitID
        _draft = State(initialValue: state.value)
    }

    var body: some View {
        SecureField(placeholder, text: $draft)
            .onAppear {
                suiSetValidState(validState, value: draft, pattern: validationPattern)
            }
            .onChange(of: draft) { _, newValue in
                if suppressDraftSync {
                    suppressDraftSync = false
                    return
                }
                guard suiRegexMatches(allowedPattern, value: newValue) else {
                    suppressDraftSync = true
                    draft = state.value
                    suiSetValidState(validState, value: state.value, pattern: validationPattern)
                    return
                }
                if state.value != newValue {
                    state.setAndBump(newValue)
                }
                suiSetValidState(validState, value: newValue, pattern: validationPattern)
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
                suiSetValidState(validState, value: newValue, pattern: validationPattern)
            }
            .onSubmit {
                if state.value != draft {
                    state.setAndBump(draft)
                }
                suiSetValidState(validState, value: draft, pattern: validationPattern)
                if onSubmitID != 0 {
                    _SUIButtonCallback?(onSubmitID)
                }
            }
    }
}

@MainActor
private struct BridgedPolicyTextEditor: View {
    let state: BridgedStringState
    let allowedPattern: String
    let validationPattern: String
    let validState: BridgedBoolState?
    let onChangeID: UInt

    @State private var draft: String
    @State private var suppressDraftSync = false

    init(
        state: BridgedStringState,
        allowedPattern: String,
        validationPattern: String,
        validState: BridgedBoolState?,
        onChangeID: UInt
    ) {
        self.state = state
        self.allowedPattern = allowedPattern
        self.validationPattern = validationPattern
        self.validState = validState
        self.onChangeID = onChangeID
        _draft = State(initialValue: state.value)
    }

    var body: some View {
        TextEditor(text: $draft)
            .onAppear {
                suiSetValidState(validState, value: draft, pattern: validationPattern)
            }
            .onChange(of: draft) { _, newValue in
                if suppressDraftSync {
                    suppressDraftSync = false
                    return
                }
                guard suiRegexMatches(allowedPattern, value: newValue) else {
                    suppressDraftSync = true
                    draft = state.value
                    suiSetValidState(validState, value: state.value, pattern: validationPattern)
                    return
                }
                if state.value != newValue {
                    state.setAndBump(newValue)
                }
                suiSetValidState(validState, value: newValue, pattern: validationPattern)
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
                suiSetValidState(validState, value: newValue, pattern: validationPattern)
            }
    }
}

@_cdecl("SUITextFieldPolicy")
@MainActor
public func SUITextFieldPolicy(
    _ placeholderPtr: UnsafePointer<CChar>,
    _ stateRef: UnsafeMutableRawPointer,
    _ allowedPtr: UnsafePointer<CChar>,
    _ validationPtr: UnsafePointer<CChar>,
    _ validStateRef: UnsafeMutableRawPointer?,
    _ onChangeID: UInt,
    _ onSubmitID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedPolicyTextField(
            placeholder: String(cString: placeholderPtr),
            state: Unmanaged<BridgedStringState>.fromOpaque(stateRef).takeUnretainedValue(),
            allowedPattern: String(cString: allowedPtr),
            validationPattern: String(cString: validationPtr),
            validState: validStateRef.map { Unmanaged<BridgedBoolState>.fromOpaque($0).takeUnretainedValue() },
            onChangeID: onChangeID,
            onSubmitID: onSubmitID
        )
    )
    return retainView(view)
}

@_cdecl("SUISecureFieldPolicy")
@MainActor
public func SUISecureFieldPolicy(
    _ placeholderPtr: UnsafePointer<CChar>,
    _ stateRef: UnsafeMutableRawPointer,
    _ allowedPtr: UnsafePointer<CChar>,
    _ validationPtr: UnsafePointer<CChar>,
    _ validStateRef: UnsafeMutableRawPointer?,
    _ onChangeID: UInt,
    _ onSubmitID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedPolicySecureField(
            placeholder: String(cString: placeholderPtr),
            state: Unmanaged<BridgedStringState>.fromOpaque(stateRef).takeUnretainedValue(),
            allowedPattern: String(cString: allowedPtr),
            validationPattern: String(cString: validationPtr),
            validState: validStateRef.map { Unmanaged<BridgedBoolState>.fromOpaque($0).takeUnretainedValue() },
            onChangeID: onChangeID,
            onSubmitID: onSubmitID
        )
    )
    return retainView(view)
}

@_cdecl("SUITextEditorPolicy")
@MainActor
public func SUITextEditorPolicy(
    _ stateRef: UnsafeMutableRawPointer,
    _ allowedPtr: UnsafePointer<CChar>,
    _ validationPtr: UnsafePointer<CChar>,
    _ validStateRef: UnsafeMutableRawPointer?,
    _ onChangeID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedPolicyTextEditor(
            state: Unmanaged<BridgedStringState>.fromOpaque(stateRef).takeUnretainedValue(),
            allowedPattern: String(cString: allowedPtr),
            validationPattern: String(cString: validationPtr),
            validState: validStateRef.map { Unmanaged<BridgedBoolState>.fromOpaque($0).takeUnretainedValue() },
            onChangeID: onChangeID
        )
    )
    return retainView(view)
}

@_cdecl("SUIViewFrameAligned")
public func SUIViewFrameAligned(
    _ viewRef: UnsafeMutableRawPointer,
    _ width: Double,
    _ height: Double,
    _ horizontal: Int32,
    _ vertical: Int32
) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    let w = width > 0 ? CGFloat(width) : nil
    let h = height > 0 ? CGFloat(height) : nil
    let view = AnyView(base.frame(width: w, height: h, alignment: suiAlignment(horizontal, vertical)))
    return retainDerivedView(from: viewRef, view)
}

@_cdecl("SUIViewMaxFrameAligned")
public func SUIViewMaxFrameAligned(
    _ viewRef: UnsafeMutableRawPointer,
    _ maxWidth: Double,
    _ maxHeight: Double,
    _ horizontal: Int32,
    _ vertical: Int32
) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    var w: CGFloat?
    var h: CGFloat?
    if maxWidth < 0 {
        w = .infinity
    } else if maxWidth > 0 {
        w = CGFloat(maxWidth)
    }
    if maxHeight < 0 {
        h = .infinity
    } else if maxHeight > 0 {
        h = CGFloat(maxHeight)
    }
    let view = AnyView(base.frame(maxWidth: w, maxHeight: h, alignment: suiAlignment(horizontal, vertical)))
    return retainDerivedView(from: viewRef, view)
}

@MainActor
private struct BridgedSearchablePickerView: View {
    let label: String
    let prompt: String
    let selection: BridgedStringState
    let options: [SearchableChoiceOption]
    let callbackID: UInt

    @State private var query = ""

    private var filteredOptions: [SearchableChoiceOption] {
        if query.isEmpty {
            return options
        }
        return options.filter {
            $0.label.localizedCaseInsensitiveContains(query) ||
            $0.value.localizedCaseInsensitiveContains(query)
        }
    }

    private var selectedLabel: String {
        options.first(where: { $0.value == selection.value })?.label ?? selection.value
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if !label.isEmpty {
                Text(label)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if !selection.value.isEmpty {
                HStack(spacing: 6) {
                    Image(systemName: "checkmark.circle.fill")
                        .imageScale(.small)
                        .foregroundStyle(.secondary)
                    Text(selectedLabel)
                        .font(.caption)
                        .lineLimit(1)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(Color.secondary.opacity(0.10))
                .clipShape(Capsule())
            }
            TextField(prompt, text: $query)
                .textFieldStyle(.roundedBorder)
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 6) {
                    if filteredOptions.isEmpty {
                        Text("No matches")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 8)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    } else {
                        ForEach(filteredOptions, id: \.value) { option in
                            Button {
                                selection.value = option.value
                                if callbackID != 0 {
                                    _SUIButtonCallback?(callbackID)
                                }
                            } label: {
                                HStack {
                                    Text(option.label)
                                    Spacer()
                                    if selection.value == option.value {
                                        Image(systemName: "checkmark")
                                            .foregroundStyle(.secondary)
                                    }
                                }
                                .padding(.horizontal, 10)
                                .padding(.vertical, 7)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(selection.value == option.value ? Color.accentColor.opacity(0.14) : Color.clear)
                                .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(8)
            }
            .frame(maxHeight: min(CGFloat(max(filteredOptions.count, 1)) * 34, 132))
            .background(Color.secondary.opacity(0.06))
            .clipShape(RoundedRectangle(cornerRadius: 10))
        }
    }
}

@MainActor
private struct BridgedSearchableMultiPickerView: View {
    let label: String
    let prompt: String
    let selection: BridgedStringListState
    let options: [SearchableChoiceOption]
    let callbackID: UInt

    @State private var query = ""

    private var filteredOptions: [SearchableChoiceOption] {
        if query.isEmpty {
            return options
        }
        return options.filter {
            $0.label.localizedCaseInsensitiveContains(query) ||
            $0.value.localizedCaseInsensitiveContains(query)
        }
    }

    private func isSelected(_ option: SearchableChoiceOption) -> Bool {
        selection.value.contains(option.value)
    }

    private func toggle(_ option: SearchableChoiceOption) {
        var values = selection.value
        if let i = values.firstIndex(of: option.value) {
            values.remove(at: i)
        } else {
            values.append(option.value)
        }
        selection.value = values
        if callbackID != 0 {
            _SUIButtonCallback?(callbackID)
        }
    }

    private var selectedLabels: [String] {
        options.filter { selection.value.contains($0.value) }.map(\.label)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if !label.isEmpty {
                Text(label)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if !selectedLabels.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 6) {
                        ForEach(selectedLabels, id: \.self) { label in
                            Text(label)
                                .font(.caption)
                                .padding(.horizontal, 10)
                                .padding(.vertical, 6)
                                .background(Color.accentColor.opacity(0.14))
                                .clipShape(Capsule())
                        }
                    }
                }
            }
            TextField(prompt, text: $query)
                .textFieldStyle(.roundedBorder)
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 6) {
                    if filteredOptions.isEmpty {
                        Text("No matches")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 8)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    } else {
                        ForEach(filteredOptions, id: \.value) { option in
                            Button {
                                toggle(option)
                            } label: {
                                HStack {
                                    Image(systemName: isSelected(option) ? "checkmark.square.fill" : "square")
                                        .foregroundStyle(isSelected(option) ? Color.accentColor : Color.secondary)
                                    Text(option.label)
                                    Spacer()
                                }
                                .padding(.horizontal, 10)
                                .padding(.vertical, 7)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(isSelected(option) ? Color.accentColor.opacity(0.14) : Color.clear)
                                .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(8)
            }
            .frame(maxHeight: min(CGFloat(max(filteredOptions.count, 1)) * 34, 144))
            .background(Color.secondary.opacity(0.06))
            .clipShape(RoundedRectangle(cornerRadius: 10))
        }
    }
}

@_cdecl("SUISearchablePicker")
@MainActor
public func SUISearchablePicker(
    _ labelPtr: UnsafePointer<CChar>,
    _ promptPtr: UnsafePointer<CChar>,
    _ selectionRef: UnsafeMutableRawPointer,
    _ optionsPtr: UnsafePointer<CChar>,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedSearchablePickerView(
            label: String(cString: labelPtr),
            prompt: String(cString: promptPtr),
            selection: Unmanaged<BridgedStringState>.fromOpaque(selectionRef).takeUnretainedValue(),
            options: suiChoiceOptions(from: String(cString: optionsPtr)),
            callbackID: callbackID
        )
    )
    return retainView(view)
}

@_cdecl("SUIAccessibilityIdentifier")
public func SUIAccessibilityIdentifier(_ viewRef: UnsafeMutableRawPointer, _ identifier: UnsafePointer<CChar>) -> UnsafeMutableRawPointer {
    let base = Unmanaged<Box<AnyView>>.fromOpaque(viewRef).takeUnretainedValue().value
    let id = String(cString: identifier)
    let view = AnyView(base.accessibilityIdentifier(id))
    return retainDerivedView(from: viewRef, view)
}

@_cdecl("SUISearchableMultiPicker")
@MainActor
public func SUISearchableMultiPicker(
    _ labelPtr: UnsafePointer<CChar>,
    _ promptPtr: UnsafePointer<CChar>,
    _ selectionRef: UnsafeMutableRawPointer,
    _ optionsPtr: UnsafePointer<CChar>,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedSearchableMultiPickerView(
            label: String(cString: labelPtr),
            prompt: String(cString: promptPtr),
            selection: Unmanaged<BridgedStringListState>.fromOpaque(selectionRef).takeUnretainedValue(),
            options: suiChoiceOptions(from: String(cString: optionsPtr)),
            callbackID: callbackID
        )
    )
    return retainView(view)
}

// Packed-wire picker entry points (P4). Options are encoded as
// [count: UInt32] [ [label_len: UInt32] [label] [value_len: UInt32] [value] ]*
@_cdecl("SUISearchablePickerPacked")
@MainActor
public func SUISearchablePickerPacked(
    _ labelPtr: UnsafePointer<CChar>,
    _ promptPtr: UnsafePointer<CChar>,
    _ selectionRef: UnsafeMutableRawPointer,
    _ optionsPtr: UnsafePointer<UInt8>?,
    _ optionsLen: Int32,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedSearchablePickerView(
            label: String(cString: labelPtr),
            prompt: String(cString: promptPtr),
            selection: Unmanaged<BridgedStringState>.fromOpaque(selectionRef).takeUnretainedValue(),
            options: suiChoiceOptions(packed: optionsPtr, length: Int(optionsLen)),
            callbackID: callbackID
        )
    )
    return retainView(view)
}

@_cdecl("SUISearchableMultiPickerPacked")
@MainActor
public func SUISearchableMultiPickerPacked(
    _ labelPtr: UnsafePointer<CChar>,
    _ promptPtr: UnsafePointer<CChar>,
    _ selectionRef: UnsafeMutableRawPointer,
    _ optionsPtr: UnsafePointer<UInt8>?,
    _ optionsLen: Int32,
    _ callbackID: UInt
) -> UnsafeMutableRawPointer {
    let view = AnyView(
        BridgedSearchableMultiPickerView(
            label: String(cString: labelPtr),
            prompt: String(cString: promptPtr),
            selection: Unmanaged<BridgedStringListState>.fromOpaque(selectionRef).takeUnretainedValue(),
            options: suiChoiceOptions(packed: optionsPtr, length: Int(optionsLen)),
            callbackID: callbackID
        )
    )
    return retainView(view)
}

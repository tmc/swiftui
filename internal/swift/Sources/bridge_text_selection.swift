import AppKit
import SwiftUI

@Observable
final class BridgedTextSelectionState: @unchecked Sendable {
  var start: Int
  var end: Int

  init(start: Int, end: Int) {
    let normalized = suiNormalizeSelection(start, end)
    self.start = normalized.start
    self.end = normalized.end
  }

  func set(_ start: Int, _ end: Int) {
    let normalized = suiNormalizeSelection(start, end)
    if self.start == normalized.start, self.end == normalized.end {
      return
    }
    self.start = normalized.start
    self.end = normalized.end
  }
}

private struct SUISelectionRange {
  let start: Int
  let end: Int
}

private func suiNormalizeSelection(_ start: Int, _ end: Int) -> SUISelectionRange {
  let lower = max(0, min(start, end))
  let upper = max(lower, max(start, end))
  return SUISelectionRange(start: lower, end: upper)
}

@_cdecl("SUIStateCreateTextSelection")
public func SUIStateCreateTextSelection(_ start: Int32, _ end: Int32) -> UnsafeMutableRawPointer {
  Unmanaged.passRetained(BridgedTextSelectionState(start: Int(start), end: Int(end))).toOpaque()
}

@_cdecl("SUIStateGetTextSelectionStart")
public func SUIStateGetTextSelectionStart(_ ref: UnsafeMutableRawPointer) -> Int32 {
  Int32(Unmanaged<BridgedTextSelectionState>.fromOpaque(ref).takeUnretainedValue().start)
}

@_cdecl("SUIStateGetTextSelectionEnd")
public func SUIStateGetTextSelectionEnd(_ ref: UnsafeMutableRawPointer) -> Int32 {
  Int32(Unmanaged<BridgedTextSelectionState>.fromOpaque(ref).takeUnretainedValue().end)
}

@_cdecl("SUIStateSetTextSelection")
public func SUIStateSetTextSelection(_ ref: UnsafeMutableRawPointer, _ start: Int32, _ end: Int32) {
  let selection = Unmanaged<BridgedTextSelectionState>.fromOpaque(ref).takeUnretainedValue()
  selection.set(Int(start), Int(end))
}

private struct SUITextFieldStyleKindKey: EnvironmentKey {
  static let defaultValue: Int32 = 0
}

private struct SUIHiddenScrollContentBackgroundKey: EnvironmentKey {
  static let defaultValue = false
}

extension EnvironmentValues {
  var suiTextFieldStyleKind: Int32 {
    get { self[SUITextFieldStyleKindKey.self] }
    set { self[SUITextFieldStyleKindKey.self] = newValue }
  }

  var suiScrollContentBackgroundHidden: Bool {
    get { self[SUIHiddenScrollContentBackgroundKey.self] }
    set { self[SUIHiddenScrollContentBackgroundKey.self] = newValue }
  }
}

private func suiClampSelection(
  _ selectionStart: Binding<Int>?,
  _ selectionEnd: Binding<Int>?,
  text: String
) -> NSRange? {
  guard let selectionStart, let selectionEnd else {
    return nil
  }
  let normalized = suiNormalizeSelection(selectionStart.wrappedValue, selectionEnd.wrappedValue)
  let limit = text.utf16.count
  let clamped = SUISelectionRange(start: min(normalized.start, limit), end: min(normalized.end, limit))
  if selectionStart.wrappedValue != clamped.start {
    selectionStart.wrappedValue = clamped.start
  }
  if selectionEnd.wrappedValue != clamped.end {
    selectionEnd.wrappedValue = clamped.end
  }
  return NSRange(location: clamped.start, length: clamped.end - clamped.start)
}

private func suiSyncSelection(_ range: NSRange, start: Binding<Int>?, end: Binding<Int>?) {
  guard let start, let end else {
    return
  }
  let lower = max(0, range.location)
  let upper = max(lower, range.location + range.length)
  if start.wrappedValue != lower {
    start.wrappedValue = lower
  }
  if end.wrappedValue != upper {
    end.wrappedValue = upper
  }
}

@MainActor
private func suiApplyTextFieldStyle(_ field: NSTextField, style: Int32) {
  switch style {
  case 2:
    field.isBezeled = false
    field.isBordered = false
    field.drawsBackground = false
  default:
    field.isBezeled = true
    field.isBordered = true
    field.drawsBackground = true
    field.bezelStyle = .roundedBezel
  }
}

@MainActor
final class SUIEditableFieldCoordinator: NSObject, NSTextFieldDelegate {
  var text: Binding<String>
  var selectionStart: Binding<Int>?
  var selectionEnd: Binding<Int>?
  var onChange: (() -> Void)?
  var onSubmit: (() -> Void)?
  var applying = false
  var pendingSelection: NSRange?

  init(
    text: Binding<String>,
    selectionStart: Binding<Int>?,
    selectionEnd: Binding<Int>?,
    onChange: (() -> Void)?,
    onSubmit: (() -> Void)?
  ) {
    self.text = text
    self.selectionStart = selectionStart
    self.selectionEnd = selectionEnd
    self.onChange = onChange
    self.onSubmit = onSubmit
  }

  func update(
    text: Binding<String>,
    selectionStart: Binding<Int>?,
    selectionEnd: Binding<Int>?,
    onChange: (() -> Void)?,
    onSubmit: (() -> Void)?
  ) {
    self.text = text
    self.selectionStart = selectionStart
    self.selectionEnd = selectionEnd
    self.onChange = onChange
    self.onSubmit = onSubmit
  }

  func apply(to field: NSTextField) {
    let value = text.wrappedValue
    if field.stringValue != value {
      applying = true
      field.stringValue = value
      applying = false
    }
    applySelection(to: field)
  }

  func applySelection(to field: NSTextField) {
    guard let range = suiClampSelection(selectionStart, selectionEnd, text: field.stringValue) else {
      return
    }
    if let editor = field.currentEditor() as? NSTextView {
      if !NSEqualRanges(editor.selectedRange(), range) {
        editor.setSelectedRange(range)
      }
      pendingSelection = nil
      return
    }
    pendingSelection = range
  }

  func sync(from field: NSTextField) {
    if applying {
      return
    }
    let value = field.stringValue
    if text.wrappedValue != value {
      text.wrappedValue = value
      onChange?()
    }
    if let editor = field.currentEditor() as? NSTextView {
      suiSyncSelection(editor.selectedRange(), start: selectionStart, end: selectionEnd)
    }
  }

  func controlTextDidBeginEditing(_ notification: Notification) {
    guard let field = notification.object as? NSTextField else { return }
    if let pendingSelection, let editor = field.currentEditor() as? NSTextView {
      editor.setSelectedRange(pendingSelection)
      self.pendingSelection = nil
    }
    sync(from: field)
  }

  func controlTextDidChange(_ notification: Notification) {
    guard let field = notification.object as? NSTextField else { return }
    sync(from: field)
  }

  func controlTextDidEndEditing(_ notification: Notification) {
    guard let field = notification.object as? NSTextField else { return }
    sync(from: field)
  }

  @objc func submit(_ sender: Any?) {
    if let field = sender as? NSTextField {
      sync(from: field)
    }
    onSubmit?()
  }
}

@MainActor
private struct SUIEditableField<Control: NSTextField>: NSViewRepresentable {
  let placeholder: String
  @Binding var text: String
  let selectionStart: Binding<Int>?
  let selectionEnd: Binding<Int>?
  let onChange: (() -> Void)?
  let onSubmit: (() -> Void)?
  let makeControl: () -> Control

  @Environment(\.suiTextFieldStyleKind) private var style

  func makeCoordinator() -> SUIEditableFieldCoordinator {
    SUIEditableFieldCoordinator(
      text: $text,
      selectionStart: selectionStart,
      selectionEnd: selectionEnd,
      onChange: onChange,
      onSubmit: onSubmit
    )
  }

  func makeNSView(context: Context) -> Control {
    let control = makeControl()
    control.placeholderString = placeholder
    control.delegate = context.coordinator
    control.target = context.coordinator
    control.action = #selector(SUIEditableFieldCoordinator.submit(_:))
    control.focusRingType = .default
    control.controlSize = .regular
    control.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
    suiApplyTextFieldStyle(control, style: style)
    context.coordinator.apply(to: control)
    return control
  }

  func updateNSView(_ control: Control, context: Context) {
    control.placeholderString = placeholder
    context.coordinator.update(
      text: $text,
      selectionStart: selectionStart,
      selectionEnd: selectionEnd,
      onChange: onChange,
      onSubmit: onSubmit
    )
    suiApplyTextFieldStyle(control, style: style)
    context.coordinator.apply(to: control)
  }
}

@MainActor
struct SUIEditableTextField: View {
  let placeholder: String
  @Binding var text: String
  let selectionStart: Binding<Int>?
  let selectionEnd: Binding<Int>?
  let onChange: (() -> Void)?
  let onSubmit: (() -> Void)?

  var body: some View {
    SUIEditableField(
      placeholder: placeholder,
      text: $text,
      selectionStart: selectionStart,
      selectionEnd: selectionEnd,
      onChange: onChange,
      onSubmit: onSubmit,
      makeControl: { NSTextField(string: "") }
    )
  }
}

@MainActor
struct SUIEditableSecureField: View {
  let placeholder: String
  @Binding var text: String
  let selectionStart: Binding<Int>?
  let selectionEnd: Binding<Int>?
  let onChange: (() -> Void)?
  let onSubmit: (() -> Void)?

  var body: some View {
    SUIEditableField(
      placeholder: placeholder,
      text: $text,
      selectionStart: selectionStart,
      selectionEnd: selectionEnd,
      onChange: onChange,
      onSubmit: onSubmit,
      makeControl: { NSSecureTextField(string: "") }
    )
  }
}

@MainActor
final class SUIEditableTextEditorCoordinator: NSObject, NSTextViewDelegate {
  var text: Binding<String>
  var selectionStart: Binding<Int>?
  var selectionEnd: Binding<Int>?
  var onChange: (() -> Void)?
  var applying = false

  init(
    text: Binding<String>,
    selectionStart: Binding<Int>?,
    selectionEnd: Binding<Int>?,
    onChange: (() -> Void)?
  ) {
    self.text = text
    self.selectionStart = selectionStart
    self.selectionEnd = selectionEnd
    self.onChange = onChange
  }

  func update(
    text: Binding<String>,
    selectionStart: Binding<Int>?,
    selectionEnd: Binding<Int>?,
    onChange: (() -> Void)?
  ) {
    self.text = text
    self.selectionStart = selectionStart
    self.selectionEnd = selectionEnd
    self.onChange = onChange
  }

  func apply(to textView: NSTextView) {
    let value = text.wrappedValue
    if textView.string != value {
      applying = true
      textView.string = value
      applying = false
    }
    if let range = suiClampSelection(selectionStart, selectionEnd, text: textView.string),
      !NSEqualRanges(textView.selectedRange(), range)
    {
      textView.setSelectedRange(range)
    }
  }

  func sync(from textView: NSTextView) {
    if applying {
      return
    }
    let value = textView.string
    if text.wrappedValue != value {
      text.wrappedValue = value
      onChange?()
    }
    suiSyncSelection(textView.selectedRange(), start: selectionStart, end: selectionEnd)
  }

  func textDidChange(_ notification: Notification) {
    guard let textView = notification.object as? NSTextView else { return }
    sync(from: textView)
  }

  func textViewDidChangeSelection(_ notification: Notification) {
    guard let textView = notification.object as? NSTextView else { return }
    suiSyncSelection(textView.selectedRange(), start: selectionStart, end: selectionEnd)
  }
}

@MainActor
private func suiConfigureTextEditorBackground(
  _ scrollView: NSScrollView,
  _ textView: NSTextView,
  hidden: Bool
) {
  scrollView.borderType = hidden ? .noBorder : .bezelBorder
  scrollView.drawsBackground = !hidden
  textView.drawsBackground = !hidden
  if hidden {
    scrollView.backgroundColor = .clear
    textView.backgroundColor = .clear
  } else {
    scrollView.backgroundColor = .textBackgroundColor
    textView.backgroundColor = .textBackgroundColor
  }
}

@MainActor
struct SUIEditableTextEditor: NSViewRepresentable {
  @Binding var text: String
  let selectionStart: Binding<Int>?
  let selectionEnd: Binding<Int>?
  let onChange: (() -> Void)?
  let onSubmit: (() -> Void)?

  @Environment(\.suiScrollContentBackgroundHidden) private var hidesBackground

  func makeCoordinator() -> SUIEditableTextEditorCoordinator {
    SUIEditableTextEditorCoordinator(
      text: $text,
      selectionStart: selectionStart,
      selectionEnd: selectionEnd,
      onChange: onChange
    )
  }

  func makeNSView(context: Context) -> NSScrollView {
    let scrollView = NSScrollView()
    scrollView.hasVerticalScroller = true
    scrollView.hasHorizontalScroller = false
    scrollView.autohidesScrollers = true

    let textView = NSTextView()
    textView.delegate = context.coordinator
    textView.isRichText = false
    textView.importsGraphics = false
    textView.usesAdaptiveColorMappingForDarkAppearance = true
    textView.usesFindPanel = true
    textView.allowsUndo = true
    textView.isVerticallyResizable = true
    textView.isHorizontallyResizable = false
    textView.autoresizingMask = [.width]
    textView.textContainerInset = NSSize(width: 6, height: 8)
    textView.textContainer?.containerSize = NSSize(width: 0, height: CGFloat.greatestFiniteMagnitude)
    textView.textContainer?.widthTracksTextView = true
    scrollView.documentView = textView

    suiConfigureTextEditorBackground(scrollView, textView, hidden: hidesBackground)
    context.coordinator.apply(to: textView)
    return scrollView
  }

  func updateNSView(_ scrollView: NSScrollView, context: Context) {
    guard let textView = scrollView.documentView as? NSTextView else { return }
    context.coordinator.update(
      text: $text,
      selectionStart: selectionStart,
      selectionEnd: selectionEnd,
      onChange: onChange
    )
    suiConfigureTextEditorBackground(scrollView, textView, hidden: hidesBackground)
    context.coordinator.apply(to: textView)
  }
}

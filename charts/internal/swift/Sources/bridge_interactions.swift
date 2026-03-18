import Charts
import Foundation
import SwiftUI

private func interactionFixedString(_ value: Double, precision: Int) -> String {
    String(format: "%.\(max(precision, 0))f", value)
}

private func interactionCompactString(_ value: Double, precision: Int) -> String {
    let magnitude = abs(value)
    switch magnitude {
    case 1_000_000_000_000...:
        return interactionFixedString(value / 1_000_000_000_000, precision: precision) + "T"
    case 1_000_000_000...:
        return interactionFixedString(value / 1_000_000_000, precision: precision) + "B"
    case 1_000_000...:
        return interactionFixedString(value / 1_000_000, precision: precision) + "M"
    case 1_000...:
        return interactionFixedString(value / 1_000, precision: precision) + "K"
    default:
        return interactionFixedString(value, precision: precision)
    }
}

private func interactionDurationString(_ value: Double, format: ValueFormatSpec) -> String {
    let scale: Double
    let suffix: String
    switch format.unit {
    case 4:
        scale = 1.0 / 60.0
        suffix = " min"
    case 1:
        scale = 1_000
        suffix = " ms"
    case 2:
        scale = 1_000_000
        suffix = " µs"
    case 3:
        scale = 1_000_000_000
        suffix = " ns"
    default:
        scale = 1
        suffix = " s"
    }
    return interactionFixedString(value * scale, precision: format.precision) + suffix
}

private func decodeDateComponents(_ spec: DateComponentsSpec) -> DateComponents {
    var out = DateComponents()
    out.year = spec.year == 0 ? nil : spec.year
    out.month = spec.month == 0 ? nil : spec.month
    out.day = spec.day == 0 ? nil : spec.day
    out.hour = spec.hour == 0 ? nil : spec.hour
    out.minute = spec.minute == 0 ? nil : spec.minute
    out.second = spec.second == 0 ? nil : spec.second
    return out
}

private func decodeLimitBehavior(_ raw: Int32) -> ValueAlignedLimitBehavior {
    switch raw {
    case 1: return .always
    case 2: return .never
    default: return .automatic
    }
}

private func decodeMajorAlignmentNumber(_ spec: MajorValueAlignmentSpec?) -> MajorValueAlignment<Double>? {
    guard let spec else { return nil }
    switch spec.kind {
    case 1:
        return .unit(spec.numberUnit)
    case 3:
        return .page
    default:
        return nil
    }
}

private func decodeMajorAlignmentDate(_ spec: MajorValueAlignmentSpec?) -> MajorValueAlignment<Date>? {
    guard let spec else { return nil }
    switch spec.kind {
    case 2:
        return .matching(decodeDateComponents(spec.dateUnit))
    case 3:
        return .page
    default:
        return nil
    }
}

@MainActor
func applySelections(_ view: AnyView, spec: ChartSpec) -> AnyView {
    var chart = view
    if let state = spec.xSelection {
        switch state.kind {
        case 2:
            let ref = Unmanaged<OptionalNumberBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartXSelection(value: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 3:
            let ref = Unmanaged<OptionalDateBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartXSelection(value: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    if let state = spec.xSelectionRange {
        switch state.kind {
        case 4:
            let ref = Unmanaged<NumberRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartXSelection(range: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 5:
            let ref = Unmanaged<DateRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartXSelection(range: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    if let state = spec.ySelection {
        switch state.kind {
        case 2:
            let ref = Unmanaged<OptionalNumberBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartYSelection(value: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 3:
            let ref = Unmanaged<OptionalDateBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartYSelection(value: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    if let state = spec.ySelectionRange {
        switch state.kind {
        case 4:
            let ref = Unmanaged<NumberRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartYSelection(range: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 5:
            let ref = Unmanaged<DateRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartYSelection(range: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    if let state = spec.angleSelection {
        switch state.kind {
        case 2:
            let ref = Unmanaged<OptionalNumberBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartAngleSelection(value: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 3:
            let ref = Unmanaged<OptionalDateBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
            chart = AnyView(chart.chartAngleSelection(value: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    return chart
}

@MainActor
func applyScrolling(
    _ view: AnyView,
    axes: Int32,
    position: ScrollPositionSpec?,
    target: ScrollTargetBehaviorSpec?,
    xBinding: StateRefSpec?,
    yBinding: StateRefSpec?,
    xVisibleLength: VisibleDomainLengthSpec?,
    yVisibleLength: VisibleDomainLengthSpec?
) -> AnyView {
    var chart = view
    switch axes {
    case 1:
        chart = AnyView(chart.chartScrollableAxes(.horizontal))
    case 2:
        chart = AnyView(chart.chartScrollableAxes(.vertical))
    case 3:
        chart = AnyView(chart.chartScrollableAxes([.horizontal, .vertical]))
    default:
        break
    }
    if let position {
        switch (position.axis, position.kind) {
        case (1, 1):
            chart = AnyView(chart.chartScrollPosition(initialY: Date(timeIntervalSince1970: Double(position.time) / 1000)))
        case (1, _):
            chart = AnyView(chart.chartScrollPosition(initialY: position.number))
        case (_, 1):
            chart = AnyView(chart.chartScrollPosition(initialX: Date(timeIntervalSince1970: Double(position.time) / 1000)))
        default:
            chart = AnyView(chart.chartScrollPosition(initialX: position.number))
        }
    }
    if let xBinding {
        switch xBinding.kind {
        case 0:
            let ref = Unmanaged<NumberBindingState>.fromOpaque(decodeStateRef(xBinding)!).takeUnretainedValue()
            chart = AnyView(chart.chartScrollPosition(x: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 1:
            let ref = Unmanaged<DateBindingState>.fromOpaque(decodeStateRef(xBinding)!).takeUnretainedValue()
            chart = AnyView(chart.chartScrollPosition(x: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    if let yBinding {
        switch yBinding.kind {
        case 0:
            let ref = Unmanaged<NumberBindingState>.fromOpaque(decodeStateRef(yBinding)!).takeUnretainedValue()
            chart = AnyView(chart.chartScrollPosition(y: Binding(get: { ref.value }, set: { ref.value = $0 })))
        case 1:
            let ref = Unmanaged<DateBindingState>.fromOpaque(decodeStateRef(yBinding)!).takeUnretainedValue()
            chart = AnyView(chart.chartScrollPosition(y: Binding(get: { ref.value }, set: { ref.value = $0 })))
        default:
            break
        }
    }
    if let xVisibleLength {
        chart = AnyView(chart.chartXVisibleDomain(length: xVisibleLength.number))
    }
    if let yVisibleLength {
        chart = AnyView(chart.chartYVisibleDomain(length: yVisibleLength.number))
    }
    if let target {
        switch target.kind {
        case 0:
            chart = AnyView(chart.chartScrollTargetBehavior(.paging))
        case 1:
            chart = AnyView(chart.chartScrollTargetBehavior(.valueAligned(
                unit: target.xUnit,
                majorAlignment: decodeMajorAlignmentNumber(target.xMajor),
                limitBehavior: decodeLimitBehavior(target.limit)
            )))
        case 2:
            chart = AnyView(chart.chartScrollTargetBehavior(.valueAligned(
                matching: decodeDateComponents(target.xDate),
                majorAlignment: decodeMajorAlignmentDate(target.xMajor),
                limitBehavior: decodeLimitBehavior(target.limit)
            )))
        case 3:
            chart = AnyView(chart.chartScrollTargetBehavior(.valueAligned(
                xUnit: target.xUnit,
                yUnit: target.yUnit,
                xMajorAlignment: decodeMajorAlignmentNumber(target.xMajor),
                yMajorAlignment: decodeMajorAlignmentNumber(target.yMajor),
                limitBehavior: decodeLimitBehavior(target.limit)
            )))
        case 4:
            chart = AnyView(chart.chartScrollTargetBehavior(.valueAligned(
                xMatching: decodeDateComponents(target.xDate),
                yMatching: decodeDateComponents(target.yDate),
                xMajorAlignment: decodeMajorAlignmentDate(target.xMajor),
                yMajorAlignment: decodeMajorAlignmentDate(target.yMajor),
                limitBehavior: decodeLimitBehavior(target.limit)
            )))
        default:
            break
        }
    }
    return chart
}

private func formatSelectionNumber(_ value: Double, format: ValueFormatSpec) -> String {
    switch format.kind {
    case 1:
        return interactionFixedString(value, precision: format.precision)
    case 2:
        return interactionFixedString(value * 100, precision: format.precision) + "%"
    case 3:
        return interactionCompactString(value, precision: format.precision)
    case 4:
        return interactionDurationString(value, format: format)
    case 5:
        return interactionFixedString(value, precision: format.precision) + format.suffix
    default:
        return interactionFixedString(value, precision: 2)
    }
}

@MainActor
private func selectionValueText(_ state: StateRefSpec?, format: ValueFormatSpec) -> String? {
    guard let state else { return nil }
    switch state.kind {
    case 2:
        let ref = Unmanaged<OptionalNumberBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let value = ref.value else { return nil }
        return formatSelectionNumber(value, format: format)
    case 3:
        let ref = Unmanaged<OptionalDateBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let value = ref.value else { return nil }
        return value.formatted(date: .abbreviated, time: .shortened)
    default:
        return nil
    }
}

@MainActor
private func xPosition(_ proxy: ChartProxy, _ state: StateRefSpec?) -> CGFloat? {
    guard let state else { return nil }
    switch state.kind {
    case 2:
        let ref = Unmanaged<OptionalNumberBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let value = ref.value else { return nil }
        return proxy.position(forX: value)
    case 3:
        let ref = Unmanaged<OptionalDateBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let value = ref.value else { return nil }
        return proxy.position(forX: value)
    default:
        return nil
    }
}

@MainActor
private func yPosition(_ proxy: ChartProxy, _ state: StateRefSpec?) -> CGFloat? {
    guard let state else { return nil }
    switch state.kind {
    case 2:
        let ref = Unmanaged<OptionalNumberBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let value = ref.value else { return nil }
        return proxy.position(forY: value)
    case 3:
        let ref = Unmanaged<OptionalDateBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let value = ref.value else { return nil }
        return proxy.position(forY: value)
    default:
        return nil
    }
}

@MainActor
private func xRangePositions(_ proxy: ChartProxy, _ state: StateRefSpec?) -> ClosedRange<CGFloat>? {
    guard let state else { return nil }
    switch state.kind {
    case 4:
        let ref = Unmanaged<NumberRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let range = ref.value, let lower = proxy.position(forX: range.lowerBound), let upper = proxy.position(forX: range.upperBound) else { return nil }
        return min(lower, upper) ... max(lower, upper)
    case 5:
        let ref = Unmanaged<DateRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let range = ref.value, let lower = proxy.position(forX: range.lowerBound), let upper = proxy.position(forX: range.upperBound) else { return nil }
        return min(lower, upper) ... max(lower, upper)
    default:
        return nil
    }
}

@MainActor
private func yRangePositions(_ proxy: ChartProxy, _ state: StateRefSpec?) -> ClosedRange<CGFloat>? {
    guard let state else { return nil }
    switch state.kind {
    case 4:
        let ref = Unmanaged<NumberRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let range = ref.value, let lower = proxy.position(forY: range.lowerBound), let upper = proxy.position(forY: range.upperBound) else { return nil }
        return min(lower, upper) ... max(lower, upper)
    case 5:
        let ref = Unmanaged<DateRangeBindingState>.fromOpaque(decodeStateRef(state)!).takeUnretainedValue()
        guard let range = ref.value, let lower = proxy.position(forY: range.lowerBound), let upper = proxy.position(forY: range.upperBound) else { return nil }
        return min(lower, upper) ... max(lower, upper)
    default:
        return nil
    }
}

private func overlayAlignment(_ raw: Int32) -> Alignment {
    switch raw {
    case 1: return .leading
    case 2: return .trailing
    case 3: return .top
    case 4: return .bottom
    case 5: return .topLeading
    case 6: return .topTrailing
    case 7: return .bottomLeading
    case 8: return .bottomTrailing
    default: return .center
    }
}

@MainActor
@ViewBuilder
private func proxyLayerView(_ proxy: ChartProxy, _ spec: ProxyLayerSpec) -> some View {
    GeometryReader { geo in
        let frame = proxy.plotFrame.map { geo[$0] } ?? geo.frame(in: .local)
        switch spec.kind {
        case 0:
            ZStack {
                if let x = xPosition(proxy, spec.xState) {
                    Rectangle()
                        .fill(decodeColor(spec.colorR, spec.colorG, spec.colorB, spec.colorA))
                        .frame(width: max(spec.width, 1), height: frame.height)
                        .position(x: frame.minX + x, y: frame.midY)
                }
                if let y = yPosition(proxy, spec.yState) {
                    Rectangle()
                        .fill(decodeColor(spec.colorR, spec.colorG, spec.colorB, spec.colorA))
                        .frame(width: frame.width, height: max(spec.width, 1))
                        .position(x: frame.midX, y: frame.minY + y)
                }
            }
        case 1:
            VStack(alignment: .leading, spacing: 4) {
                if let x = selectionValueText(spec.xState, format: spec.xFormat) {
                    Text("x \(x)").monospacedDigit()
                }
                if let y = selectionValueText(spec.yState, format: spec.yFormat) {
                    Text("y \(y)").monospacedDigit()
                }
            }
            .font(.caption2)
            .padding(6)
            .background(.thinMaterial, in: RoundedRectangle(cornerRadius: 6))
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: overlayAlignment(spec.alignment))
        case 2:
            if let span = xRangePositions(proxy, spec.range) {
                Rectangle()
                    .fill(decodeColor(spec.colorR, spec.colorG, spec.colorB, spec.colorA))
                    .frame(width: span.upperBound - span.lowerBound, height: frame.height)
                    .position(x: frame.minX + (span.lowerBound + span.upperBound) / 2, y: frame.midY)
            }
        case 3:
            if let span = yRangePositions(proxy, spec.range) {
                Rectangle()
                    .fill(decodeColor(spec.colorR, spec.colorG, spec.colorB, spec.colorA))
                    .frame(width: frame.width, height: span.upperBound - span.lowerBound)
                    .position(x: frame.midX, y: frame.minY + (span.lowerBound + span.upperBound) / 2)
            }
        default:
            EmptyView()
        }
    }
}

@MainActor
func applyProxyLayers(_ view: AnyView, overlays: [ProxyLayerSpec]?, backgrounds: [ProxyLayerSpec]?) -> AnyView {
    var chart = view
    if let overlays, !overlays.isEmpty {
        for overlay in overlays {
            chart = AnyView(chart.chartOverlay { proxy in
                proxyLayerView(proxy, overlay)
            })
        }
    }
    if let backgrounds, !backgrounds.isEmpty {
        for background in backgrounds {
            chart = AnyView(chart.chartBackground { proxy in
                proxyLayerView(proxy, background)
            })
        }
    }
    return chart
}

@MainActor
func applyProxyGesture(_ view: AnyView, gesture spec: ProxyGestureSpec?) -> AnyView {
    guard let spec else { return view }
    return AnyView(view.chartGesture { proxy in
        DragGesture(minimumDistance: spec.minDistance).onChanged { value in
            switch spec.kind {
            case 0:
                proxy.selectXValue(at: value.location.x)
            case 1:
                proxy.selectXRange(from: value.startLocation.x, to: value.location.x)
            case 2:
                proxy.selectYValue(at: value.location.y)
            case 3:
                proxy.selectYRange(from: value.startLocation.y, to: value.location.y)
            case 4:
                proxy.selectAngleValue(at: proxy.angle(at: value.location))
            default:
                break
            }
        }
    })
}

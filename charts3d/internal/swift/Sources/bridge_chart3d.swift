import Charts
import Foundation
import Spatial
import SwiftUI

nonisolated(unsafe) var _CHSurfaceCallback: (@convention(c) (UInt, Double, Double) -> Double)?

@_cdecl("CHSetSurfaceCallback")
public func CHSetSurfaceCallback(_ fn: @convention(c) (UInt, Double, Double) -> Double) {
    _CHSurfaceCallback = fn
}

@_cdecl("CHBuildChart3D")
@MainActor
public func CHBuildChart3D(_ specPtr: UnsafePointer<CChar>, _ specLen: Int32) -> UnsafeMutableRawPointer {
    let data = Data(bytes: specPtr, count: Int(specLen))
    let decoder = JSONDecoder()
    guard let spec = try? decoder.decode(Chart3DSpec.self, from: data) else {
        return Unmanaged.passRetained(Box(AnyView(EmptyView()))).toOpaque()
    }
    let view = AnyView(buildChart3DView(spec))
    return Unmanaged.passRetained(Box(view)).toOpaque()
}

private func dim(_ spec: MarkSpec, role: Int32) -> DimSpec? {
    spec.dims.first { $0.role == role }
}

private func decodeScaleType(_ spec: ScaleTypeSpec) -> Charts.ScaleType? {
    switch spec.kind {
    case 1:
        return .linear
    case 2:
        return .log
    case 3:
        return .date
    case 4:
        return .power(exponent: spec.value)
    case 5:
        return .squareRoot
    case 6:
        return .symmetricLog(slopeAtZero: spec.value)
    default:
        return nil
    }
}

private func dateRange(_ domain: DomainSpec) -> ClosedRange<Date> {
    Date(timeIntervalSince1970: Double(domain.startUnixMS) / 1000) ... Date(timeIntervalSince1970: Double(domain.endUnixMS) / 1000)
}

private func annotationPosition(_ raw: Int32) -> AnnotationPosition {
    switch raw {
    case 1: return .bottom
    case 2: return .leading
    case 3: return .trailing
    case 4: return .overlay
    case 5: return .topLeading
    case 6: return .topTrailing
    case 7: return .bottomLeading
    case 8: return .bottomTrailing
    case 9: return .automatic
    default: return .top
    }
}

private func labelAlignment(_ raw: Int32) -> Alignment {
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

private func foregroundColor(_ mods: [MarkModSpec]) -> Color {
    guard let mod = mods.first(where: { $0.kind == 0 }) else {
        return .accentColor
    }
    return decodeColor(mod.colorR, mod.colorG, mod.colorB, mod.colorA)
}

@MainActor
private func buildChart3DView(_ spec: Chart3DSpec) -> some View {
    var chart = AnyView(Chart3D {
        ForEach(spec.marks.indices, id: \.self) { index in
            build3DMark(spec.marks[index])
        }
        if let surface = spec.surface {
            buildSurfacePlot(surface)
        }
    })
    chart = applyScale(chart, axis: "x", domain: spec.xDomain, scaleType: spec.xScaleType)
    chart = applyScale(chart, axis: "y", domain: spec.yDomain, scaleType: spec.yScaleType)
    chart = apply3DZScale(chart, domain: spec.zDomain, scaleType: spec.zScaleType)
    if let label = spec.xAxisLabel {
        chart = applyAxisLabel(chart, axis: "x", label: label)
    }
    if let label = spec.yAxisLabel {
        chart = applyAxisLabel(chart, axis: "y", label: label)
    }
    if let label = spec.zAxisLabel {
        chart = applyZAxisLabel(chart, label: label)
    }
    if let pose = spec.pose {
        chart = AnyView(chart.chart3DPose(decodeChart3DPose(pose)))
    }
    chart = AnyView(chart.chart3DCameraProjection(decodeChart3DCameraProjection(spec.projection)))
    return chart
}

@Chart3DContentBuilder
private func build3DMark(_ spec: MarkSpec) -> some Chart3DContent {
    switch spec.kind {
    case 0:
        build3DPointMark(spec)
    case 1:
        build3DRectangleMark(spec)
    case 2:
        build3DRuleMark(spec)
    default:
        EmptyChart3DContent()
    }
}

private struct EmptyChart3DContent: Chart3DContent {
    var body: Never { fatalError() }
}

@Chart3DContentBuilder
private func build3DPointMark(_ spec: MarkSpec) -> some Chart3DContent {
    if let x = dim(spec, role: 0)?.value.toPlottable(),
       let y = dim(spec, role: 1)?.value.toPlottable(),
       let z = dim(spec, role: 2)?.value.toPlottable() {
        switch (x, y, z) {
        case (.number(let xl, let xv), .number(let yl, let yv), .number(let zl, let zv)):
            PointMark(x: .value(xl, xv), y: .value(yl, yv), z: .value(zl, zv))
                .foregroundStyle(foregroundColor(spec.mods))
        case (.integer(let xl, let xv), .number(let yl, let yv), .number(let zl, let zv)):
            PointMark(x: .value(xl, xv), y: .value(yl, yv), z: .value(zl, zv))
                .foregroundStyle(foregroundColor(spec.mods))
        case (.time(let xl, let xv, let xu), .number(let yl, let yv), .number(let zl, let zv)):
            PointMark(x: .value(xl, xv, unit: xu), y: .value(yl, yv), z: .value(zl, zv))
                .foregroundStyle(foregroundColor(spec.mods))
        default:
            EmptyChart3DContent()
        }
    } else {
        EmptyChart3DContent()
    }
}

@Chart3DContentBuilder
private func build3DRectangleMark(_ spec: MarkSpec) -> some Chart3DContent {
    if let x = dim(spec, role: 0)?.value.toPlottable(),
       let y = dim(spec, role: 1)?.value.toPlottable(),
       let z = dim(spec, role: 2)?.value.toPlottable() {
        switch (x, y, z) {
        case (.number(let xl, let xv), .number(let yl, let yv), .number(let zl, let zv)):
            RectangleMark(x: .value(xl, xv), y: .value(yl, yv), z: .value(zl, zv))
                .foregroundStyle(foregroundColor(spec.mods))
        default:
            EmptyChart3DContent()
        }
    } else {
        EmptyChart3DContent()
    }
}

@Chart3DContentBuilder
private func build3DRuleMark(_ spec: MarkSpec) -> some Chart3DContent {
    if let x = dim(spec, role: 0)?.value.toPlottable(),
       let y = dim(spec, role: 1)?.value.toPlottable(),
       let z = dim(spec, role: 2)?.value.toPlottable() {
        switch (x, y, z) {
        case (.number(let xl, let xv), .number(let yl, let yv), .number(let zl, let zv)):
            RuleMark(x: .value(xl, xv), y: .value(yl, yv), z: .value(zl, zv))
                .foregroundStyle(foregroundColor(spec.mods))
        default:
            EmptyChart3DContent()
        }
    } else {
        EmptyChart3DContent()
    }
}

private func buildSurfacePlot(_ spec: SurfacePlotSpec) -> some Chart3DContent {
    SurfacePlot(x: spec.xLabel, y: spec.yLabel, z: spec.zLabel) { x, z in
        _CHSurfaceCallback?(UInt(spec.callbackID), x, z) ?? 0
    }
}

private func applyScale(_ view: AnyView, axis: String, domain: DomainSpec?, scaleType: ScaleTypeSpec) -> AnyView {
    let ty = decodeScaleType(scaleType)
    switch (axis, domain?.kind) {
    case ("x", .some(0)):
        return AnyView(view.chartXScale(domain: domain!.minInt ... domain!.maxInt, type: ty))
    case ("x", .some(1)):
        return AnyView(view.chartXScale(domain: domain!.minNumber ... domain!.maxNumber, type: ty))
    case ("x", .some(2)):
        return AnyView(view.chartXScale(domain: dateRange(domain!), type: ty))
    case ("y", .some(0)):
        return AnyView(view.chartYScale(domain: domain!.minInt ... domain!.maxInt, type: ty))
    case ("y", .some(1)):
        return AnyView(view.chartYScale(domain: domain!.minNumber ... domain!.maxNumber, type: ty))
    case ("y", .some(2)):
        return AnyView(view.chartYScale(domain: dateRange(domain!), type: ty))
    case ("x", _):
        return AnyView(view.chartXScale(type: ty))
    case ("y", _):
        return AnyView(view.chartYScale(type: ty))
    default:
        return view
    }
}

private func apply3DZScale(_ view: AnyView, domain: DomainSpec?, scaleType: ScaleTypeSpec) -> AnyView {
    let ty = decodeScaleType(scaleType)
    switch domain?.kind {
    case .some(0):
        return AnyView(view.chartZScale(domain: domain!.minInt ... domain!.maxInt, type: ty))
    case .some(1):
        return AnyView(view.chartZScale(domain: domain!.minNumber ... domain!.maxNumber, type: ty))
    case .some(2):
        return AnyView(view.chartZScale(domain: dateRange(domain!), type: ty))
    default:
        return view
    }
}

private func applyAxisLabel(_ view: AnyView, axis: String, label: AxisLabelSpec) -> AnyView {
    let position = label.hasPosition ? annotationPosition(label.position) : .automatic
    let alignment = label.hasAlignment ? labelAlignment(label.alignment) : nil
    let spacing = label.hasSpacing ? CGFloat(label.spacing) : nil
    switch axis {
    case "x":
        return AnyView(view.chartXAxisLabel(label.text, position: position, alignment: alignment, spacing: spacing))
    case "y":
        return AnyView(view.chartYAxisLabel(label.text, position: position, alignment: alignment, spacing: spacing))
    default:
        return view
    }
}

private func applyZAxisLabel(_ view: AnyView, label: AxisLabelSpec) -> AnyView {
    let position = label.hasPosition ? annotationPosition(label.position) : .automatic
    let alignment = label.hasAlignment ? labelAlignment(label.alignment) : nil
    let spacing = label.hasSpacing ? CGFloat(label.spacing) : nil
    return AnyView(view.chartZAxisLabel(label.text, position: position, alignment: alignment, spacing: spacing))
}

private func decodeChart3DPose(_ spec: Chart3DPoseSpec) -> Charts.Chart3DPose {
    switch spec.kind {
    case 1: return .front
    case 2: return .back
    case 3: return .top
    case 4: return .bottom
    case 5: return .left
    case 6: return .right
    case 7:
        return Charts.Chart3DPose(
            azimuth: Spatial.Angle2D.degrees(spec.azimuth),
            inclination: Spatial.Angle2D.degrees(spec.inclination)
        )
    default:
        return .default
    }
}

private func decodeChart3DCameraProjection(_ raw: Int32) -> Charts.Chart3DCameraProjection {
    switch raw {
    case 1: return .orthographic
    case 2: return .perspective
    default: return .automatic
    }
}

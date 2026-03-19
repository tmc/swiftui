import Charts
import SwiftUI

private func valueKind(_ spec: ValueSpec?) -> Int32 {
    spec?.kind ?? -1
}

private func decodeMarkDimensions<Row>(_ spec: MarkDimensionSpec, keyPath: KeyPath<Row, CGFloat>) -> MarkDimensions<Row> {
    switch spec.kind {
    case 1:
        return .fixed(keyPath)
    case 2:
        return .ratio(keyPath)
    case 3:
        return .inset(keyPath)
    default:
        return .automatic
    }
}

func buildPointPlot(_ spec: MarkSpec) -> AnyChartContent {
    guard let data = spec.pointData, let first = data.first else {
        return emptyChartContent()
    }
    switch (first.x.kind, first.y.kind) {
    case (2, 2):
        struct Row: Identifiable { let id: Int; let x: Double; let y: Double }
        let rows = data.enumerated().map { Row(id: $0.offset, x: $0.element.x.number, y: $0.element.y.number) }
        return AnyChartContent(PointPlot(rows, x: .value(first.x.label, \Row.x), y: .value(first.y.label, \Row.y)))
    case (1, 2):
        struct Row: Identifiable { let id: Int; let x: Int; let y: Double }
        let rows = data.enumerated().map { Row(id: $0.offset, x: $0.element.x.integer, y: $0.element.y.number) }
        return AnyChartContent(PointPlot(rows, x: .value(first.x.label, \Row.x), y: .value(first.y.label, \Row.y)))
    case (0, 2):
        struct Row: Identifiable { let id: Int; let x: String; let y: Double }
        let rows = data.enumerated().map { Row(id: $0.offset, x: $0.element.x.category, y: $0.element.y.number) }
        return AnyChartContent(PointPlot(rows, x: .value(first.x.label, \Row.x), y: .value(first.y.label, \Row.y)))
    case (3, 2):
        struct Row: Identifiable { let id: Int; let x: Date; let y: Double }
        let rows = data.enumerated().map {
            Row(id: $0.offset, x: Date(timeIntervalSince1970: Double($0.element.x.timeUnixMS) / 1000), y: $0.element.y.number)
        }
        return AnyChartContent(PointPlot(rows, x: .value(first.x.label, \Row.x, unit: calendarUnit(first.x.timeUnit)), y: .value(first.y.label, \Row.y)))
    default:
        return emptyChartContent()
    }
}

func buildRectanglePlot(_ spec: MarkSpec) -> AnyChartContent {
    guard let data = spec.rectangleData, let first = data.first else {
        return emptyChartContent()
    }
    if valueKind(first.x) == 2 && valueKind(first.y) == 2 {
        struct Row: Identifiable {
            let id: Int
            let x: Double
            let y: Double
            let width: CGFloat
            let height: CGFloat
        }
        let rows = data.enumerated().compactMap { index, item -> Row? in
            guard let x = item.x?.number, let y = item.y?.number else { return nil }
            return Row(id: index, x: x, y: y, width: item.width.value, height: item.height.value)
        }
        return AnyChartContent(RectanglePlot(
            rows,
            x: .value(first.x!.label, \Row.x),
            y: .value(first.y!.label, \Row.y),
            width: decodeMarkDimensions(first.width, keyPath: \Row.width),
            height: decodeMarkDimensions(first.height, keyPath: \Row.height)
        ))
    }
    if valueKind(first.xStart) == 2 && valueKind(first.xEnd) == 2 && valueKind(first.y) == 0 {
        struct Row: Identifiable {
            let id: Int
            let xStart: Double
            let xEnd: Double
            let y: String
            let height: CGFloat
        }
        let rows = data.enumerated().compactMap { index, item -> Row? in
            guard let xStart = item.xStart?.number, let xEnd = item.xEnd?.number, let y = item.y?.category else { return nil }
            return Row(id: index, xStart: xStart, xEnd: xEnd, y: y, height: item.height.value)
        }
        return AnyChartContent(RectanglePlot(
            rows,
            xStart: .value(first.xStart!.label, \Row.xStart),
            xEnd: .value(first.xStart!.label, \Row.xEnd),
            y: .value(first.y!.label, \Row.y),
            height: decodeMarkDimensions(first.height, keyPath: \Row.height)
        ))
    }
    if valueKind(first.x) == 0 && valueKind(first.yStart) == 2 && valueKind(first.yEnd) == 2 {
        struct Row: Identifiable {
            let id: Int
            let x: String
            let yStart: Double
            let yEnd: Double
            let width: CGFloat
        }
        let rows = data.enumerated().compactMap { index, item -> Row? in
            guard let x = item.x?.category, let yStart = item.yStart?.number, let yEnd = item.yEnd?.number else { return nil }
            return Row(id: index, x: x, yStart: yStart, yEnd: yEnd, width: item.width.value)
        }
        return AnyChartContent(RectanglePlot(
            rows,
            x: .value(first.x!.label, \Row.x),
            yStart: .value(first.yStart!.label, \Row.yStart),
            yEnd: .value(first.yStart!.label, \Row.yEnd),
            width: decodeMarkDimensions(first.width, keyPath: \Row.width)
        ))
    }
    if valueKind(first.xStart) == 2 && valueKind(first.xEnd) == 2 && valueKind(first.yStart) == 2 && valueKind(first.yEnd) == 2 {
        struct Row: Identifiable {
            let id: Int
            let xStart: Double
            let xEnd: Double
            let yStart: Double
            let yEnd: Double
        }
        let rows = data.enumerated().compactMap { index, item -> Row? in
            guard let xStart = item.xStart?.number, let xEnd = item.xEnd?.number, let yStart = item.yStart?.number, let yEnd = item.yEnd?.number else { return nil }
            return Row(id: index, xStart: xStart, xEnd: xEnd, yStart: yStart, yEnd: yEnd)
        }
        return AnyChartContent(RectanglePlot(
            rows,
            xStart: .value(first.xStart!.label, \Row.xStart),
            xEnd: .value(first.xStart!.label, \Row.xEnd),
            yStart: .value(first.yStart!.label, \Row.yStart),
            yEnd: .value(first.yStart!.label, \Row.yEnd)
        ))
    }
    return emptyChartContent()
}

func buildRulePlot(_ spec: MarkSpec) -> AnyChartContent {
    guard let data = spec.ruleData, let first = data.first else {
        return emptyChartContent()
    }
    if valueKind(first.xStart) == 2 && valueKind(first.xEnd) == 2 && valueKind(first.y) == 0 {
        struct Row: Identifiable { let id: Int; let xStart: Double; let xEnd: Double; let y: String }
        let rows = data.enumerated().compactMap { index, item -> Row? in
            guard let xStart = item.xStart?.number, let xEnd = item.xEnd?.number, let y = item.y?.category else { return nil }
            return Row(id: index, xStart: xStart, xEnd: xEnd, y: y)
        }
        return AnyChartContent(RulePlot(rows, xStart: .value(first.xStart!.label, \Row.xStart), xEnd: .value(first.xStart!.label, \Row.xEnd), y: .value(first.y!.label, \Row.y)))
    }
    if valueKind(first.x) == 0 && valueKind(first.yStart) == 2 && valueKind(first.yEnd) == 2 {
        struct Row: Identifiable { let id: Int; let x: String; let yStart: Double; let yEnd: Double }
        let rows = data.enumerated().compactMap { index, item -> Row? in
            guard let x = item.x?.category, let yStart = item.yStart?.number, let yEnd = item.yEnd?.number else { return nil }
            return Row(id: index, x: x, yStart: yStart, yEnd: yEnd)
        }
        return AnyChartContent(RulePlot(rows, x: .value(first.x!.label, \Row.x), yStart: .value(first.yStart!.label, \Row.yStart), yEnd: .value(first.yStart!.label, \Row.yEnd)))
    }
    return emptyChartContent()
}

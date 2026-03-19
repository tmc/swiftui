import Charts
import Foundation

struct Chart3DSpec: Decodable {
    let marks: [MarkSpec]
    let surface: SurfacePlotSpec?
    let xDomain: DomainSpec?
    let yDomain: DomainSpec?
    let zDomain: DomainSpec?
    let xScaleType: ScaleTypeSpec
    let yScaleType: ScaleTypeSpec
    let zScaleType: ScaleTypeSpec
    let xAxisLabel: AxisLabelSpec?
    let yAxisLabel: AxisLabelSpec?
    let zAxisLabel: AxisLabelSpec?
    let pose: Chart3DPoseSpec?
    let projection: Int32
}

struct SurfacePlotSpec: Decodable {
    let xLabel: String
    let yLabel: String
    let zLabel: String
    let callbackID: UInt64
}

struct ValueSpec: Decodable {
    let kind: Int32
    let label: String
    let integer: Int
    let number: Double
    let timeUnixMS: Int64
    let timeUnit: Int32
}

struct DimSpec: Decodable {
    let role: Int32
    let value: ValueSpec
}

struct MarkModSpec: Decodable {
    let kind: Int32
    let colorR: Double
    let colorG: Double
    let colorB: Double
    let colorA: Double
}

struct MarkSpec: Decodable {
    let kind: Int32
    let dims: [DimSpec]
    let mods: [MarkModSpec]
}

struct DomainSpec: Decodable {
    let kind: Int32
    let minInt: Int
    let maxInt: Int
    let minNumber: Double
    let maxNumber: Double
    let startUnixMS: Int64
    let endUnixMS: Int64
}

struct ScaleTypeSpec: Decodable {
    let kind: Int32
    let value: Double
}

struct AxisLabelSpec: Decodable {
    let text: String
    let position: Int32
    let alignment: Int32
    let hasPosition: Bool
    let hasAlignment: Bool
    let hasSpacing: Bool
    let spacing: Double
}

struct Chart3DPoseSpec: Decodable {
    let kind: Int32
    let azimuth: Double
    let inclination: Double
}

enum PlottableData {
    case integer(String, Int)
    case number(String, Double)
    case time(String, Date, Calendar.Component)
}

extension ValueSpec {
    func toPlottable() -> PlottableData {
        switch kind {
        case 0:
            return .integer(label, integer)
        case 2:
            return .time(label, Date(timeIntervalSince1970: Double(timeUnixMS) / 1000), calendarUnit(timeUnit))
        default:
            return .number(label, number)
        }
    }
}

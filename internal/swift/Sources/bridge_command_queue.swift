// P7.α primitive: main-thread coalescing queue for animated-state writes.
//
// This file is hand-written, not generated. It lives below the Swift bridge
// surface as an internal optimization. Callers (the animated-state setters in
// bridge_state.gen.swift) route writes through BridgeCommandQueue; everything
// else in the bridge is oblivious to its existence.
//
// Invariants (charter §3, §4):
//   1. Animation kind grouping is preserved. Writes are partitioned by the
//      Int32 kind the setter already carries, and each partition is flushed
//      inside its own withAnimation(animationForKind(kind)) { ... } scope.
//   2. Frame boundary. The flush runs on the next main-thread turn via a
//      single DispatchQueue.main.async. We never batch across frames.
//   3. Synchronous fast path. If the caller is already on the main thread,
//      the write skips the queue entirely and is applied inline inside the
//      same withAnimation scope.
//   4. State.Set from any goroutine safety. enqueue returns immediately,
//      never blocks the caller, never fails, never returns status.
//
// Non-goals (charter §8):
//   - No async/await. DispatchQueue.main.async is the only dispatch primitive.
//   - No cross-frame batching, cancellation, or priority.
//   - No generalization beyond animated writes. apply closures are trusted
//      to perform exactly one bridged state write (setAndBump).
//
// Counter exports are read by P7.γ's SwiftUIDebug.Stats() extension.

import Foundation
import SwiftUI

/// Number of writes enqueued for coalesced main-thread flush (background
/// thread path). Incremented once per enqueue call, regardless of whether
/// the write armed a new flush or appended to an already-armed one.
nonisolated(unsafe) var _SUIBridgeCoalescedWrites: UInt64 = 0

/// Number of main-thread hops attributed to the queue. Counts both the
/// async flushes (one per armed frame) and the inline fast-path applies
/// (already on main thread, no dispatch). The ratio
/// coalescedWrites / mainHops is the P7 win metric; charter §5 exit
/// criterion #3 wants > 1.5.
nonisolated(unsafe) var _SUIBridgeMainHops: UInt64 = 0

/// Lock guarding the two counters above. Counter reads/writes are cheap
/// and contention is bounded by the already-serialized enqueue lock, so a
/// plain NSLock is fine; we just need the reads from γ's Stats() surface
/// to be non-torn.
private let _SUIBridgeCounterLock = NSLock()

private func _SUIBridgeBumpCoalesced() {
    _SUIBridgeCounterLock.lock()
    _SUIBridgeCoalescedWrites &+= 1
    _SUIBridgeCounterLock.unlock()
}

private func _SUIBridgeBumpMainHops() {
    _SUIBridgeCounterLock.lock()
    _SUIBridgeMainHops &+= 1
    _SUIBridgeCounterLock.unlock()
}

/// Snapshot of the P7 counters. Read by P7.γ's SwiftUIDebug.Stats()
/// extension. Tuple fields are plain UInt64; the caller derives the ratio.
public func _SUIBridgeQueueCounters() -> (coalescedWrites: UInt64, mainHops: UInt64) {
    _SUIBridgeCounterLock.lock()
    defer { _SUIBridgeCounterLock.unlock() }
    return (_SUIBridgeCoalescedWrites, _SUIBridgeMainHops)
}

/// Reset the counters to zero. Exposed for benchmark harnesses so
/// individual bench iterations can isolate their own write counts.
public func _SUIBridgeResetQueueCounters() {
    _SUIBridgeCounterLock.lock()
    _SUIBridgeCoalescedWrites = 0
    _SUIBridgeMainHops = 0
    _SUIBridgeCounterLock.unlock()
}

// MARK: - C-ABI counter accessors

// _SUIBridgeQueueCounters above returns a Swift tuple, which is not
// C-ABI. purego can dlsym but cannot reconstruct a tuple on the Go side.
// The following @_cdecl trio exposes scalar getters plus a reset, each
// returning / taking plain UInt64, which purego can call directly.
// P7.γ's Go-side benchmarks read these.

@_cdecl("SUIBridgeGetCoalescedWrites")
public func SUIBridgeGetCoalescedWrites() -> UInt64 {
    _SUIBridgeCounterLock.lock()
    defer { _SUIBridgeCounterLock.unlock() }
    return _SUIBridgeCoalescedWrites
}

@_cdecl("SUIBridgeGetMainHops")
public func SUIBridgeGetMainHops() -> UInt64 {
    _SUIBridgeCounterLock.lock()
    defer { _SUIBridgeCounterLock.unlock() }
    return _SUIBridgeMainHops
}

@_cdecl("SUIBridgeResetQueueCounters")
public func SUIBridgeResetQueueCountersCABI() {
    _SUIBridgeCounterLock.lock()
    _SUIBridgeCoalescedWrites = 0
    _SUIBridgeMainHops = 0
    _SUIBridgeCounterLock.unlock()
}

/// A pending state write captured on a background thread for main-thread
/// replay. The apply closure must:
///   - call exactly one setAndBump (or equivalent bridged state write),
///   - not call back into BridgeCommandQueue,
///   - not block,
///   - not read a gen counter before the setAndBump it itself performs.
/// The queue does not validate these; β's integration is expected to honor
/// them.
private struct PendingStateWrite {
    let apply: () -> Void
}

private struct AnimationPartitionKey: Hashable {
    let kind: Int32
    let durationBits: UInt64

    init(kind: Int32, duration: Double) {
        let normalized: Double
        if duration.isFinite && duration > 0 {
            normalized = duration
        } else {
            normalized = 0
        }
        self.kind = kind
        self.durationBits = normalized.bitPattern
    }

    var duration: Double {
        Double(bitPattern: durationBits)
    }
}

/// Main-thread coalescing queue for animated bridged-state writes.
///
/// Call sites (β owns integration in bridge_state.gen.swift):
///   - `BridgeCommandQueue.shared.applyInline(kind:)` — already on main
///     thread; skip the queue.
///   - `BridgeCommandQueue.shared.enqueue(kind:)` — off main; append to
///     the kind partition and arm a flush if not already armed.
///
/// A single queue instance is shared process-wide via `shared`. The queue
/// is process-scoped because animation kinds are a process-scoped concept
/// and coalescing is only interesting at main-thread granularity.
final class BridgeCommandQueue: @unchecked Sendable {
    /// Shared instance. Use `BridgeCommandQueue.shared` — do not construct
    /// BridgeCommandQueue directly.
    static let shared = BridgeCommandQueue()

    /// Per-animation-kind partitions. A write with kind K lands in
    /// pending[K], and flush wraps each partition in a single
    /// withAnimation(suiAnimationForKind(K.kind, duration: K.duration)) {
    /// ... } scope so grouping is preserved across coalesced writes of the
    /// same animation key.
    private var pending: [AnimationPartitionKey: [PendingStateWrite]] = [:]

    /// armed == true iff a main-thread flush is scheduled (via
    /// DispatchQueue.main.async) but has not yet fired. The flush clears
    /// armed under the lock before iterating pending, which means a write
    /// that arrives during the flush observes armed == false and re-arms
    /// correctly. That is the load-bearing race-free pattern for
    /// frame-boundary coalescing.
    private var armed: Bool = false

    /// Guards pending + armed. Short critical section (map append + bool
    /// swap); contention is bounded by the fan-out of goroutines calling
    /// Set(animated) and is tolerable. Charter §7 #3 anticipates this;
    /// if contention bites we revisit with atomics.
    private let lock = NSLock()

    private init() {}

    /// Enqueue a write from any thread. If the queue is not already armed,
    /// arms it by scheduling a single main-thread flush. Returns
    /// immediately; never blocks the caller.
    ///
    /// The apply closure runs on the main thread inside a
    /// withAnimation(animationForKind(kind)) scope when the flush fires.
    /// Preconditions on apply: see PendingStateWrite.
    func enqueue(kind: Int32, _ apply: @escaping () -> Void) {
        enqueue(kind: kind, duration: 0, apply)
    }

    /// Enqueue a write with an explicit duration-partitioned animation key.
    func enqueue(kind: Int32, duration: Double, _ apply: @escaping () -> Void) {
        _SUIBridgeBumpCoalesced()

        lock.lock()
        let key = AnimationPartitionKey(kind: kind, duration: duration)
        pending[key, default: []].append(PendingStateWrite(apply: apply))
        let shouldArm = !armed
        if shouldArm {
            armed = true
        }
        lock.unlock()

        if shouldArm {
            DispatchQueue.main.async { [weak self] in
                self?.flush()
            }
        }
    }

    /// Fast-path apply for callers that are already on the main thread.
    /// The write runs synchronously inside a
    /// withAnimation(animationForKind(kind)) scope; no dispatch, no queue.
    ///
    /// MUST be called on the main thread. The caller establishes that via
    /// Thread.isMainThread before invoking. We don't assert @MainActor on
    /// the method because the callers are @_cdecl entry points that are
    /// nonisolated; the charter's synchronous-fast-path invariant (§3 #3)
    /// requires this direct-call shape.
    func applyInline(kind: Int32, _ apply: () -> Void) {
        applyInline(kind: kind, duration: 0, apply)
    }

    /// Fast-path apply with an explicit duration-aware animation key.
    func applyInline(kind: Int32, duration: Double, _ apply: () -> Void) {
        _SUIBridgeBumpMainHops()
        withAnimation(suiAnimationForKind(kind, duration: duration)) {
            apply()
        }
    }

    /// Main-thread flush. Called exactly once per armed cycle.
    ///
    /// Clears armed under the lock before iterating. This means a write
    /// that arrives mid-flush (same main-thread turn is not possible, but
    /// a background-thread write racing with us is) observes armed ==
    /// false and arms a fresh flush for the next turn. Writes that
    /// arrived before we cleared pending are included in this flush;
    /// writes that arrive after are in the next flush.
    private func flush() {
        _SUIBridgeBumpMainHops()

        lock.lock()
        let partitions = pending
        pending.removeAll(keepingCapacity: true)
        armed = false
        lock.unlock()

        // Order across kinds is map-iteration order (unstable across Swift
        // versions, but all writes of the same kind preserve insertion
        // order within the partition, which is the load-bearing guarantee
        // for animation grouping). Cross-kind ordering is undefined today;
        // if a caller needs stable cross-kind order, they must use a
        // single kind.
        for (key, writes) in partitions {
            withAnimation(suiAnimationForKind(key.kind, duration: key.duration)) {
                for write in writes {
                    write.apply()
                }
            }
        }
    }

    // MARK: - Test hooks
    //
    // Read-only surface exposed for unit/behavioral tests. Not part of the
    // shipping API; callers must not rely on these outside tests.

    /// Number of writes currently pending (not yet flushed), summed across
    /// all kind partitions. Racy by construction; for tests only.
    func _testPendingCount() -> Int {
        lock.lock()
        defer { lock.unlock() }
        return pending.values.reduce(0) { $0 + $1.count }
    }

    /// Whether a main-thread flush is currently armed. Racy; for tests only.
    func _testIsArmed() -> Bool {
        lock.lock()
        defer { lock.unlock() }
        return armed
    }
}

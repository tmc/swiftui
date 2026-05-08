package swiftui

import (
	"sync"

	"github.com/ebitengine/purego"
)

// SwiftUIDebugStats is a snapshot of the bridge's internal debug counters.
// Runtime surface.
//
// The zero value is valid and reports zeroed counters (useful when the
// underlying Swift-side queue primitive is not loaded, e.g. in unit tests
// that stub out the dylib).
//
// Counters are process-global and monotonically increasing; read them
// before and after a workload and take the delta for per-workload metrics.
//
// Fields map 1:1 to the Swift-side counters in bridge_command_queue.swift:
//   - BridgeCoalescedWrites bumps once per enqueue on the background-thread
//     path. It does NOT bump on the synchronous inline path.
//   - BridgeMainHops bumps once per main-thread apply. It covers BOTH the
//     inline fast-path (already on main thread, no dispatch) AND the
//     dispatched flush (one per armed frame). "mainHops" is the correct
//     label because inline applies count as hops too.
type SwiftUIDebugStats struct {
	// BridgeCoalescedWrites is the number of animated-state writes that
	// have been enqueued for coalesced main-thread flush since process
	// start. Each off-main SetAnimatedWith call increments this by one.
	BridgeCoalescedWrites uint64

	// BridgeMainHops is the number of main-thread applies since process
	// start. Covers both applyInline (sync fast-path) and flush (async
	// dispatched) — both are "hops" in the sense that state writes
	// materialize on the main thread.
	BridgeMainHops uint64
}

// BridgeCoalesceRatio returns the derived coalescing ratio:
//
//	CoalesceRatio = BridgeCoalescedWrites / BridgeMainHops
//
// The P7 charter §5 exit criterion #3 wants this > 1.5 on the animated
// state mutation stress benchmark. A ratio of 1.0 means every coalesced
// write took its own main hop (no coalescing happened); a ratio of N
// means N writes shared one hop on average. Returns 0 when MainHops is
// zero (no bridge activity yet).
func (s SwiftUIDebugStats) BridgeCoalesceRatio() float64 {
	if s.BridgeMainHops == 0 {
		return 0
	}
	return float64(s.BridgeCoalescedWrites) / float64(s.BridgeMainHops)
}

// SwiftUIDebug exposes read-only diagnostics for the Swift bridge runtime.
// Runtime surface.
//
// Consumers use it to assert on coalescing behavior, catch leaks, and
// collect perf counters. The struct has no fields; it namespaces method
// receivers so the public surface reads as SwiftUIDebug.Stats() at call
// sites.
type SwiftUIDebug struct{}

// Stats returns a snapshot of the bridge debug counters. Safe to call
// from any goroutine. Returns the zero value if the Swift-side queue
// primitive has not been loaded (e.g., when the dylib is absent or an
// older dylib without the P7 counter exports is in use).
func (SwiftUIDebug) Stats() SwiftUIDebugStats {
	ensureBridgeDebugCountersLoaded()
	var out SwiftUIDebugStats
	if _bridgeGetCoalescedWritesFn != nil {
		out.BridgeCoalescedWrites = _bridgeGetCoalescedWritesFn()
	}
	if _bridgeGetMainHopsFn != nil {
		out.BridgeMainHops = _bridgeGetMainHopsFn()
	}
	return out
}

// ResetStats clears the bridge debug counters to zero. Exposed for
// benchmark harnesses so individual bench iterations can isolate their
// own counter deltas. Safe to call from any goroutine; serializes
// against Stats via the Swift-side NSLock. No-op when the Swift-side
// queue primitive has not been loaded.
func (SwiftUIDebug) ResetStats() {
	ensureBridgeDebugCountersLoaded()
	if _bridgeResetQueueCountersFn != nil {
		_bridgeResetQueueCountersFn()
	}
}

// Counter-accessor function pointers resolved once against the embedded
// SwiftUIBridge dylib. The three symbols are @_cdecl exports from
// bridge_command_queue.swift, landed as a follow-up to α:
//
//	uint64_t SUIBridgeGetCoalescedWrites(void);
//	uint64_t SUIBridgeGetMainHops(void);
//	void     SUIBridgeResetQueueCounters(void);
//
// Resolution is lazy + idempotent; failures leave the fn pointers nil
// and the public Stats/ResetStats become no-ops that return the zero
// value.
var (
	_bridgeDebugCountersOnce    sync.Once
	_bridgeGetCoalescedWritesFn func() uint64
	_bridgeGetMainHopsFn        func() uint64
	_bridgeResetQueueCountersFn func()
)

func ensureBridgeDebugCountersLoaded() {
	_bridgeDebugCountersOnce.Do(func() {
		if libHandle == 0 {
			return
		}
		if _, err := purego.Dlsym(libHandle, "SUIBridgeGetCoalescedWrites"); err == nil {
			var fn func() uint64
			purego.RegisterLibFunc(&fn, libHandle, "SUIBridgeGetCoalescedWrites")
			_bridgeGetCoalescedWritesFn = fn
		}
		if _, err := purego.Dlsym(libHandle, "SUIBridgeGetMainHops"); err == nil {
			var fn func() uint64
			purego.RegisterLibFunc(&fn, libHandle, "SUIBridgeGetMainHops")
			_bridgeGetMainHopsFn = fn
		}
		if _, err := purego.Dlsym(libHandle, "SUIBridgeResetQueueCounters"); err == nil {
			var fn func()
			purego.RegisterLibFunc(&fn, libHandle, "SUIBridgeResetQueueCounters")
			_bridgeResetQueueCountersFn = fn
		}
	})
}

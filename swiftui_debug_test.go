package swiftui

import "testing"

// TestDebugStatsCounters exercises the Swift-ABI bridge from
// SwiftUIDebug.Stats()/ResetStats() to α's _SUIBridgeQueueCounters /
// _SUIBridgeResetQueueCounters. The test asserts that after a single
// SetAnimatedWith call from a non-main goroutine (the default in Go's
// test harness), the coalescedWrites counter increments by one.
//
// mainHops is NOT asserted here because the Go test harness does not
// drive the main-thread runloop, so the Swift-side flush cannot fire
// within the test window. The representative-bench docstring covers
// this caveat; see BenchmarkCoalesceRatio.
//
// Skips when the bridge dylib is not loaded (parity with animated_state
// _test.go's TestAnimatedStateSetConcurrent).
func TestDebugStatsCounters(t *testing.T) {
	if loadErr != nil {
		t.Skipf("bridge dylib not loaded: %v", loadErr)
	}

	var dbg SwiftUIDebug
	dbg.ResetStats()
	before := dbg.Stats()

	s := NewIntState(0)
	t.Cleanup(s.Release)
	s.SetAnimatedWith(42, AnimationEaseInOut)

	after := dbg.Stats()
	got := after.BridgeCoalescedWrites - before.BridgeCoalescedWrites
	if got != 1 {
		t.Fatalf("BridgeCoalescedWrites delta = %d, want 1 (after one off-main SetAnimatedWith)", got)
	}
}

// TestDebugStatsRatioZeroWhenNoHops verifies the derived ratio returns
// zero rather than a NaN/Inf when MainHops is zero. The charter §5 exit
// #3 uses this ratio as a gate; the API must not panic or yield a
// degenerate value when the runloop has not yet drained the queue.
func TestDebugStatsRatioZeroWhenNoHops(t *testing.T) {
	s := SwiftUIDebugStats{BridgeCoalescedWrites: 10, BridgeMainHops: 0}
	if r := s.BridgeCoalesceRatio(); r != 0 {
		t.Fatalf("BridgeCoalesceRatio() with zero hops = %v, want 0", r)
	}
}

// TestDebugStatsRatioComputation asserts the ratio arithmetic matches
// the charter definition: coalescedWrites / mainHops.
func TestDebugStatsRatioComputation(t *testing.T) {
	s := SwiftUIDebugStats{BridgeCoalescedWrites: 15, BridgeMainHops: 3}
	if r := s.BridgeCoalesceRatio(); r != 5.0 {
		t.Fatalf("BridgeCoalesceRatio() = %v, want 5.0", r)
	}
}

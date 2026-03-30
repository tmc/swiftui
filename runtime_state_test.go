package swiftui

import (
	"math"
	"testing"
	"time"
)

func TestDateSelectionState(t *testing.T) {
	first := time.Date(2026, 3, 29, 13, 20, 42, 0, time.FixedZone("PDT", -7*60*60))
	sameDay := time.Date(2026, 3, 29, 1, 5, 0, 0, time.UTC)
	nextDay := time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)

	state := NewDateSelectionState(first, sameDay, nextDay, first)

	got := state.Get()
	want := []time.Time{canonicalDay(first), canonicalDay(nextDay)}
	if len(got) != len(want) {
		t.Fatalf("len(Get()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("Get()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if !state.Has(first) || !state.Has(sameDay) || !state.Has(nextDay) {
		t.Fatalf("Has did not recognize selected dates")
	}
	if state.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", state.Count())
	}
	if state.Revision() == 0 {
		t.Fatalf("Revision() = 0, want a non-zero initial revision after construction")
	}

	prevRevision := state.Revision()
	state.Add(nextDay)
	if state.Revision() != prevRevision {
		t.Fatalf("revision changed on no-op add: got %d want %d", state.Revision(), prevRevision)
	}

	state.Remove(first)
	if state.Count() != 1 {
		t.Fatalf("Count() after remove = %d, want 1", state.Count())
	}
	if state.Has(first) {
		t.Fatal("removed date still present")
	}

	if revState := state.RevisionState(); revState != nil {
		if got := revState.Get(); got != state.Revision() {
			t.Fatalf("RevisionState.Get() = %d, want %d", got, state.Revision())
		}
	}
	if countState := state.CountState(); countState != nil {
		if got := countState.Get(); got != state.Count() {
			t.Fatalf("CountState.Get() = %d, want %d", got, state.Count())
		}
	}

	state.Clear()
	if state.Count() != 0 {
		t.Fatalf("Count() after clear = %d, want 0", state.Count())
	}
	if len(state.Get()) != 0 {
		t.Fatalf("Get() after clear returned %d items, want 0", len(state.Get()))
	}

	state.Release()
	if state.RevisionState() != nil {
		t.Fatal("RevisionState() not cleared after Release")
	}
	if state.CountState() != nil {
		t.Fatal("CountState() not cleared after Release")
	}
}

func TestDateRangeState(t *testing.T) {
	start := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 18, 15, 30, 0, 0, time.UTC)

	state := NewDateRangeState(start, end, true)

	gotStart, gotEnd, ok := state.Get()
	if !ok {
		t.Fatal("Get() reported invalid range")
	}
	if !gotStart.Equal(end) || !gotEnd.Equal(start) {
		t.Fatalf("Get() = (%v, %v), want (%v, %v)", gotStart, gotEnd, end, start)
	}

	state.Set(start.Add(48*time.Hour), end.Add(72*time.Hour))
	gotStart, gotEnd, ok = state.Get()
	if !ok {
		t.Fatal("Get() after Set reported invalid range")
	}
	if gotStart.After(gotEnd) {
		t.Fatalf("range not normalized: start=%v end=%v", gotStart, gotEnd)
	}

	if startState := state.StartState(); startState != nil {
		if got := time.Unix(int64(startState.Get()), 0).UTC(); !got.Equal(gotStart.Truncate(time.Second).UTC()) {
			t.Fatalf("StartState.Get() = %v, want %v", got, gotStart)
		}
	}
	if endState := state.EndState(); endState != nil {
		if got := time.Unix(int64(endState.Get()), 0).UTC(); !got.Equal(gotEnd.Truncate(time.Second).UTC()) {
			t.Fatalf("EndState.Get() = %v, want %v", got, gotEnd)
		}
	}
	if validState := state.ValidState(); validState != nil {
		if !validState.Get() {
			t.Fatal("ValidState.Get() = false, want true")
		}
	}
	if revState := state.RevisionState(); revState != nil {
		if got := revState.Get(); got == 0 {
			t.Fatal("RevisionState.Get() = 0, want non-zero after mutation")
		}
	}

	state.Clear()
	gotStart, gotEnd, ok = state.Get()
	if ok {
		t.Fatal("Get() after Clear reported valid range")
	}
	if !gotStart.IsZero() || !gotEnd.IsZero() {
		t.Fatalf("Get() after Clear returned non-zero times: %v %v", gotStart, gotEnd)
	}
	if validState := state.ValidState(); validState != nil && validState.Get() {
		t.Fatal("ValidState.Get() = true after Clear")
	}

	state.Release()
	if state.StartState() != nil || state.EndState() != nil || state.ValidState() != nil || state.RevisionState() != nil {
		t.Fatal("primitive accessors not cleared after Release")
	}
}

func TestTimerState(t *testing.T) {
	state := NewTimerState(90*time.Second, 120*time.Second, true)

	if got := state.Total(); got != 90*time.Second {
		t.Fatalf("Total() = %v, want 90s", got)
	}
	if got := state.Remaining(); got != 90*time.Second {
		t.Fatalf("Remaining() = %v, want 90s", got)
	}
	if !state.Running() {
		t.Fatal("Running() = false, want true")
	}
	if got := state.Progress(); math.Abs(got-0) > 1e-9 {
		t.Fatalf("Progress() = %v, want 0", got)
	}

	if totalState := state.TotalState(); totalState != nil && totalState.Get() != 90 {
		t.Fatalf("TotalState.Get() = %d, want 90", totalState.Get())
	}
	if remainingState := state.RemainingState(); remainingState != nil && remainingState.Get() != 90 {
		t.Fatalf("RemainingState.Get() = %d, want 90", remainingState.Get())
	}
	if runningState := state.RunningState(); runningState != nil && !runningState.Get() {
		t.Fatal("RunningState.Get() = false, want true")
	}

	state.SetRemaining(45 * time.Second)
	if got := state.Remaining(); got != 45*time.Second {
		t.Fatalf("Remaining() after SetRemaining = %v, want 45s", got)
	}
	if got := state.Progress(); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("Progress() after SetRemaining = %v, want 0.5", got)
	}

	state.SetRemaining(3 * time.Minute)
	if got := state.Remaining(); got != 90*time.Second {
		t.Fatalf("Remaining() after clamp = %v, want 90s", got)
	}

	state.SetRunning(false)
	if state.Running() {
		t.Fatal("Running() after SetRunning(false) = true, want false")
	}

	state.SetRemaining(2 * time.Second)
	state.SetRunning(true)
	state.Tick()
	if got := state.Remaining(); got != 1*time.Second {
		t.Fatalf("Remaining() after Tick = %v, want 1s", got)
	}
	state.Tick()
	if got := state.Remaining(); got != 0 {
		t.Fatalf("Remaining() after final Tick = %v, want 0", got)
	}
	if state.Running() {
		t.Fatal("Running() after final Tick = true, want false")
	}

	state.SetRemaining(90 * time.Second)
	state.SetTotal(2 * time.Minute)
	if got := state.Total(); got != 2*time.Minute {
		t.Fatalf("Total() after SetTotal = %v, want 2m", got)
	}
	if got := state.Remaining(); got != 90*time.Second {
		t.Fatalf("Remaining() after SetTotal = %v, want 90s", got)
	}

	state.Reset()
	if got := state.Remaining(); got != 2*time.Minute {
		t.Fatalf("Remaining() after Reset = %v, want 2m", got)
	}
	if state.Running() {
		t.Fatal("Running() after Reset = true, want false")
	}
	if got := state.Progress(); math.Abs(got-0) > 1e-9 {
		t.Fatalf("Progress() after Reset = %v, want 0", got)
	}

	if revState := state.RevisionState(); revState != nil && revState.Get() == 0 {
		t.Fatal("RevisionState.Get() = 0 after timer mutations")
	}
	if progressState := state.ProgressState(); progressState != nil {
		if got := progressState.Get(); math.Abs(got-state.Progress()) > 1e-9 {
			t.Fatalf("ProgressState.Get() = %v, want %v", got, state.Progress())
		}
	}

	state.Release()
	if state.TotalState() != nil || state.RemainingState() != nil || state.RunningState() != nil || state.ProgressState() != nil || state.RevisionState() != nil {
		t.Fatal("primitive accessors not cleared after Release")
	}
}

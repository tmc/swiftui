package swiftui

import (
	"sort"
	"sync"
	"time"
)

type dateKey struct {
	year  int
	month time.Month
	day   int
}

func bridgeReady() bool {
	return loadErr == nil && libHandle != 0
}

func newIntStateIfReady(initial int) *IntState {
	if !bridgeReady() {
		return nil
	}
	return NewIntState(initial)
}

func newBoolStateIfReady(initial bool) *BoolState {
	if !bridgeReady() {
		return nil
	}
	return NewBoolState(initial)
}

func newFloatStateIfReady(initial float64) *FloatState {
	if !bridgeReady() {
		return nil
	}
	return NewFloatState(initial)
}

func newDateStateIfReady(initial time.Time) *DateState {
	if !bridgeReady() {
		return nil
	}
	return NewDateState(epochSeconds(initial))
}

func updateIntState(state *IntState, v int) {
	if state != nil {
		state.Set(v)
	}
}

func updateBoolState(state *BoolState, v bool) {
	if state != nil {
		state.Set(v)
	}
}

func updateFloatState(state *FloatState, v float64) {
	if state != nil {
		state.Set(v)
	}
}

func updateDateState(state *DateState, v time.Time) {
	if state != nil {
		state.Set(epochSeconds(v))
	}
}

func epochSeconds(t time.Time) float64 {
	return float64(t.UnixNano()) / 1e9
}

func canonicalDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func dayKey(t time.Time) dateKey {
	year, month, day := t.Date()
	return dateKey{year: year, month: month, day: day}
}

func sortTimes(src []time.Time) {
	sort.Slice(src, func(i, j int) bool {
		if src[i].Before(src[j]) {
			return true
		}
		if src[j].Before(src[i]) {
			return false
		}
		return false
	})
}

// DateSelectionState owns a set of selected calendar days.
//
// The state is normalized to calendar-day precision and exposes a revision
// counter plus count accessor states for use with DynamicView when the bridge
// is available.
type DateSelectionState struct {
	mu sync.Mutex

	dates map[dateKey]time.Time

	revision int

	revisionState *IntState
	countState    *IntState
}

// NewDateSelectionState creates a new date-selection state.
func NewDateSelectionState(initial ...time.Time) *DateSelectionState {
	s := &DateSelectionState{}
	s.set(initial)
	if bridgeReady() {
		s.revisionState = newIntStateIfReady(s.revision)
		s.countState = newIntStateIfReady(len(s.dates))
	}
	return s
}

// Get returns the selected dates in ascending order.
func (s *DateSelectionState) Get() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked()
}

// Set replaces the selected dates.
func (s *DateSelectionState) Set(dates []time.Time) {
	s.set(dates)
}

// Add selects a single date.
func (s *DateSelectionState) Add(date time.Time) {
	s.mu.Lock()
	if s.dates == nil {
		s.dates = make(map[dateKey]time.Time)
	}
	key := dayKey(date)
	canon := canonicalDay(date)
	if existing, ok := s.dates[key]; ok && existing.Equal(canon) {
		s.mu.Unlock()
		return
	}
	s.dates[key] = canon
	s.bumpLocked()
	revision := s.revision
	count := len(s.dates)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

// Remove clears one selected date.
func (s *DateSelectionState) Remove(date time.Time) {
	s.mu.Lock()
	if len(s.dates) == 0 {
		s.mu.Unlock()
		return
	}
	key := dayKey(date)
	if _, ok := s.dates[key]; !ok {
		s.mu.Unlock()
		return
	}
	delete(s.dates, key)
	s.bumpLocked()
	revision := s.revision
	count := len(s.dates)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

// Clear removes all selected dates.
func (s *DateSelectionState) Clear() {
	s.mu.Lock()
	if len(s.dates) == 0 {
		s.mu.Unlock()
		return
	}
	s.dates = make(map[dateKey]time.Time)
	s.bumpLocked()
	revision := s.revision
	count := len(s.dates)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

// Has reports whether the given date is selected.
func (s *DateSelectionState) Has(date time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.dates[dayKey(date)]
	return ok
}

// Count returns the number of selected dates.
func (s *DateSelectionState) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.dates)
}

// Revision returns the current revision counter.
func (s *DateSelectionState) Revision() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision
}

// RevisionState returns the owned revision primitive state when the bridge is available.
func (s *DateSelectionState) RevisionState() *IntState { return s.revisionState }

// CountState returns the owned count primitive state when the bridge is available.
func (s *DateSelectionState) CountState() *IntState { return s.countState }

// Release releases any owned primitive state handles.
func (s *DateSelectionState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	revisionState := s.revisionState
	countState := s.countState
	s.revisionState = nil
	s.countState = nil
	s.dates = nil
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
	if countState != nil {
		countState.Release()
	}
}

func (s *DateSelectionState) set(dates []time.Time) {
	next := make(map[dateKey]time.Time, len(dates))
	for _, date := range dates {
		key := dayKey(date)
		next[key] = canonicalDay(date)
	}

	s.mu.Lock()
	if sameDateSet(s.dates, next) {
		s.mu.Unlock()
		return
	}
	s.dates = next
	s.bumpLocked()
	revision := s.revision
	count := len(s.dates)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

func (s *DateSelectionState) getLocked() []time.Time {
	if len(s.dates) == 0 {
		return nil
	}
	out := make([]time.Time, 0, len(s.dates))
	for _, date := range s.dates {
		out = append(out, date)
	}
	sortTimes(out)
	return out
}

func (s *DateSelectionState) bumpLocked() {
	s.revision++
}

func sameDateSet(a, b map[dateKey]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for key, got := range a {
		want, ok := b[key]
		if !ok || !got.Equal(want) {
			return false
		}
	}
	return true
}

// DateRangeState owns a single date interval.
//
// The state preserves the current range plus a revision counter and exposes
// optional primitive states for the start, end, valid, and revision values.
type DateRangeState struct {
	mu sync.Mutex

	start time.Time
	end   time.Time
	valid bool

	revision int

	startState    *DateState
	endState      *DateState
	validState    *BoolState
	revisionState *IntState
}

// NewDateRangeState creates a new date-range state.
func NewDateRangeState(start, end time.Time, ok bool) *DateRangeState {
	start, end = normalizeRange(start, end)
	s := &DateRangeState{
		start: start,
		end:   end,
		valid: ok,
	}
	if bridgeReady() {
		s.startState = newDateStateIfReady(start)
		s.endState = newDateStateIfReady(end)
		s.validState = newBoolStateIfReady(ok)
		s.revisionState = newIntStateIfReady(0)
	}
	return s
}

// Get returns the current range and whether it is valid.
func (s *DateRangeState) Get() (time.Time, time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.valid {
		return time.Time{}, time.Time{}, false
	}
	return s.start, s.end, true
}

// Set updates the range and marks it valid.
func (s *DateRangeState) Set(start, end time.Time) {
	start, end = normalizeRange(start, end)
	s.mu.Lock()
	if s.valid && s.start.Equal(start) && s.end.Equal(end) {
		s.mu.Unlock()
		return
	}
	s.start = start
	s.end = end
	s.valid = true
	s.bumpLocked()
	revision := s.revision
	startState := s.startState
	endState := s.endState
	validState := s.validState
	revisionState := s.revisionState
	s.mu.Unlock()
	updateDateState(startState, start)
	updateDateState(endState, end)
	updateBoolState(validState, true)
	updateIntState(revisionState, revision)
}

// Clear invalidates the range.
func (s *DateRangeState) Clear() {
	s.mu.Lock()
	if !s.valid && s.start.IsZero() && s.end.IsZero() {
		s.mu.Unlock()
		return
	}
	s.start = time.Time{}
	s.end = time.Time{}
	s.valid = false
	s.bumpLocked()
	revision := s.revision
	startState := s.startState
	endState := s.endState
	validState := s.validState
	revisionState := s.revisionState
	s.mu.Unlock()
	updateDateState(startState, time.Time{})
	updateDateState(endState, time.Time{})
	updateBoolState(validState, false)
	updateIntState(revisionState, revision)
}

// StartState returns the owned start-date primitive state when the bridge is available.
func (s *DateRangeState) StartState() *DateState { return s.startState }

// EndState returns the owned end-date primitive state when the bridge is available.
func (s *DateRangeState) EndState() *DateState { return s.endState }

// ValidState returns the owned validity primitive state when the bridge is available.
func (s *DateRangeState) ValidState() *BoolState { return s.validState }

// RevisionState returns the owned revision primitive state when the bridge is available.
func (s *DateRangeState) RevisionState() *IntState { return s.revisionState }

// Release releases any owned primitive state handles.
func (s *DateRangeState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	startState := s.startState
	endState := s.endState
	validState := s.validState
	revisionState := s.revisionState
	s.startState = nil
	s.endState = nil
	s.validState = nil
	s.revisionState = nil
	s.mu.Unlock()
	if startState != nil {
		startState.Release()
	}
	if endState != nil {
		endState.Release()
	}
	if validState != nil {
		validState.Release()
	}
	if revisionState != nil {
		revisionState.Release()
	}
}

func (s *DateRangeState) bumpLocked() {
	s.revision++
}

func normalizeRange(start, end time.Time) (time.Time, time.Time) {
	if end.Before(start) {
		start, end = end, start
	}
	return start, end
}

// TimerState owns the state required for a countdown or stopwatch surface.
//
// It stores the total duration, remaining duration, running flag, progress,
// and a revision counter. When the bridge is available it also exposes the
// current values as primitive state handles so existing DynamicView patterns
// can observe it.
type TimerState struct {
	mu sync.Mutex

	total     time.Duration
	remaining time.Duration
	running   bool
	progress  float64

	revision int

	totalState     *IntState
	remainingState *IntState
	runningState   *BoolState
	progressState  *FloatState
	revisionState  *IntState
}

// NewTimerState creates a new timer state.
func NewTimerState(total, remaining time.Duration, running bool) *TimerState {
	total, remaining, progress := normalizeTimer(total, remaining)
	s := &TimerState{
		total:     total,
		remaining: remaining,
		running:   running,
		progress:  progress,
	}
	if bridgeReady() {
		s.totalState = newIntStateIfReady(durationSecondsInt(total))
		s.remainingState = newIntStateIfReady(durationSecondsInt(remaining))
		s.runningState = newBoolStateIfReady(running)
		s.progressState = newFloatStateIfReady(progress)
		s.revisionState = newIntStateIfReady(0)
	}
	return s
}

// Total returns the configured total duration.
func (s *TimerState) Total() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}

// Remaining returns the current remaining duration.
func (s *TimerState) Remaining() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.remaining
}

// Running reports whether the timer is running.
func (s *TimerState) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Progress returns the current normalized progress.
func (s *TimerState) Progress() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// SetTotal updates the timer duration.
func (s *TimerState) SetTotal(total time.Duration) {
	s.mu.Lock()
	nextTotal, nextRemaining, nextProgress := normalizeTimer(total, s.remaining)
	if s.total == nextTotal && s.remaining == nextRemaining && s.progress == nextProgress {
		s.mu.Unlock()
		return
	}
	s.total = nextTotal
	s.remaining = nextRemaining
	s.progress = nextProgress
	s.bumpLocked()
	revision := s.revision
	totalState := s.totalState
	remainingState := s.remainingState
	progressState := s.progressState
	revisionState := s.revisionState
	totalCurrent := s.total
	remainingCurrent := s.remaining
	progressCurrent := s.progress
	s.mu.Unlock()
	updateIntState(totalState, durationSecondsInt(totalCurrent))
	updateIntState(remainingState, durationSecondsInt(remainingCurrent))
	updateFloatState(progressState, progressCurrent)
	updateIntState(revisionState, revision)
}

// SetRemaining updates the remaining duration.
func (s *TimerState) SetRemaining(v time.Duration) {
	s.mu.Lock()
	nextRemaining := clampRemaining(v, s.total)
	nextProgress := progressForTimer(s.total, nextRemaining)
	if s.remaining == nextRemaining && s.progress == nextProgress {
		s.mu.Unlock()
		return
	}
	s.remaining = nextRemaining
	s.progress = nextProgress
	s.bumpLocked()
	revision := s.revision
	remainingState := s.remainingState
	progressState := s.progressState
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(remainingState, durationSecondsInt(nextRemaining))
	updateFloatState(progressState, nextProgress)
	updateIntState(revisionState, revision)
}

// SetRunning updates the running flag.
func (s *TimerState) SetRunning(v bool) {
	s.mu.Lock()
	if s.running == v {
		s.mu.Unlock()
		return
	}
	s.running = v
	s.bumpLocked()
	revision := s.revision
	runningState := s.runningState
	revisionState := s.revisionState
	s.mu.Unlock()
	updateBoolState(runningState, v)
	updateIntState(revisionState, revision)
}

// Tick advances the timer by one second when it is running.
func (s *TimerState) Tick() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	nextRemaining := s.remaining - time.Second
	if nextRemaining < 0 {
		nextRemaining = 0
	}
	nextProgress := progressForTimer(s.total, nextRemaining)
	nextRunning := nextRemaining > 0
	if s.remaining == nextRemaining && s.running == nextRunning && s.progress == nextProgress {
		s.mu.Unlock()
		return
	}
	s.remaining = nextRemaining
	s.running = nextRunning
	s.progress = nextProgress
	s.bumpLocked()
	revision := s.revision
	remainingState := s.remainingState
	runningState := s.runningState
	progressState := s.progressState
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(remainingState, durationSecondsInt(nextRemaining))
	updateBoolState(runningState, nextRunning)
	updateFloatState(progressState, nextProgress)
	updateIntState(revisionState, revision)
}

// Reset restores the timer to its total duration and stops it.
func (s *TimerState) Reset() {
	s.mu.Lock()
	if s.remaining == s.total && !s.running && s.progress == 0 {
		s.mu.Unlock()
		return
	}
	s.remaining = clampRemaining(s.total, s.total)
	s.running = false
	s.progress = progressForTimer(s.total, s.remaining)
	s.bumpLocked()
	revision := s.revision
	totalState := s.totalState
	remainingState := s.remainingState
	runningState := s.runningState
	progressState := s.progressState
	revisionState := s.revisionState
	total := s.total
	remaining := s.remaining
	s.mu.Unlock()
	updateIntState(totalState, durationSecondsInt(total))
	updateIntState(remainingState, durationSecondsInt(remaining))
	updateBoolState(runningState, false)
	updateFloatState(progressState, 0)
	updateIntState(revisionState, revision)
}

// TotalState returns the owned total-duration primitive state when the bridge is available.
func (s *TimerState) TotalState() *IntState { return s.totalState }

// RemainingState returns the owned remaining-duration primitive state when the bridge is available.
func (s *TimerState) RemainingState() *IntState { return s.remainingState }

// RunningState returns the owned running primitive state when the bridge is available.
func (s *TimerState) RunningState() *BoolState { return s.runningState }

// ProgressState returns the owned progress primitive state when the bridge is available.
func (s *TimerState) ProgressState() *FloatState { return s.progressState }

// RevisionState returns the owned revision primitive state when the bridge is available.
func (s *TimerState) RevisionState() *IntState { return s.revisionState }

// Release releases any owned primitive state handles.
func (s *TimerState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	totalState := s.totalState
	remainingState := s.remainingState
	runningState := s.runningState
	progressState := s.progressState
	revisionState := s.revisionState
	s.totalState = nil
	s.remainingState = nil
	s.runningState = nil
	s.progressState = nil
	s.revisionState = nil
	s.mu.Unlock()
	if totalState != nil {
		totalState.Release()
	}
	if remainingState != nil {
		remainingState.Release()
	}
	if runningState != nil {
		runningState.Release()
	}
	if progressState != nil {
		progressState.Release()
	}
	if revisionState != nil {
		revisionState.Release()
	}
}

func (s *TimerState) bumpLocked() {
	s.revision++
}

func normalizeTimer(total, remaining time.Duration) (time.Duration, time.Duration, float64) {
	if total < 0 {
		total = 0
	}
	if total == 0 {
		return 0, 0, 0
	}
	if remaining < 0 {
		remaining = 0
	}
	if remaining > total {
		remaining = total
	}
	return total, remaining, progressForTimer(total, remaining)
}

func clampRemaining(v, total time.Duration) time.Duration {
	if v < 0 {
		return 0
	}
	if total <= 0 {
		return 0
	}
	if v > total {
		return total
	}
	return v
}

func progressForTimer(total, remaining time.Duration) float64 {
	if total <= 0 {
		return 0
	}
	progress := 1 - float64(remaining)/float64(total)
	switch {
	case progress < 0:
		return 0
	case progress > 1:
		return 1
	default:
		return progress
	}
}

func durationSecondsInt(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(d / time.Second)
}

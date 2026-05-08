package swiftui

import (
	"strings"
	"sync"
)

// NavigationSplitViewColumnKind identifies the preferred compact column.
//
// Bridge surface.
type NavigationSplitViewColumnKind int32

const (
	NavigationSplitViewColumnAutomatic NavigationSplitViewColumnKind = 0
	NavigationSplitViewColumnSidebar   NavigationSplitViewColumnKind = 1
	NavigationSplitViewColumnContent   NavigationSplitViewColumnKind = 2
	NavigationSplitViewColumnDetail    NavigationSplitViewColumnKind = 3
)

// NavigationPathState owns a stable stack of route tokens for manual router flows.
//
// Curated surface.
type NavigationPathState struct {
	mu       sync.RWMutex
	segments []string
	revision int

	revisionState *IntState
}

// NewNavigationPathState creates an empty navigation path.
func NewNavigationPathState() *NavigationPathState {
	return &NavigationPathState{revisionState: newIntStateIfReady(0)}
}

// NewNavigationPathStateWith creates a navigation path seeded with the given segments.
func NewNavigationPathStateWith(path ...string) *NavigationPathState {
	s := &NavigationPathState{revisionState: newIntStateIfReady(0)}
	s.Set(path)
	return s
}

// Get returns a copy of the current path segments.
func (s *NavigationPathState) Get() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.segments...)
}

// Set replaces the path with the provided segments.
func (s *NavigationPathState) Set(path []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	next := normalizePathSegments(path)
	if stringSlicesEqual(s.segments, next) {
		s.mu.Unlock()
		return
	}
	s.segments = next
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Push appends a path segment to the end of the stack.
func (s *NavigationPathState) Push(segment string) {
	if s == nil {
		return
	}
	segment = normalizePathSegment(segment)
	if segment == "" {
		return
	}
	s.mu.Lock()
	s.segments = append(s.segments, segment)
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Pop removes the last path segment.
func (s *NavigationPathState) Pop() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	if len(s.segments) == 0 {
		s.mu.Unlock()
		return false
	}
	s.segments = s.segments[:len(s.segments)-1]
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	return true
}

// Clear removes all path segments.
func (s *NavigationPathState) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if len(s.segments) == 0 {
		s.mu.Unlock()
		return
	}
	s.segments = nil
	s.revision++
	revision := s.revision
	revisionState := s.revisionState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
}

// Depth reports the number of segments in the current path.
func (s *NavigationPathState) Depth() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.segments)
}

// Current reports the last path segment, if any.
func (s *NavigationPathState) Current() (string, bool) {
	if s == nil {
		return "", false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.segments) == 0 {
		return "", false
	}
	return s.segments[len(s.segments)-1], true
}

// Revision reports the current mutation counter.
func (s *NavigationPathState) Revision() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}

// RevisionState exposes the owned revision IntState when the bridge is available.
func (s *NavigationPathState) RevisionState() *IntState {
	if s == nil {
		return nil
	}
	return s.revisionState
}

// String returns a slash-separated rendering of the path.
func (s *NavigationPathState) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(s.Get(), "/")
}

// Release clears the path.
func (s *NavigationPathState) Release() {
	if s == nil {
		return
	}
	s.Clear()
	if s.revisionState != nil {
		s.revisionState.Release()
		s.revisionState = nil
	}
}

// CompactColumnState owns the preferred compact column for split navigation.
//
// Curated surface.
//
// It is backed by the existing IntState bridge so current examples can feed it
// into the generated split-view visibility helpers today.
type CompactColumnState struct {
	state *IntState
}

// NewCompactColumnState creates a compact-column state with the provided initial column.
func NewCompactColumnState(initial NavigationSplitViewColumnKind) *CompactColumnState {
	return &CompactColumnState{
		state: NewIntState(int(columnKindToVisibility(initial))),
	}
}

// Get returns the preferred compact column.
func (s *CompactColumnState) Get() NavigationSplitViewColumnKind {
	if s == nil || s.state == nil {
		return NavigationSplitViewColumnAutomatic
	}
	return visibilityToColumnKind(NavigationSplitViewVisibilityKind(s.state.Get()))
}

// Set updates the preferred compact column.
func (s *CompactColumnState) Set(v NavigationSplitViewColumnKind) {
	if s == nil || s.state == nil {
		return
	}
	s.state.Set(int(columnKindToVisibility(v)))
}

// Visibility returns the closest supported split-view visibility mapping.
func (s *CompactColumnState) Visibility() NavigationSplitViewVisibilityKind {
	if s == nil || s.state == nil {
		return NavigationSplitViewVisibilityAutomatic
	}
	return NavigationSplitViewVisibilityKind(s.state.Get())
}

// VisibilityState exposes the underlying IntState so current split-view helpers can use it.
func (s *CompactColumnState) VisibilityState() *IntState {
	if s == nil {
		return nil
	}
	return s.state
}

// Release releases the underlying IntState.
func (s *CompactColumnState) Release() {
	if s == nil || s.state == nil {
		return
	}
	s.state.Release()
	s.state = nil
}

func normalizePathSegment(segment string) string {
	return strings.TrimSpace(segment)
}

func normalizePathSegments(path []string) []string {
	if len(path) == 0 {
		return nil
	}
	out := make([]string, 0, len(path))
	for _, segment := range path {
		segment = normalizePathSegment(segment)
		if segment == "" {
			continue
		}
		out = append(out, segment)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func columnKindToVisibility(kind NavigationSplitViewColumnKind) NavigationSplitViewVisibilityKind {
	switch kind {
	case NavigationSplitViewColumnSidebar:
		return NavigationSplitViewVisibilityAll
	case NavigationSplitViewColumnContent:
		return NavigationSplitViewVisibilityDoubleColumn
	case NavigationSplitViewColumnDetail:
		return NavigationSplitViewVisibilityDetailOnly
	default:
		return NavigationSplitViewVisibilityAutomatic
	}
}

func visibilityToColumnKind(kind NavigationSplitViewVisibilityKind) NavigationSplitViewColumnKind {
	switch kind {
	case NavigationSplitViewVisibilityAll:
		return NavigationSplitViewColumnSidebar
	case NavigationSplitViewVisibilityDoubleColumn:
		return NavigationSplitViewColumnContent
	case NavigationSplitViewVisibilityDetailOnly:
		return NavigationSplitViewColumnDetail
	default:
		return NavigationSplitViewColumnAutomatic
	}
}

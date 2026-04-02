package a2uiruntime

import (
	"fmt"
	"sync"
	"time"

	"github.com/tmc/swiftui"
	"github.com/tmc/swiftui/a2ui"
)

type stateCache struct {
	mu           sync.Mutex
	strings      map[string]*swiftui.StringState
	stringLists  map[string]*swiftui.StringListState
	ints         map[string]*swiftui.IntState
	floats       map[string]*swiftui.FloatState
	bools        map[string]*swiftui.BoolState
	dates        map[string]*swiftui.DateState
	modalStates  map[string]*swiftui.BoolState
	validation   map[string]*swiftui.StringState
	stringValues map[string]string
	players      map[string]uintptr
}

func newStateCache() *stateCache {
	return &stateCache{
		strings:      make(map[string]*swiftui.StringState),
		stringLists:  make(map[string]*swiftui.StringListState),
		ints:         make(map[string]*swiftui.IntState),
		floats:       make(map[string]*swiftui.FloatState),
		bools:        make(map[string]*swiftui.BoolState),
		dates:        make(map[string]*swiftui.DateState),
		modalStates:  make(map[string]*swiftui.BoolState),
		validation:   make(map[string]*swiftui.StringState),
		stringValues: make(map[string]string),
		players:      make(map[string]uintptr),
	}
}

func (sc *stateCache) getString(path, initial string) *swiftui.StringState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.strings[path]; ok {
		return s
	}
	s := swiftui.NewStringState(initial)
	sc.strings[path] = s
	sc.stringValues[path] = initial
	return s
}

func (sc *stateCache) getValidation(path string) *swiftui.StringState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.validation[path]; ok {
		return s
	}
	s := swiftui.NewStringState("")
	sc.validation[path] = s
	return s
}

func (sc *stateCache) setStringValue(path, value string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.stringValues[path] = value
}

func (sc *stateCache) getStringList(path string, initial []string) *swiftui.StringListState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.stringLists[path]; ok {
		return s
	}
	s := swiftui.NewStringListState(initial)
	sc.stringLists[path] = s
	return s
}

func (sc *stateCache) getInt(path string, initial int) *swiftui.IntState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.ints[path]; ok {
		return s
	}
	s := swiftui.NewIntState(initial)
	sc.ints[path] = s
	return s
}

func (sc *stateCache) getFloat(path string, initial float64) *swiftui.FloatState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.floats[path]; ok {
		return s
	}
	s := swiftui.NewFloatState(initial)
	sc.floats[path] = s
	return s
}

func (sc *stateCache) getDate(path string, initial float64) *swiftui.DateState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.dates[path]; ok {
		return s
	}
	s := swiftui.NewDateState(initial)
	sc.dates[path] = s
	return s
}

func (sc *stateCache) getBool(path string, initial bool) *swiftui.BoolState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.bools[path]; ok {
		return s
	}
	s := swiftui.NewBoolState(initial)
	sc.bools[path] = s
	return s
}

func (sc *stateCache) getModal(compID string) *swiftui.BoolState {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.modalStates[compID]; ok {
		return s
	}
	s := swiftui.NewBoolState(false)
	sc.modalStates[compID] = s
	return s
}

func (sc *stateCache) syncFromDataModel(dm *a2ui.DataModel) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for path, s := range sc.strings {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		if str, ok := v.(string); ok {
			s.Set(str)
			sc.stringValues[path] = str
		}
	}
	for path, s := range sc.ints {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			s.Set(int(n))
		case int:
			s.Set(n)
		case bool:
			if n {
				s.Set(1)
			} else {
				s.Set(0)
			}
		}
	}
	for path, s := range sc.floats {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			s.Set(n)
		case int:
			s.Set(float64(n))
		}
	}
	for path, s := range sc.bools {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		if b, ok := v.(bool); ok {
			s.Set(b)
		}
	}
	for path, s := range sc.stringLists {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch items := v.(type) {
		case []string:
			s.Set(items)
		case []any:
			out := make([]string, 0, len(items))
			for _, item := range items {
				out = append(out, fmt.Sprintf("%v", item))
			}
			s.Set(out)
		}
	}
	for path, s := range sc.dates {
		v, err := dm.Get(path)
		if err != nil {
			continue
		}
		switch n := v.(type) {
		case float64:
			s.Set(n)
		case string:
			if t, err := time.Parse(time.RFC3339, n); err == nil {
				s.Set(float64(t.Unix()))
			}
		}
	}
}

func (sc *stateCache) snapshotValues() map[string]any {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	out := make(map[string]any, len(sc.stringValues)+len(sc.stringLists)+len(sc.ints)+len(sc.floats)+len(sc.bools)+len(sc.dates))
	for path, value := range sc.stringValues {
		out[path] = value
	}
	for path, state := range sc.stringLists {
		out[path] = state.Get()
	}
	for path, state := range sc.ints {
		out[path] = state.Get()
	}
	for path, state := range sc.floats {
		out[path] = state.Get()
	}
	for path, state := range sc.bools {
		out[path] = state.Get()
	}
	for path, state := range sc.dates {
		out[path] = time.Unix(int64(state.Get()), 0).UTC().Format(time.RFC3339)
	}
	return out
}

func (sc *stateCache) getPlayer(key string) (uintptr, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	ptr, ok := sc.players[key]
	return ptr, ok
}

func (sc *stateCache) setPlayer(key string, ptr uintptr) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.players[key] = ptr
}

package a2ui

import (
	"testing"
)

func TestDataModelGet(t *testing.T) {
	dm := NewDataModel()
	dm.Data = map[string]any{
		"name": "test",
		"nested": map[string]any{
			"value": float64(42),
		},
		"items": []any{"a", "b", "c"},
	}

	tests := []struct {
		name    string
		pointer string
		want    any
		wantErr bool
	}{
		{name: "root", pointer: "", want: dm.Data},
		{name: "top-level key", pointer: "/name", want: "test"},
		{name: "nested key", pointer: "/nested/value", want: float64(42)},
		{name: "array index 0", pointer: "/items/0", want: "a"},
		{name: "array index 2", pointer: "/items/2", want: "c"},
		{name: "missing key", pointer: "/missing", wantErr: true},
		{name: "bad array index", pointer: "/items/x", wantErr: true},
		{name: "out of range index", pointer: "/items/5", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dm.Get(tt.pointer)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Compare using JSON for maps/slices.
			if !jsonEqual(got, tt.want) {
				t.Errorf("Get(%q) = %v, want %v", tt.pointer, got, tt.want)
			}
		})
	}
}

func TestDataModelSet(t *testing.T) {
	tests := []struct {
		name    string
		setup   map[string]any
		pointer string
		value   any
		check   func(*DataModel) bool
		wantErr bool
	}{
		{
			name:    "set top-level",
			setup:   map[string]any{},
			pointer: "/greeting",
			value:   "hello",
			check:   func(dm *DataModel) bool { v, _ := dm.Get("/greeting"); return v == "hello" },
		},
		{
			name:    "set nested creates intermediates",
			setup:   map[string]any{},
			pointer: "/a/b",
			value:   float64(1),
			check:   func(dm *DataModel) bool { v, _ := dm.Get("/a/b"); return v == float64(1) },
		},
		{
			name:    "set array element",
			setup:   map[string]any{"items": []any{"a", "b", "c"}},
			pointer: "/items/1",
			value:   "B",
			check:   func(dm *DataModel) bool { v, _ := dm.Get("/items/1"); return v == "B" },
		},
		{
			name:    "set root",
			setup:   map[string]any{"old": true},
			pointer: "",
			value:   map[string]any{"new": true},
			check:   func(dm *DataModel) bool { _, err := dm.Get("/old"); return err != nil },
		},
		{
			name:    "set root non-map",
			setup:   map[string]any{},
			pointer: "",
			value:   "string",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDataModel()
			dm.Data = tt.setup
			err := dm.Set(tt.pointer, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.check(dm) {
				t.Errorf("check failed after Set(%q, %v)", tt.pointer, tt.value)
			}
		})
	}
}

func TestDataModelRemove(t *testing.T) {
	tests := []struct {
		name    string
		setup   map[string]any
		pointer string
		check   func(*DataModel) bool
		wantErr bool
	}{
		{
			name:    "remove key",
			setup:   map[string]any{"a": float64(1), "b": float64(2)},
			pointer: "/a",
			check:   func(dm *DataModel) bool { _, err := dm.Get("/a"); return err != nil },
		},
		{
			name:    "remove nested key",
			setup:   map[string]any{"x": map[string]any{"y": float64(1), "z": float64(2)}},
			pointer: "/x/y",
			check:   func(dm *DataModel) bool { _, err := dm.Get("/x/y"); return err != nil },
		},
		{
			name:    "remove root clears data",
			setup:   map[string]any{"a": float64(1)},
			pointer: "",
			check:   func(dm *DataModel) bool { return len(dm.Data) == 0 },
		},
		{
			name:    "remove missing key",
			setup:   map[string]any{},
			pointer: "/missing",
			wantErr: true,
		},
		{
			name:    "remove array element",
			setup:   map[string]any{"items": []any{"a", "b", "c"}},
			pointer: "/items/1",
			check: func(dm *DataModel) bool {
				v, _ := dm.Get("/items")
				arr := v.([]any)
				return len(arr) == 2 && arr[0] == "a" && arr[1] == "c"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDataModel()
			dm.Data = tt.setup
			err := dm.Remove(tt.pointer)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.check(dm) {
				t.Errorf("check failed after Remove(%q)", tt.pointer)
			}
		})
	}
}

func TestEscapedPointers(t *testing.T) {
	dm := NewDataModel()
	dm.Data = map[string]any{
		"a/b":  "slash",
		"c~d":  "tilde",
		"e~1f": "literal",
	}
	tests := []struct {
		name    string
		pointer string
		want    any
	}{
		{name: "escaped slash", pointer: "/a~1b", want: "slash"},
		{name: "escaped tilde", pointer: "/c~0d", want: "tilde"},
		{name: "literal ~1 in key", pointer: "/e~01f", want: "literal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dm.Get(tt.pointer)
			if err != nil {
				t.Fatalf("Get(%q): %v", tt.pointer, err)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %v, want %v", tt.pointer, got, tt.want)
			}
		})
	}
}

func jsonEqual(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !jsonEqual(v, bv[k]) {
				return false
			}
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

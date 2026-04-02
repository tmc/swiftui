package a2ui

import (
	"encoding/json"
	"testing"
)

func TestResolveLiteral(t *testing.T) {
	tests := []struct {
		name string
		val  DynamicValue
		want any
	}{
		{name: "string", val: ValueString("hello"), want: "hello"},
		{name: "number", val: ValueNumber(42), want: float64(42)},
		{name: "bool", val: ValueBool(true), want: true},
		{name: "array", val: ValueArray([]any{"a", float64(2)}), want: []any{"a", float64(2)}},
	}

	dm := NewDataModel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.val, dm)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if !jsonEqual(got, tt.want) {
				t.Fatalf("Resolve() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveBinding(t *testing.T) {
	dm := NewDataModel()
	dm.Data = map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  float64(30),
		},
	}

	got, err := Resolve(ValueBinding("/user/name"), dm)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "Alice" {
		t.Fatalf("Resolve() = %v, want Alice", got)
	}
}

func TestResolveFunctions(t *testing.T) {
	dm := NewDataModel()
	dm.Data = map[string]any{
		"email": "user@example.com",
		"name":  "Bob",
		"items": []any{"a", "b", "c"},
		"count": float64(5),
		"empty": "",
	}

	tests := []struct {
		name string
		val  DynamicValue
		want any
	}{
		{
			name: "required with value",
			val: ValueFunc(FunctionCall{
				Call: "required",
				Args: map[string]any{"value": ValueBinding("/name")},
			}),
			want: true,
		},
		{
			name: "required with empty",
			val: ValueFunc(FunctionCall{
				Call: "required",
				Args: map[string]any{"value": ValueBinding("/empty")},
			}),
			want: false,
		},
		{
			name: "length string",
			val: ValueFunc(FunctionCall{
				Call: "length",
				Args: map[string]any{"value": ValueBinding("/name")},
			}),
			want: 3,
		},
		{
			name: "length bounded",
			val: ValueFunc(FunctionCall{
				Call: "length",
				Args: map[string]any{"value": ValueBinding("/name"), "min": 2, "max": 4},
			}),
			want: true,
		},
		{
			name: "numeric true",
			val: ValueFunc(FunctionCall{
				Call: "numeric",
				Args: map[string]any{"value": ValueBinding("/count")},
			}),
			want: true,
		},
		{
			name: "regex true",
			val: ValueFunc(FunctionCall{
				Call: "regex",
				Args: map[string]any{"value": ValueBinding("/name"), "pattern": "^B.*"},
			}),
			want: true,
		},
		{
			name: "email valid",
			val: ValueFunc(FunctionCall{
				Call: "email",
				Args: map[string]any{"value": ValueBinding("/email")},
			}),
			want: true,
		},
		{
			name: "and true",
			val: ValueFunc(FunctionCall{
				Call: "and",
				Args: map[string]any{"a": ValueBool(true), "b": ValueBool(true)},
			}),
			want: true,
		},
		{
			name: "or true",
			val: ValueFunc(FunctionCall{
				Call: "or",
				Args: map[string]any{"a": ValueBool(false), "b": ValueBool(true)},
			}),
			want: true,
		},
		{
			name: "not false",
			val: ValueFunc(FunctionCall{
				Call: "not",
				Args: map[string]any{"value": ValueBool(true)},
			}),
			want: false,
		},
		{
			name: "formatString",
			val: ValueFunc(FunctionCall{
				Call: "formatString",
				Args: map[string]any{"value": "Hello ${/name}!"},
			}),
			want: "Hello Bob!",
		},
		{
			name: "formatNumber",
			val: ValueFunc(FunctionCall{
				Call: "formatNumber",
				Args: map[string]any{
					"decimals": 0,
					"grouping": true,
					"value":    float64(12345),
				},
			}),
			want: "12,345",
		},
		{
			name: "formatCurrency",
			val: ValueFunc(FunctionCall{
				Call: "formatCurrency",
				Args: map[string]any{
					"currency": "$",
					"value":    float64(42.5),
				},
			}),
			want: "$42.50",
		},
		{
			name: "formatDate",
			val: ValueFunc(FunctionCall{
				Call: "formatDate",
				Args: map[string]any{
					"format": "YYYY-MM-dd",
					"value":  "2025-01-02T15:04:05Z",
				},
			}),
			want: "2025-01-02",
		},
		{
			name: "pluralize",
			val: ValueFunc(FunctionCall{
				Call: "pluralize",
				Args: map[string]any{
					"one":   "one item",
					"other": "items",
					"value": float64(1),
				},
			}),
			want: "one item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.val, dm)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if !jsonEqual(got, tt.want) {
				t.Fatalf("Resolve() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveDynamicStringHelpers(t *testing.T) {
	dm := NewDataModel()
	dm.Data = map[string]any{"x": "hello", "n": float64(7), "b": true}

	if got, err := ResolveDynamicString(StringLiteral("world"), dm); err != nil || got != "world" {
		t.Fatalf("ResolveDynamicString() = %q, %v", got, err)
	}
	if got, err := ResolveDynamicString(StringBinding("/x"), dm); err != nil || got != "hello" {
		t.Fatalf("ResolveDynamicString(binding) = %q, %v", got, err)
	}
	if got, err := ResolveDynamicNumber(NumberBinding("/n"), dm); err != nil || got != 7 {
		t.Fatalf("ResolveDynamicNumber() = %v, %v", got, err)
	}
	if got, err := ResolveDynamicBoolean(BoolBinding("/b"), dm); err != nil || !got {
		t.Fatalf("ResolveDynamicBoolean() = %v, %v", got, err)
	}
}

func TestDynamicValueJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{name: "string literal", json: `"hello"`},
		{name: "number literal", json: `42`},
		{name: "bool literal", json: `true`},
		{name: "binding", json: `{"path":"/user/name"}`},
		{name: "function", json: `{"call":"required","args":{"value":"test"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got DynamicValue
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			data, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var roundTrip any
			if err := json.Unmarshal(data, &roundTrip); err != nil {
				t.Fatalf("Unmarshal round-trip: %v", err)
			}
		})
	}
}

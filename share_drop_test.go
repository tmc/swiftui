package swiftui

import "testing"

func TestShareItemPayload(t *testing.T) {
	tests := []struct {
		name  string
		item  ShareItem
		kind  string
		value string
		ok    bool
	}{
		{
			name:  "text",
			item:  ShareItem{Text: "hello"},
			kind:  "text",
			value: "hello",
			ok:    true,
		},
		{
			name:  "url",
			item:  ShareItem{URL: "https://example.com"},
			kind:  "url",
			value: "https://example.com",
			ok:    true,
		},
		{
			name:  "file",
			item:  ShareItem{FilePath: "/tmp/demo.txt"},
			kind:  "file",
			value: "/tmp/demo.txt",
			ok:    true,
		},
		{
			name: "ambiguous",
			item: ShareItem{Text: "hello", URL: "https://example.com"},
		},
		{
			name: "empty",
			item: ShareItem{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, value, ok := shareItemPayload(tc.item)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if kind != tc.kind {
				t.Fatalf("kind = %q, want %q", kind, tc.kind)
			}
			if value != tc.value {
				t.Fatalf("value = %q, want %q", value, tc.value)
			}
		})
	}
}

func TestShareItemAndDropPayloadKinds(t *testing.T) {
	share := ShareItem{URL: "https://example.com"}
	if got, want := share.Kind(), "url"; got != want {
		t.Fatalf("ShareItem.Kind() = %q, want %q", got, want)
	}
	if got, want := share.Value(), "https://example.com"; got != want {
		t.Fatalf("ShareItem.Value() = %q, want %q", got, want)
	}
	invalidShare := ShareItem{Text: "hello", URL: "https://example.com"}
	if got := invalidShare.Kind(); got != "" {
		t.Fatalf("invalid ShareItem.Kind() = %q, want empty", got)
	}
	if got := invalidShare.Value(); got != "" {
		t.Fatalf("invalid ShareItem.Value() = %q, want empty", got)
	}

	drop := DropPayload{FilePath: "/tmp/demo.txt"}
	if got, want := drop.Kind(), "file"; got != want {
		t.Fatalf("DropPayload.Kind() = %q, want %q", got, want)
	}
	if got, want := drop.Value(), "/tmp/demo.txt"; got != want {
		t.Fatalf("DropPayload.Value() = %q, want %q", got, want)
	}
	invalidDrop := DropPayload{Text: "hello", FilePath: "/tmp/demo.txt"}
	if got := invalidDrop.Kind(); got != "" {
		t.Fatalf("invalid DropPayload.Kind() = %q, want empty", got)
	}
	if got := invalidDrop.Value(); got != "" {
		t.Fatalf("invalid DropPayload.Value() = %q, want empty", got)
	}
}

func TestPastePayloadPayload(t *testing.T) {
	tests := []struct {
		name  string
		item  PastePayload
		kind  string
		value string
		ok    bool
	}{
		{
			name:  "text",
			item:  PastePayload{Text: "hello"},
			kind:  "text",
			value: "hello",
			ok:    true,
		},
		{
			name:  "url",
			item:  PastePayload{URL: "https://example.com"},
			kind:  "url",
			value: "https://example.com",
			ok:    true,
		},
		{
			name:  "file",
			item:  PastePayload{FilePath: "/tmp/demo.txt"},
			kind:  "file",
			value: "/tmp/demo.txt",
			ok:    true,
		},
		{
			name: "ambiguous",
			item: PastePayload{Text: "hello", URL: "https://example.com"},
		},
		{
			name: "empty",
			item: PastePayload{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, value, ok := singlePayload(tc.item.Text, tc.item.URL, tc.item.FilePath)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if kind != tc.kind {
				t.Fatalf("kind = %q, want %q", kind, tc.kind)
			}
			if value != tc.value {
				t.Fatalf("value = %q, want %q", value, tc.value)
			}
		})
	}
}

func TestDropPayloadPayload(t *testing.T) {
	tests := []struct {
		name  string
		item  DropPayload
		kind  string
		value string
		ok    bool
	}{
		{
			name:  "text",
			item:  DropPayload{Text: "hello"},
			kind:  "text",
			value: "hello",
			ok:    true,
		},
		{
			name:  "url",
			item:  DropPayload{URL: "https://example.com"},
			kind:  "url",
			value: "https://example.com",
			ok:    true,
		},
		{
			name:  "file",
			item:  DropPayload{FilePath: "/tmp/demo.txt"},
			kind:  "file",
			value: "/tmp/demo.txt",
			ok:    true,
		},
		{
			name: "ambiguous",
			item: DropPayload{Text: "hello", FilePath: "/tmp/demo.txt"},
		},
		{
			name: "empty",
			item: DropPayload{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, value, ok := dropPayloadPayload(tc.item)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if kind != tc.kind {
				t.Fatalf("kind = %q, want %q", kind, tc.kind)
			}
			if value != tc.value {
				t.Fatalf("value = %q, want %q", value, tc.value)
			}
		})
	}
}

func TestPayloadKindAndValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		paste PastePayload
		want  string
	}{
		{
			name:  "paste text",
			paste: PastePayload{Text: "hello"},
			value: "hello",
			want:  "text",
		},
		{
			name:  "paste url",
			paste: PastePayload{URL: "https://example.com"},
			value: "https://example.com",
			want:  "url",
		},
		{
			name:  "paste file",
			paste: PastePayload{FilePath: "/tmp/demo.txt"},
			value: "/tmp/demo.txt",
			want:  "file",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.paste.Kind(); got != tc.want {
				t.Fatalf("PastePayload.Kind() = %q, want %q", got, tc.want)
			}
			if got := tc.paste.Value(); got != tc.value {
				t.Fatalf("PastePayload.Value() = %q, want %q", got, tc.value)
			}
		})
	}
}

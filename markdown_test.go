package swiftui

import (
	"reflect"
	"testing"
)

func TestMarkdownBlocks(t *testing.T) {
	got := markdownBlocks("before\n\n```go\nfmt.Println(\"hi\")\n```\n\nafter")
	want := []markdownBlock{
		{text: "before\n\n"},
		{text: "fmt.Println(\"hi\")", lang: "go", code: true},
		{text: "\n\nafter"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("markdownBlocks(...) = %#v, want %#v", got, want)
	}
}

func TestMarkdownHeading(t *testing.T) {
	level, body, ok := markdownHeading("### heading")
	if !ok || level != 3 || body != "heading" {
		t.Fatalf("markdownHeading(...) = (%d, %q, %v), want (3, %q, true)", level, body, ok, "heading")
	}
}

func TestMarkdownListItems(t *testing.T) {
	got := markdownListItems("- one\n2. two")
	want := []markdownListItem{
		{marker: "•", text: "one"},
		{marker: "2.", text: "two"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("markdownListItems(...) = %#v, want %#v", got, want)
	}
}

func TestMarkdownBlockquote(t *testing.T) {
	got, ok := markdownBlockquote("> first\n> second")
	if !ok || got != "first\nsecond" {
		t.Fatalf("markdownBlockquote(...) = (%q, %v), want (%q, true)", got, ok, "first\nsecond")
	}
}

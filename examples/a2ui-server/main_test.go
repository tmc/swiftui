package main

import (
	"fmt"
	"testing"

	"github.com/tmc/swiftui/a2ui"
)

func TestShowcaseComponentsResolveReferences(t *testing.T) {
	s := &server{}
	components := s.buildShowcaseComponents()

	byID := make(map[string]a2ui.Component, len(components))
	for _, c := range components {
		if c.ID == "" {
			t.Fatal("component has empty id")
		}
		if _, dup := byID[c.ID]; dup {
			t.Fatalf("duplicate component id %q", c.ID)
		}
		byID[c.ID] = c
	}

	for _, c := range components {
		for _, ref := range componentRefs(c) {
			if _, ok := byID[ref]; !ok {
				t.Fatalf("component %q references missing component %q", c.ID, ref)
			}
		}
	}
}

func componentRefs(c a2ui.Component) []string {
	var refs []string
	add := func(id string) {
		if id != "" {
			refs = append(refs, id)
		}
	}

	switch {
	case c.Button != nil:
		add(c.Button.Child)
	case c.Card != nil:
		add(c.Card.Child)
	case c.Column != nil:
		refs = append(refs, c.Column.Children.IDs...)
		if c.Column.Children.Template != nil {
			add(c.Column.Children.Template.ComponentID)
		}
	case c.Row != nil:
		refs = append(refs, c.Row.Children.IDs...)
		if c.Row.Children.Template != nil {
			add(c.Row.Children.Template.ComponentID)
		}
	case c.List != nil:
		refs = append(refs, c.List.Children.IDs...)
		if c.List.Children.Template != nil {
			add(c.List.Children.Template.ComponentID)
		}
	case c.Tabs != nil:
		for _, tab := range c.Tabs.Tabs {
			add(tab.Child)
		}
	case c.Modal != nil:
		add(c.Modal.Trigger)
		add(c.Modal.Content)
	}

	return refs
}

func TestShowcaseComponentsIncludeMediaProgressCards(t *testing.T) {
	s := &server{}
	components := s.buildShowcaseComponents()

	want := map[string]bool{
		"progress-card": false,
		"spinner-card":  false,
	}
	for _, c := range components {
		if _, ok := want[c.ID]; ok {
			want[c.ID] = true
		}
	}
	for id, ok := range want {
		if !ok {
			t.Fatal(fmt.Sprintf("missing component %q", id))
		}
	}
}

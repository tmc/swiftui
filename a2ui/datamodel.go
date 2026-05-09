package a2ui

import (
	"fmt"
	"strconv"
	"strings"
)

// DataModel holds a JSON-compatible data tree and supports access via JSON Pointer (RFC 6901).
type DataModel struct {
	Data map[string]any
}

// NewDataModel returns an initialized DataModel.
func NewDataModel() *DataModel {
	return &DataModel{Data: make(map[string]any)}
}

// Get resolves a JSON Pointer path and returns the value.
func (dm *DataModel) Get(pointer string) (any, error) {
	if pointer == "" {
		return dm.Data, nil
	}
	tokens, err := parsePointer(pointer)
	if err != nil {
		return nil, err
	}
	var current any = dm.Data
	for _, tok := range tokens {
		current, err = resolve(current, tok)
		if err != nil {
			return nil, fmt.Errorf("get %q: %w", pointer, err)
		}
	}
	return current, nil
}

// Set sets the value at the given JSON Pointer path, creating intermediate maps as needed.
func (dm *DataModel) Set(pointer string, value any) error {
	if pointer == "" || pointer == "/" {
		m, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("set root: value must be map[string]any")
		}
		dm.Data = m
		return nil
	}
	tokens, err := parsePointer(pointer)
	if err != nil {
		return err
	}
	if dm.Data == nil {
		dm.Data = make(map[string]any)
	}
	var current any = dm.Data
	for _, tok := range tokens[:len(tokens)-1] {
		next, err := resolve(current, tok)
		if err != nil {
			// create intermediate map
			m, ok := current.(map[string]any)
			if !ok {
				return fmt.Errorf("set %q: cannot traverse non-map at %q", pointer, tok)
			}
			child := make(map[string]any)
			m[tok] = child
			current = child
			continue
		}
		current = next
	}
	last := tokens[len(tokens)-1]
	switch c := current.(type) {
	case map[string]any:
		c[last] = value
		return nil
	case []any:
		idx, err := strconv.Atoi(last)
		if err != nil {
			return fmt.Errorf("set %q: invalid array index %q", pointer, last)
		}
		if idx < 0 || idx >= len(c) {
			return fmt.Errorf("set %q: index %d out of range", pointer, idx)
		}
		c[idx] = value
		return nil
	default:
		return fmt.Errorf("set %q: cannot index into %T", pointer, current)
	}
}

// Remove removes the value at the given JSON Pointer path.
func (dm *DataModel) Remove(pointer string) error {
	if pointer == "" {
		dm.Data = make(map[string]any)
		return nil
	}
	tokens, err := parsePointer(pointer)
	if err != nil {
		return err
	}

	// Walk to the parent of the target.
	var parent any = dm.Data
	for _, tok := range tokens[:len(tokens)-1] {
		parent, err = resolve(parent, tok)
		if err != nil {
			return fmt.Errorf("remove %q: %w", pointer, err)
		}
	}
	last := tokens[len(tokens)-1]
	switch c := parent.(type) {
	case map[string]any:
		if _, ok := c[last]; !ok {
			return fmt.Errorf("remove %q: key %q not found", pointer, last)
		}
		// Check if the value is being removed from a map whose value is a slice entry.
		delete(c, last)
		return nil
	case []any:
		idx, err := strconv.Atoi(last)
		if err != nil {
			return fmt.Errorf("remove %q: invalid array index %q", pointer, last)
		}
		if idx < 0 || idx >= len(c) {
			return fmt.Errorf("remove %q: index %d out of range", pointer, idx)
		}
		// To splice the array we need the grandparent to replace the slice value.
		// Walk to grandparent (parent of the array).
		var grandparent any = dm.Data
		for _, tok := range tokens[:len(tokens)-2] {
			grandparent, _ = resolve(grandparent, tok)
		}
		arrayKey := tokens[len(tokens)-2]
		newSlice := append(c[:idx], c[idx+1:]...)
		switch gp := grandparent.(type) {
		case map[string]any:
			gp[arrayKey] = newSlice
		case []any:
			gpIdx, _ := strconv.Atoi(arrayKey)
			gp[gpIdx] = newSlice
		}
		return nil
	default:
		return fmt.Errorf("remove %q: cannot index into %T", pointer, parent)
	}
}

// parsePointer splits a JSON Pointer string into unescaped reference tokens.
func parsePointer(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if pointer[0] != '/' {
		return nil, fmt.Errorf("invalid JSON Pointer %q: must start with /", pointer)
	}
	parts := strings.Split(pointer[1:], "/")
	tokens := make([]string, len(parts))
	for i, p := range parts {
		// Unescape per RFC 6901: ~1 -> /, ~0 -> ~
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		tokens[i] = p
	}
	return tokens, nil
}

// resolve looks up a single token in the given container.
func resolve(container any, token string) (any, error) {
	switch c := container.(type) {
	case map[string]any:
		v, ok := c[token]
		if !ok {
			return nil, fmt.Errorf("key %q not found", token)
		}
		return v, nil
	case []any:
		idx, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid array index %q", token)
		}
		if idx < 0 || idx >= len(c) {
			return nil, fmt.Errorf("index %d out of range (len %d)", idx, len(c))
		}
		return c[idx], nil
	default:
		return nil, fmt.Errorf("cannot index into %T with %q", container, token)
	}
}

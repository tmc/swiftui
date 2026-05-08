package swiftui

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// PhotosPickerLazyFileHandle describes one lazily loadable file-backed asset.
//
// Curated surface.
type PhotosPickerLazyFileHandle struct {
	Path string
}

// NewPhotosPickerLazyFileHandle creates a lazy file handle for a concrete path.
//
// The zero value is not useful; pass a real file path.
func NewPhotosPickerLazyFileHandle(path string) *PhotosPickerLazyFileHandle {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	return &PhotosPickerLazyFileHandle{Path: filepath.Clean(path)}
}

// Load reads the underlying file data on demand.
func (h *PhotosPickerLazyFileHandle) Load() ([]byte, error) {
	if h == nil {
		return nil, errors.New("swiftui: lazy file handle is nil")
	}
	path := strings.TrimSpace(h.Path)
	if path == "" {
		return nil, errors.New("swiftui: lazy file handle path is empty")
	}
	return os.ReadFile(path)
}

// PhotosPickerItem describes one normalized photo selection.
//
// Curated surface.
type PhotosPickerItem struct {
	ID        string
	Filename  string
	UTType    string
	MediaKind string
	Order     int
	// LazyFile is an optional curated file-backed preview handle used by
	// deterministic sample items. The native bridge remains metadata-only.
	LazyFile *PhotosPickerLazyFileHandle
}

// PhotosPickerSelectionState owns the current normalized photo selection.
//
// Curated surface.
//
// It models stable selection identity before the native PhotosPicker bridge is
// fully wired into the public view surface.
type PhotosPickerSelectionState struct {
	mu sync.Mutex

	items map[string]PhotosPickerItem

	revision int

	revisionState *IntState
	countState    *IntState
}

// NewPhotosPickerSelectionState creates a new photo selection state.
func NewPhotosPickerSelectionState(initial ...PhotosPickerItem) *PhotosPickerSelectionState {
	s := &PhotosPickerSelectionState{}
	s.set(initial)
	if bridgeReady() {
		s.revisionState = newIntStateIfReady(s.revision)
		s.countState = newIntStateIfReady(len(s.items))
	}
	return s
}

// Items returns the selected items in stable ID order.
func (s *PhotosPickerSelectionState) Items() []PhotosPickerItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.itemsLocked()
}

// Set replaces the selected items.
func (s *PhotosPickerSelectionState) Set(items []PhotosPickerItem) {
	s.set(items)
}

// Add inserts or replaces one selected item by stable ID.
func (s *PhotosPickerSelectionState) Add(item PhotosPickerItem) {
	item, ok := normalizePhotosPickerItem(item)
	if !ok {
		return
	}
	s.mu.Lock()
	if s.items == nil {
		s.items = make(map[string]PhotosPickerItem)
	}
	if existing, ok := s.items[item.ID]; ok && existing == item {
		s.mu.Unlock()
		return
	}
	s.items[item.ID] = item
	s.bumpLocked()
	revision := s.revision
	count := len(s.items)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

// Toggle inserts item when it is not selected and removes it when it is.
func (s *PhotosPickerSelectionState) Toggle(item PhotosPickerItem) {
	item, ok := normalizePhotosPickerItem(item)
	if !ok {
		return
	}
	if s.Has(item.ID) {
		s.Remove(item.ID)
		return
	}
	s.Add(item)
}

// Remove removes one item by stable ID.
func (s *PhotosPickerSelectionState) Remove(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	if len(s.items) == 0 {
		s.mu.Unlock()
		return
	}
	if _, ok := s.items[id]; !ok {
		s.mu.Unlock()
		return
	}
	delete(s.items, id)
	s.bumpLocked()
	revision := s.revision
	count := len(s.items)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

// Clear removes all selected items.
func (s *PhotosPickerSelectionState) Clear() {
	s.mu.Lock()
	if len(s.items) == 0 {
		s.mu.Unlock()
		return
	}
	s.items = make(map[string]PhotosPickerItem)
	s.bumpLocked()
	revision := s.revision
	count := len(s.items)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

// Has reports whether id is selected.
func (s *PhotosPickerSelectionState) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[id]
	return ok
}

// Count reports the number of selected items.
func (s *PhotosPickerSelectionState) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

// Revision reports the mutation counter.
func (s *PhotosPickerSelectionState) Revision() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision
}

// RevisionState returns the owned revision state when the bridge is available.
func (s *PhotosPickerSelectionState) RevisionState() *IntState { return s.revisionState }

// CountState returns the owned count state when the bridge is available.
func (s *PhotosPickerSelectionState) CountState() *IntState { return s.countState }

// Release releases any owned primitive state handles.
func (s *PhotosPickerSelectionState) Release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	revisionState := s.revisionState
	countState := s.countState
	s.revisionState = nil
	s.countState = nil
	s.items = nil
	s.mu.Unlock()
	if revisionState != nil {
		revisionState.Release()
	}
	if countState != nil {
		countState.Release()
	}
}

func (s *PhotosPickerSelectionState) set(items []PhotosPickerItem) {
	s.mu.Lock()
	s.items = make(map[string]PhotosPickerItem)
	for _, item := range items {
		item, ok := normalizePhotosPickerItem(item)
		if !ok {
			continue
		}
		s.items[item.ID] = item
	}
	s.bumpLocked()
	revision := s.revision
	count := len(s.items)
	revisionState := s.revisionState
	countState := s.countState
	s.mu.Unlock()
	updateIntState(revisionState, revision)
	updateIntState(countState, count)
}

func (s *PhotosPickerSelectionState) itemsLocked() []PhotosPickerItem {
	out := make([]PhotosPickerItem, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	sortPhotosPickerItems(out)
	return out
}

func (s *PhotosPickerSelectionState) bumpLocked() {
	s.revision++
}

func sortPhotosPickerItems(items []PhotosPickerItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		return items[i].ID < items[j].ID
	})
}

func normalizePhotosPickerItem(item PhotosPickerItem) (PhotosPickerItem, bool) {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		return PhotosPickerItem{}, false
	}
	item.Filename = strings.TrimSpace(item.Filename)
	item.UTType = strings.TrimSpace(item.UTType)
	item.MediaKind = strings.TrimSpace(item.MediaKind)
	if item.LazyFile != nil {
		item.LazyFile.Path = strings.TrimSpace(item.LazyFile.Path)
		if item.LazyFile.Path == "" {
			item.LazyFile = nil
		} else {
			item.LazyFile.Path = filepath.Clean(item.LazyFile.Path)
		}
	}
	if item.MediaKind == "" {
		item.MediaKind = inferPhotosPickerMediaKind(item.UTType)
	}
	return item, true
}

func inferPhotosPickerMediaKind(utType string) string {
	switch {
	case strings.Contains(utType, "image"),
		strings.Contains(utType, "png"),
		strings.Contains(utType, "jpeg"),
		strings.Contains(utType, "heic"),
		strings.Contains(utType, "gif"),
		strings.Contains(utType, "tiff"):
		return "image"
	case strings.Contains(utType, "movie"),
		strings.Contains(utType, "video"),
		strings.Contains(utType, "mpeg"),
		strings.Contains(utType, "quicktime"):
		return "video"
	default:
		return ""
	}
}

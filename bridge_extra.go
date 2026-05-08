package swiftui

import (
	"encoding/json"
	"runtime"
	"sync"
	"unsafe"
)

// ImageFit identifies remote image sizing behavior.
//
// Bridge surface.
type ImageFit int32

const (
	ImageFitContain ImageFit = iota
	ImageFitCover
	ImageFitFill
	ImageFitNone
	ImageFitScaleDown
)

// DateBounds constrains a date picker to an optional lower and upper bound.
//
// Bridge surface.
type DateBounds struct {
	Min *float64
	Max *float64
}

// DatePickerMode controls whether the picker shows a date, a time, or both.
//
// Bridge surface.
type DatePickerMode int32

const (
	DatePickerModeDate DatePickerMode = iota
	DatePickerModeTime
	DatePickerModeDateAndTime
)

// TextInputPolicy controls filtered text entry and validation state reporting.
//
// Bridge surface.
type TextInputPolicy struct {
	AllowedPattern    string
	ValidationPattern string
	ValidState        *BoolState
}

// ChoiceOption defines a labeled selectable value for searchable pickers.
//
// Bridge surface.
type ChoiceOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// PhotosPickerMatching narrows the native PhotosPicker selection filter.
//
// Bridge surface.
type PhotosPickerMatching int32

const (
	PhotosPickerMatchingAny PhotosPickerMatching = iota
	PhotosPickerMatchingImages
	PhotosPickerMatchingVideos
)

// PhotosPickerConfig controls the native PhotosPicker bridge.
//
// Bridge surface.
type PhotosPickerConfig struct {
	Matching          PhotosPickerMatching
	MaxSelectionCount int
}

// StringListState is an observable slice of strings.
//
// Bridge surface.
type StringListState struct {
	ptr      uintptr
	retained *retainedOwned
}

var (
	extraLibOnce sync.Once

	_SUIAsyncImageFit          func(*byte, int32) uintptr
	_SUIDatePickerBounded      func(*byte, uintptr, int32, float64, int32, float64, uintptr) uintptr
	_SUIDatePickerBoundedMode  func(*byte, uintptr, int32, float64, int32, float64, int32, uintptr) uintptr
	_SUITextFieldPolicy        func(*byte, uintptr, *byte, *byte, uintptr, uintptr, uintptr) uintptr
	_SUISecureFieldPolicy      func(*byte, uintptr, *byte, *byte, uintptr, uintptr, uintptr) uintptr
	_SUITextEditorPolicy       func(uintptr, *byte, *byte, uintptr, uintptr) uintptr
	_SUIViewFrameAligned       func(uintptr, float64, float64, int32, int32) uintptr
	_SUIViewMaxFrameAligned    func(uintptr, float64, float64, int32, int32) uintptr
	_SUIStateCreateStringList         func(*byte) uintptr
	_SUIStateGetStringListJSON        func(uintptr) *byte
	_SUIStateSetStringListJSON        func(uintptr, *byte)
	_SUIStateCreateStringListPacked   func(*byte, int32) uintptr
	_SUIStateGetStringListPacked      func(uintptr, *int32) *byte
	_SUIStateSetStringListPacked      func(uintptr, *byte, int32)
	_SUIFreePackedBuffer              func(*byte)
	_SUISearchablePicker              func(*byte, *byte, uintptr, *byte, uintptr) uintptr
	_SUISearchableMultiPicker         func(*byte, *byte, uintptr, *byte, uintptr) uintptr
	_SUISearchablePickerPacked        func(*byte, *byte, uintptr, *byte, int32, uintptr) uintptr
	_SUISearchableMultiPickerPacked   func(*byte, *byte, uintptr, *byte, int32, uintptr) uintptr
	_SUIPhotosPicker           func(*byte, int32, int32, uintptr) uintptr
	_SUIOpenPanel              func(*byte, uintptr) uintptr
)

func ensureExtraLibFuncs() {
	extraLibOnce.Do(func() {
		if libHandle != 0 {
			tryRegisterLibFunc(&_SUIAsyncImageFit, libHandle, "SUIAsyncImageFit")
			tryRegisterLibFunc(&_SUIDatePickerBounded, libHandle, "SUIDatePickerBounded")
			tryRegisterLibFunc(&_SUIDatePickerBoundedMode, libHandle, "SUIDatePickerBoundedMode")
			tryRegisterLibFunc(&_SUITextFieldPolicy, libHandle, "SUITextFieldPolicy")
			tryRegisterLibFunc(&_SUISecureFieldPolicy, libHandle, "SUISecureFieldPolicy")
			tryRegisterLibFunc(&_SUITextEditorPolicy, libHandle, "SUITextEditorPolicy")
			tryRegisterLibFunc(&_SUIViewFrameAligned, libHandle, "SUIViewFrameAligned")
			tryRegisterLibFunc(&_SUIViewMaxFrameAligned, libHandle, "SUIViewMaxFrameAligned")
			tryRegisterLibFunc(&_SUIStateCreateStringList, libHandle, "SUIStateCreateStringList")
			tryRegisterLibFunc(&_SUIStateGetStringListJSON, libHandle, "SUIStateGetStringListJSON")
			tryRegisterLibFunc(&_SUIStateSetStringListJSON, libHandle, "SUIStateSetStringListJSON")
			tryRegisterLibFunc(&_SUIStateCreateStringListPacked, libHandle, "SUIStateCreateStringListPacked")
			tryRegisterLibFunc(&_SUIStateGetStringListPacked, libHandle, "SUIStateGetStringListPacked")
			tryRegisterLibFunc(&_SUIStateSetStringListPacked, libHandle, "SUIStateSetStringListPacked")
			tryRegisterLibFunc(&_SUIFreePackedBuffer, libHandle, "SUIFreePackedBuffer")
			tryRegisterLibFunc(&_SUISearchablePicker, libHandle, "SUISearchablePicker")
			tryRegisterLibFunc(&_SUISearchableMultiPicker, libHandle, "SUISearchableMultiPicker")
			tryRegisterLibFunc(&_SUISearchablePickerPacked, libHandle, "SUISearchablePickerPacked")
			tryRegisterLibFunc(&_SUISearchableMultiPickerPacked, libHandle, "SUISearchableMultiPickerPacked")
			tryRegisterLibFunc(&_SUIPhotosPicker, libHandle, "SUIPhotosPicker")
			tryRegisterLibFunc(&_SUIOpenPanel, libHandle, "SUIOpenPanel")
		}
		setExtraUnavailableStubs()
	})
}

func setExtraUnavailableStubs() {
	stub := func(name string) {
		if loadErr != nil {
			panic("swiftui: " + name + ": " + loadErr.Error())
		}
		panic("swiftui: " + name + ": dylib not loaded")
	}
	if _SUIAsyncImageFit == nil {
		_SUIAsyncImageFit = func(*byte, int32) uintptr { stub("SUIAsyncImageFit"); return 0 }
	}
	if _SUIDatePickerBounded == nil {
		_SUIDatePickerBounded = func(*byte, uintptr, int32, float64, int32, float64, uintptr) uintptr {
			stub("SUIDatePickerBounded")
			return 0
		}
	}
	if _SUIDatePickerBoundedMode == nil {
		_SUIDatePickerBoundedMode = func(*byte, uintptr, int32, float64, int32, float64, int32, uintptr) uintptr {
			return 0
		}
	}
	if _SUITextFieldPolicy == nil {
		_SUITextFieldPolicy = func(*byte, uintptr, *byte, *byte, uintptr, uintptr, uintptr) uintptr {
			stub("SUITextFieldPolicy")
			return 0
		}
	}
	if _SUISecureFieldPolicy == nil {
		_SUISecureFieldPolicy = func(*byte, uintptr, *byte, *byte, uintptr, uintptr, uintptr) uintptr {
			stub("SUISecureFieldPolicy")
			return 0
		}
	}
	if _SUITextEditorPolicy == nil {
		_SUITextEditorPolicy = func(uintptr, *byte, *byte, uintptr, uintptr) uintptr {
			stub("SUITextEditorPolicy")
			return 0
		}
	}
	if _SUIViewFrameAligned == nil {
		_SUIViewFrameAligned = func(uintptr, float64, float64, int32, int32) uintptr {
			stub("SUIViewFrameAligned")
			return 0
		}
	}
	if _SUIViewMaxFrameAligned == nil {
		_SUIViewMaxFrameAligned = func(uintptr, float64, float64, int32, int32) uintptr {
			stub("SUIViewMaxFrameAligned")
			return 0
		}
	}
	if _SUIStateCreateStringList == nil {
		_SUIStateCreateStringList = func(*byte) uintptr { stub("SUIStateCreateStringList"); return 0 }
	}
	if _SUIStateGetStringListJSON == nil {
		_SUIStateGetStringListJSON = func(uintptr) *byte { stub("SUIStateGetStringListJSON"); return nil }
	}
	if _SUIStateSetStringListJSON == nil {
		_SUIStateSetStringListJSON = func(uintptr, *byte) { stub("SUIStateSetStringListJSON") }
	}
	// Packed variants: left nil when the running dylib predates P4 so the
	// Go-side helpers can transparently fall back to the JSON path.
	if _SUISearchablePicker == nil {
		_SUISearchablePicker = func(*byte, *byte, uintptr, *byte, uintptr) uintptr {
			stub("SUISearchablePicker")
			return 0
		}
	}
	if _SUISearchableMultiPicker == nil {
		_SUISearchableMultiPicker = func(*byte, *byte, uintptr, *byte, uintptr) uintptr {
			stub("SUISearchableMultiPicker")
			return 0
		}
	}
	if _SUIPhotosPicker == nil {
		_SUIPhotosPicker = func(*byte, int32, int32, uintptr) uintptr {
			stub("SUIPhotosPicker")
			return 0
		}
	}
	if _SUIOpenPanel == nil {
		_SUIOpenPanel = func(*byte, uintptr) uintptr {
			stub("SUIOpenPanel")
			return 0
		}
	}
}

func goStringAndFree(p *byte) string {
	if p == nil {
		return ""
	}
	defer _SUIFreeString(p)
	var buf []byte
	for i := uintptr(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + i))
		if b == 0 {
			break
		}
		buf = append(buf, b)
	}
	return string(buf)
}

// marshalStringSliceJSON JSON-encodes v. It is the pre-P4 encoder, retained as
// an opt-in debug and persistence helper. The hot path between Go and Swift
// uses the packed wire format (see wire_packed.go and withPackedStringSlice).
func marshalStringSliceJSON(v []string) string {
	if v == nil {
		v = []string{}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// marshalStringSlice is a compatibility alias for benchmarks and tests. New
// bridge code should use withPackedStringSlice instead.
func marshalStringSlice(v []string) string { return marshalStringSliceJSON(v) }

// marshalChoiceOptionsJSON is the pre-P4 JSON encoder for ChoiceOption slices.
// Retained for debug/persistence; hot paths use withPackedChoiceOptions.
func marshalChoiceOptionsJSON(v []ChoiceOption) string {
	if v == nil {
		v = []ChoiceOption{}
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// marshalChoiceOptions is a compatibility alias; see marshalChoiceOptionsJSON.
func marshalChoiceOptions(v []ChoiceOption) string { return marshalChoiceOptionsJSON(v) }

func withTwoCStrings(a, b string, fn func(*byte, *byte)) {
	withCString(a, func(aC *byte) {
		withCString(b, func(bC *byte) {
			fn(aC, bC)
		})
	})
}

// NewStringListState creates a new observable string slice.
func NewStringListState(initial []string) *StringListState {
	ensureExtraLibFuncs()
	var ptr uintptr
	if _SUIStateCreateStringListPacked != nil {
		bp, p, n := acquirePackedStringSliceBuf(initial)
		ptr = _SUIStateCreateStringListPacked(p, int32(n))
		releasePackedBuf(bp)
	} else {
		withCString(marshalStringSliceJSON(initial), func(jsonC *byte) {
			ptr = _SUIStateCreateStringList(jsonC)
		})
	}
	return &StringListState{ptr: ptr, retained: newRetainedOwned(ptr)}
}

// Get returns the current slice value.
func (s *StringListState) Get() []string {
	ensureExtraLibFuncs()
	if s == nil || s.ptr == 0 {
		return nil
	}
	if _SUIStateGetStringListPacked != nil && _SUIFreePackedBuffer != nil {
		var n int32
		p := _SUIStateGetStringListPacked(s.ptr, &n)
		if p == nil || n <= 0 {
			if p != nil {
				_SUIFreePackedBuffer(p)
			}
			return nil
		}
		out, err := decodePackedStringSliceFromPointer(p, int(n))
		_SUIFreePackedBuffer(p)
		if err != nil {
			return nil
		}
		return out
	}
	data := goStringAndFree(_SUIStateGetStringListJSON(s.ptr))
	if data == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		return nil
	}
	return out
}

// Set updates the slice value.
func (s *StringListState) Set(v []string) {
	ensureExtraLibFuncs()
	if s == nil || s.ptr == 0 {
		return
	}
	if _SUIStateSetStringListPacked != nil {
		// Avoid a closure that would capture s and escape. The direct
		// acquire/release form keeps StringListState.Set allocation-free
		// aside from the purego reflect trampoline.
		bp, p, n := acquirePackedStringSliceBuf(v)
		_SUIStateSetStringListPacked(s.ptr, p, int32(n))
		releasePackedBuf(bp)
		return
	}
	withCString(marshalStringSliceJSON(v), func(jsonC *byte) {
		_SUIStateSetStringListJSON(s.ptr, jsonC)
	})
}

// Release decrements the underlying Swift retain count.
func (s *StringListState) Release() {
	if s == nil || s.retained == nil {
		return
	}
	s.retained.release()
	s.retained = nil
	s.ptr = 0
}

// AsyncImageFit loads and displays a remote image using the requested fit mode.
func AsyncImageFit(url string, fit ImageFit) View {
	ensureExtraLibFuncs()
	var ptr uintptr
	withCString(url, func(urlC *byte) {
		ptr = _SUIAsyncImageFit(urlC, int32(fit))
	})
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

// DatePickerBounded creates a date picker with optional minimum and maximum bounds.
func DatePickerBounded(label string, state *DateState, bounds DateBounds, onChange func()) View {
	return DatePickerBoundedMode(label, state, bounds, DatePickerModeDateAndTime, onChange)
}

// DatePickerBoundedMode creates a bounded date picker with an explicit display mode.
func DatePickerBoundedMode(label string, state *DateState, bounds DateBounds, mode DatePickerMode, onChange func()) View {
	ensureExtraLibFuncs()
	onChangeID := registerCallback(onChange)
	var (
		ptr    uintptr
		hasMin int32
		min    float64
		hasMax int32
		max    float64
	)
	if bounds.Min != nil {
		hasMin = 1
		min = *bounds.Min
	}
	if bounds.Max != nil {
		hasMax = 1
		max = *bounds.Max
	}
	withCString(label, func(labelC *byte) {
		if _SUIDatePickerBoundedMode != nil {
			ptr = _SUIDatePickerBoundedMode(labelC, state.ptr, hasMin, min, hasMax, max, int32(mode), onChangeID)
		}
		if ptr == 0 {
			ptr = _SUIDatePickerBounded(labelC, state.ptr, hasMin, min, hasMax, max, onChangeID)
		}
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	return ret
}

// TextFieldPolicy creates a text field with optional filtering and validation.
func TextFieldPolicy(placeholder string, state *StringState, policy TextInputPolicy, onChange func(), onSubmit func()) View {
	ensureExtraLibFuncs()
	onChangeID := registerCallback(onChange)
	onSubmitID := registerCallback(onSubmit)
	validPtr := uintptr(0)
	if policy.ValidState != nil {
		validPtr = policy.ValidState.ptr
	}
	var ptr uintptr
	withCString(placeholder, func(placeholderC *byte) {
		withTwoCStrings(policy.AllowedPattern, policy.ValidationPattern, func(allowedC, validationC *byte) {
			ptr = _SUITextFieldPolicy(placeholderC, state.ptr, allowedC, validationC, validPtr, onChangeID, onSubmitID)
		})
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	ret.retained.addCallbackID(onSubmitID)
	return ret
}

// SecureFieldPolicy creates a secure field with optional filtering and validation.
func SecureFieldPolicy(placeholder string, state *StringState, policy TextInputPolicy, onChange func(), onSubmit func()) View {
	ensureExtraLibFuncs()
	onChangeID := registerCallback(onChange)
	onSubmitID := registerCallback(onSubmit)
	validPtr := uintptr(0)
	if policy.ValidState != nil {
		validPtr = policy.ValidState.ptr
	}
	var ptr uintptr
	withCString(placeholder, func(placeholderC *byte) {
		withTwoCStrings(policy.AllowedPattern, policy.ValidationPattern, func(allowedC, validationC *byte) {
			ptr = _SUISecureFieldPolicy(placeholderC, state.ptr, allowedC, validationC, validPtr, onChangeID, onSubmitID)
		})
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	ret.retained.addCallbackID(onSubmitID)
	return ret
}

// TextEditorPolicy creates a text editor with optional filtering and validation.
func TextEditorPolicy(state *StringState, policy TextInputPolicy, onChange func()) View {
	ensureExtraLibFuncs()
	onChangeID := registerCallback(onChange)
	validPtr := uintptr(0)
	if policy.ValidState != nil {
		validPtr = policy.ValidState.ptr
	}
	var ptr uintptr
	withTwoCStrings(policy.AllowedPattern, policy.ValidationPattern, func(allowedC, validationC *byte) {
		ptr = _SUITextEditorPolicy(state.ptr, allowedC, validationC, validPtr, onChangeID)
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	return ret
}

// SearchablePicker creates a searchable single-select picker backed by a string selection state.
func SearchablePicker(label, prompt string, selection *StringState, options []ChoiceOption, onChange func()) View {
	ensureExtraLibFuncs()
	onChangeID := registerCallback(onChange)
	var ptr uintptr
	withCString(label, func(labelC *byte) {
		withCString(prompt, func(promptC *byte) {
			if _SUISearchablePickerPacked != nil {
				withPackedChoiceOptions(options, func(p *byte, n int) {
					ptr = _SUISearchablePickerPacked(labelC, promptC, selection.ptr, p, int32(n), onChangeID)
				})
				return
			}
			withCString(marshalChoiceOptionsJSON(options), func(optionsC *byte) {
				ptr = _SUISearchablePicker(labelC, promptC, selection.ptr, optionsC, onChangeID)
			})
		})
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	return ret
}

// SearchableMultiPicker creates a searchable multi-select picker backed by a string slice state.
func SearchableMultiPicker(label, prompt string, selection *StringListState, options []ChoiceOption, onChange func()) View {
	ensureExtraLibFuncs()
	onChangeID := registerCallback(onChange)
	var ptr uintptr
	withCString(label, func(labelC *byte) {
		withCString(prompt, func(promptC *byte) {
			if _SUISearchableMultiPickerPacked != nil {
				withPackedChoiceOptions(options, func(p *byte, n int) {
					ptr = _SUISearchableMultiPickerPacked(labelC, promptC, selection.ptr, p, int32(n), onChangeID)
				})
				return
			}
			withCString(marshalChoiceOptionsJSON(options), func(optionsC *byte) {
				ptr = _SUISearchableMultiPicker(labelC, promptC, selection.ptr, optionsC, onChangeID)
			})
		})
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(onChangeID)
	return ret
}

func unmarshalPhotosPickerItems(data string) []PhotosPickerItem {
	if data == "" {
		return nil
	}
	var items []PhotosPickerItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return nil
	}
	filtered := items[:0]
	for _, item := range items {
		item, ok := normalizePhotosPickerItem(item)
		if !ok {
			continue
		}
		filtered = append(filtered, item)
	}
	sortPhotosPickerItems(filtered)
	return filtered
}

// PhotosPickerNative creates a native PhotosPicker bridge backed by PhotosPickerSelectionState.
//
// It keeps the Go API concrete: selected items are normalized to stable IDs,
// filenames when available, and UTType identifiers. The native bridge itself
// stays metadata-only; curated items may attach lazy file handles for
// deterministic file-backed previews without exposing generic Transferable
// decoding.
func PhotosPickerNative(label string, selection *PhotosPickerSelectionState, config PhotosPickerConfig) View {
	ensureExtraLibFuncs()
	if label == "" {
		label = "Choose Photos"
	}
	if selection == nil {
		return Button(label, func() {})
	}
	callbackID := registerStringCallback(func(data string) bool {
		selection.Set(unmarshalPhotosPickerItems(data))
		return true
	})
	var ptr uintptr
	withCString(label, func(labelC *byte) {
		ptr = _SUIPhotosPicker(labelC, int32(config.Matching), int32(config.MaxSelectionCount), callbackID)
	})
	ret := View{ptr: ptr, retained: newRetainedOwned(ptr)}
	ret.retained.addCallbackID(callbackID)
	return ret
}

// FrameAligned sets an optional fixed frame with explicit alignment.
func (v View) FrameAligned(width, height float64, horizontal HorizontalAlignment, vertical VerticalAlignment) View {
	ensureExtraLibFuncs()
	ptr := _SUIViewFrameAligned(v.ptr, width, height, int32(horizontal), int32(vertical))
	runtime.KeepAlive(v.retained)
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

// MaxFrameAligned sets optional maximum frame constraints with explicit alignment.
// Use -1 for .infinity and 0 for nil, matching MaxFrame.
func (v View) MaxFrameAligned(maxWidth, maxHeight float64, horizontal HorizontalAlignment, vertical VerticalAlignment) View {
	ensureExtraLibFuncs()
	ptr := _SUIViewMaxFrameAligned(v.ptr, maxWidth, maxHeight, int32(horizontal), int32(vertical))
	runtime.KeepAlive(v.retained)
	return View{ptr: ptr, retained: newRetainedOwned(ptr)}
}

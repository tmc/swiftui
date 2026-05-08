package swiftui

// TableColumnLayoutSnapshot captures explicit column visibility, widths, and
// named presets as a concrete value object suitable for caller-owned
// persistence.
//
// Curated surface.
type TableColumnLayoutSnapshot struct {
	HiddenIDs       []string
	Widths          map[string]float64
	CurrentPresetID string
	Presets         []TableColumnPreset
}

// CaptureTableColumnLayoutSnapshot captures the current column state.
func CaptureTableColumnLayoutSnapshot(
	visibility *TableColumnVisibilityState,
	widths *TableColumnWidthState,
	presets *TableColumnPresetState,
) TableColumnLayoutSnapshot {
	var snapshot TableColumnLayoutSnapshot
	if visibility != nil {
		snapshot.HiddenIDs = visibility.HiddenIDs()
	}
	if widths != nil {
		snapshot.Widths = widths.Widths()
	}
	if presets != nil {
		presetSnapshot := presets.Snapshot()
		snapshot.CurrentPresetID = presetSnapshot.CurrentPresetID
		snapshot.Presets = presetSnapshot.Presets
	}
	return snapshot
}

// ApplyTableColumnLayoutSnapshot restores snapshot onto the provided state
// holders.
func ApplyTableColumnLayoutSnapshot(
	snapshot TableColumnLayoutSnapshot,
	visibility *TableColumnVisibilityState,
	widths *TableColumnWidthState,
	presets *TableColumnPresetState,
) {
	if presets != nil {
		presets.ReplaceSnapshot(TableColumnPresetSnapshot{
			CurrentPresetID: snapshot.CurrentPresetID,
			Presets:         snapshot.Presets,
		})
	}
	if widths != nil {
		widths.ReplaceWidths(snapshot.Widths)
	}
	if visibility != nil {
		visibility.ReplaceHiddenIDs(snapshot.HiddenIDs...)
	}
}

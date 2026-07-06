package swiftui

import "strings"

// OpenPanelOptions configures OpenPanel.
type OpenPanelOptions struct {
	Title                string
	Message              string
	Prompt               string
	Directory            string
	AllowsMultiple       bool
	CanChooseFiles       bool
	CanChooseDirectories bool
}

// SavePanelOptions configures SavePanel.
type SavePanelOptions struct {
	Title     string
	Message   string
	Prompt    string
	Directory string
	Name      string
}

// OpenPanel presents a file picker and returns selected paths.
func OpenPanel(opts OpenPanelOptions) ([]string, bool) {
	canFiles := opts.CanChooseFiles
	canDirs := opts.CanChooseDirectories
	if !canFiles && !canDirs {
		canFiles = true
	}
	var ret *byte
	withCString(opts.Title, func(titleC *byte) {
		withCString(opts.Message, func(messageC *byte) {
			withCString(opts.Prompt, func(promptC *byte) {
				withCString(opts.Directory, func(directoryC *byte) {
					ret = _SUIOpenPanel(titleC, messageC, promptC, directoryC, boolInt(opts.AllowsMultiple), boolInt(canFiles), boolInt(canDirs))
				})
			})
		})
	})
	if ret == nil {
		return nil, false
	}
	defer _SUIFreeString(ret)
	s := cStringToGoString(ret)
	if s == "" {
		return nil, false
	}
	return strings.Split(s, "\n"), true
}

// SavePanel presents a save-location picker and returns the selected path.
func SavePanel(opts SavePanelOptions) (string, bool) {
	var ret *byte
	withCString(opts.Title, func(titleC *byte) {
		withCString(opts.Message, func(messageC *byte) {
			withCString(opts.Prompt, func(promptC *byte) {
				withCString(opts.Directory, func(directoryC *byte) {
					withCString(opts.Name, func(nameC *byte) {
						ret = _SUISavePanel(titleC, messageC, promptC, directoryC, nameC)
					})
				})
			})
		})
	})
	if ret == nil {
		return "", false
	}
	defer _SUIFreeString(ret)
	s := cStringToGoString(ret)
	if s == "" {
		return "", false
	}
	return s, true
}

func boolInt(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

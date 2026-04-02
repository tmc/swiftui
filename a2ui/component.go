package a2ui

import (
	"encoding/json"
	"fmt"

	googlea2ui "github.com/google/A2UI/agent_sdks/go/a2ui"
)

// Version is the A2UI protocol version implemented by the Google SDK.
const Version = googlea2ui.Version

// Re-exported Google SDK types used by the local renderer and examples.
type (
	DynamicString           = googlea2ui.DynamicString
	DynamicNumber           = googlea2ui.DynamicNumber
	DynamicBoolean          = googlea2ui.DynamicBoolean
	DynamicStringList       = googlea2ui.DynamicStringList
	DynamicValue            = googlea2ui.DynamicValue
	DataBinding             = googlea2ui.DataBinding
	FunctionCall            = googlea2ui.FunctionCall
	ChildList               = googlea2ui.ChildList
	ChildTemplate           = googlea2ui.ChildTemplate
	CheckRule               = googlea2ui.CheckRule
	AccessibilityAttributes = googlea2ui.AccessibilityAttributes
	Theme                   = googlea2ui.Theme
	ClientCapabilities      = googlea2ui.ClientCapabilities
	ClientCapabilitiesV09   = googlea2ui.ClientCapabilitiesV09
	ServerCapabilities      = googlea2ui.ServerCapabilities
	ServerCapabilitiesV09   = googlea2ui.ServerCapabilitiesV09
	CatalogDef              = googlea2ui.CatalogDef
	FunctionDefinition      = googlea2ui.FunctionDefinition
	Action                  = googlea2ui.Action
	EventAction             = googlea2ui.EventAction
	IconNameOrPath          = googlea2ui.IconNameOrPath

	AudioPlayerComponent   = googlea2ui.AudioPlayerComponent
	ButtonComponent        = googlea2ui.ButtonComponent
	CardComponent          = googlea2ui.CardComponent
	CheckBoxComponent      = googlea2ui.CheckBoxComponent
	ChoicePickerComponent  = googlea2ui.ChoicePickerComponent
	ColumnComponent        = googlea2ui.ColumnComponent
	DateTimeInputComponent = googlea2ui.DateTimeInputComponent
	DividerComponent       = googlea2ui.DividerComponent
	IconComponent          = googlea2ui.IconComponent
	ImageComponent         = googlea2ui.ImageComponent
	ListComponent          = googlea2ui.ListComponent
	ModalComponent         = googlea2ui.ModalComponent
	RowComponent           = googlea2ui.RowComponent
	SliderComponent        = googlea2ui.SliderComponent
	TabsComponent          = googlea2ui.TabsComponent
	TextComponent          = googlea2ui.TextComponent
	TextFieldComponent     = googlea2ui.TextFieldComponent
	VideoComponent         = googlea2ui.VideoComponent

	TabDef       = googlea2ui.TabDef
	ChoiceOption = googlea2ui.ChoiceOption

	IconName                 = googlea2ui.IconName
	ButtonVariant            = googlea2ui.ButtonVariant
	ChoicePickerDisplayStyle = googlea2ui.ChoicePickerDisplayStyle
	ChoicePickerVariant      = googlea2ui.ChoicePickerVariant
	DividerAxis              = googlea2ui.DividerAxis
	ImageFit                 = googlea2ui.ImageFit
	ImageVariant             = googlea2ui.ImageVariant
	LayoutAlign              = googlea2ui.LayoutAlign
	LayoutJustify            = googlea2ui.LayoutJustify
	ListDirection            = googlea2ui.ListDirection
	TextFieldVariant         = googlea2ui.TextFieldVariant
	TextVariant              = googlea2ui.TextVariant
	ReturnType               = googlea2ui.ReturnType
)

// Re-exported Google SDK constants and constructors used by the local code.
const (
	IconInfo                             = googlea2ui.IconInfo
	ButtonVariantDefault                 = googlea2ui.ButtonVariantDefault
	ButtonVariantPrimary                 = googlea2ui.ButtonVariantPrimary
	ButtonVariantBorderless              = googlea2ui.ButtonVariantBorderless
	ChoicePickerDisplayStyleCheckbox     = googlea2ui.ChoicePickerDisplayStyleCheckbox
	ChoicePickerDisplayStyleChips        = googlea2ui.ChoicePickerDisplayStyleChips
	ChoicePickerVariantMultipleSelection = googlea2ui.ChoicePickerVariantMultipleSelection
	ChoicePickerVariantMutuallyExclusive = googlea2ui.ChoicePickerVariantMutuallyExclusive
	DividerAxisHorizontal                = googlea2ui.DividerAxisHorizontal
	DividerAxisVertical                  = googlea2ui.DividerAxisVertical
	ImageFitContain                      = googlea2ui.ImageFitContain
	ImageFitCover                        = googlea2ui.ImageFitCover
	ImageFitFill                         = googlea2ui.ImageFitFill
	ImageFitNone                         = googlea2ui.ImageFitNone
	ImageFitScaleDown                    = googlea2ui.ImageFitScaleDown
	ImageVariantIcon                     = googlea2ui.ImageVariantIcon
	ImageVariantAvatar                   = googlea2ui.ImageVariantAvatar
	ImageVariantSmallFeature             = googlea2ui.ImageVariantSmallFeature
	ImageVariantMediumFeature            = googlea2ui.ImageVariantMediumFeature
	ImageVariantLargeFeature             = googlea2ui.ImageVariantLargeFeature
	ImageVariantHeader                   = googlea2ui.ImageVariantHeader
	LayoutAlignCenter                    = googlea2ui.LayoutAlignCenter
	LayoutAlignEnd                       = googlea2ui.LayoutAlignEnd
	LayoutAlignStart                     = googlea2ui.LayoutAlignStart
	LayoutAlignStretch                   = googlea2ui.LayoutAlignStretch
	LayoutJustifyStart                   = googlea2ui.LayoutJustifyStart
	LayoutJustifyCenter                  = googlea2ui.LayoutJustifyCenter
	LayoutJustifyEnd                     = googlea2ui.LayoutJustifyEnd
	LayoutJustifySpaceBetween            = googlea2ui.LayoutJustifySpaceBetween
	LayoutJustifySpaceAround             = googlea2ui.LayoutJustifySpaceAround
	LayoutJustifySpaceEvenly             = googlea2ui.LayoutJustifySpaceEvenly
	LayoutJustifyStretch                 = googlea2ui.LayoutJustifyStretch
	ListDirectionVertical                = googlea2ui.ListDirectionVertical
	ListDirectionHorizontal              = googlea2ui.ListDirectionHorizontal
	TextFieldVariantLongText             = googlea2ui.TextFieldVariantLongText
	TextFieldVariantNumber               = googlea2ui.TextFieldVariantNumber
	TextFieldVariantShortText            = googlea2ui.TextFieldVariantShortText
	TextFieldVariantObscured             = googlea2ui.TextFieldVariantObscured
	TextVariantH1                        = googlea2ui.TextVariantH1
	TextVariantH2                        = googlea2ui.TextVariantH2
	TextVariantH3                        = googlea2ui.TextVariantH3
	TextVariantH4                        = googlea2ui.TextVariantH4
	TextVariantH5                        = googlea2ui.TextVariantH5
	TextVariantCaption                   = googlea2ui.TextVariantCaption
	TextVariantBody                      = googlea2ui.TextVariantBody
)

var (
	StringLiteral     = googlea2ui.StringLiteral
	StringBinding     = googlea2ui.StringBinding
	StringFunc        = googlea2ui.StringFunc
	NumberLiteral     = googlea2ui.NumberLiteral
	NumberBinding     = googlea2ui.NumberBinding
	NumberFunc        = googlea2ui.NumberFunc
	BoolLiteral       = googlea2ui.BoolLiteral
	BoolBinding       = googlea2ui.BoolBinding
	BoolFunc          = googlea2ui.BoolFunc
	StringListLiteral = googlea2ui.StringListLiteral
	StringListBinding = googlea2ui.StringListBinding
	StringListFunc    = googlea2ui.StringListFunc
	ValueString       = googlea2ui.ValueString
	ValueNumber       = googlea2ui.ValueNumber
	ValueBool         = googlea2ui.ValueBool
	ValueArray        = googlea2ui.ValueArray
	ValueBinding      = googlea2ui.ValueBinding
	ValueFunc         = googlea2ui.ValueFunc

	And            = googlea2ui.And
	Email          = googlea2ui.Email
	FormatCurrency = googlea2ui.FormatCurrency
	FormatDate     = googlea2ui.FormatDate
	FormatNumber   = googlea2ui.FormatNumber
	FormatString   = googlea2ui.FormatString
	Length         = googlea2ui.Length
	Not            = googlea2ui.Not
	Numeric        = googlea2ui.Numeric
	OpenURL        = googlea2ui.OpenUrl
	Or             = googlea2ui.Or
	Pluralize      = googlea2ui.Pluralize
	Regex          = googlea2ui.Regex
	Required       = googlea2ui.Required
)

// ProgressComponent is a small local extension for the demo renderer.
// The upstream SDK does not currently define a Progress component.
type ProgressComponent struct {
	Value *DynamicNumber `json:"value,omitempty"`
	Max   *float64       `json:"max,omitempty"`
}

// Component mirrors the Google SDK component model and carries a few local
// extensions that the examples still use.
type Component struct {
	ID            string                   `json:"id"`
	Accessibility *AccessibilityAttributes `json:"accessibility,omitempty"`
	Weight        *float64                 `json:"weight,omitempty"`
	Checks        []CheckRule              `json:"checks,omitempty"`

	Text          *TextComponent          `json:"-"`
	Image         *ImageComponent         `json:"-"`
	Icon          *IconComponent          `json:"-"`
	Video         *VideoComponent         `json:"-"`
	AudioPlayer   *AudioPlayerComponent   `json:"-"`
	Row           *RowComponent           `json:"-"`
	Column        *ColumnComponent        `json:"-"`
	List          *ListComponent          `json:"-"`
	Card          *CardComponent          `json:"-"`
	Tabs          *TabsComponent          `json:"-"`
	Modal         *ModalComponent         `json:"-"`
	Divider       *DividerComponent       `json:"-"`
	Button        *ButtonComponent        `json:"-"`
	TextField     *TextFieldComponent     `json:"-"`
	CheckBox      *CheckBoxComponent      `json:"-"`
	ChoicePicker  *ChoicePickerComponent  `json:"-"`
	Slider        *SliderComponent        `json:"-"`
	DateTimeInput *DateTimeInputComponent `json:"-"`
	Progress      *ProgressComponent      `json:"-"`

	// Local extensions not modeled in the upstream SDK.
	Action        *Action  `json:"-"`
	Spacing       *float64 `json:"-"`
	Padding       *float64 `json:"-"`
	Strikethrough *bool    `json:"-"`
}

// FromSDK converts a Google SDK component into the local wrapper type.
func FromSDK(c googlea2ui.Component) Component {
	return Component{
		ID:            c.ID,
		Accessibility: c.Accessibility,
		Weight:        c.Weight,
		Checks:        c.Checks,
		Text:          c.Text,
		Image:         c.Image,
		Icon:          c.Icon,
		Video:         c.Video,
		AudioPlayer:   c.AudioPlayer,
		Row:           c.Row,
		Column:        c.Column,
		List:          c.List,
		Card:          c.Card,
		Tabs:          c.Tabs,
		Modal:         c.Modal,
		Divider:       c.Divider,
		Button:        c.Button,
		TextField:     c.TextField,
		CheckBox:      c.CheckBox,
		ChoicePicker:  c.ChoicePicker,
		Slider:        c.Slider,
		DateTimeInput: c.DateTimeInput,
	}
}

// ProgressBar constructs a local Progress component.
func ProgressBar(id string, value DynamicNumber, max float64) Component {
	return Component{
		ID: id,
		Progress: &ProgressComponent{
			Value: &value,
			Max:   &max,
		},
	}
}

// Spinner constructs an indeterminate local Progress component.
func Spinner(id string) Component {
	return Component{
		ID:       id,
		Progress: &ProgressComponent{},
	}
}

// ComponentType returns the A2UI discriminator string.
func (c Component) ComponentType() string {
	var (
		typ   string
		count int
	)
	set := func(name string, ok bool) {
		if ok {
			typ = name
			count++
		}
	}
	set(ComponentText, c.Text != nil)
	set(ComponentImage, c.Image != nil)
	set(ComponentIcon, c.Icon != nil)
	set(ComponentVideo, c.Video != nil)
	set(ComponentAudioPlayer, c.AudioPlayer != nil)
	set(ComponentRow, c.Row != nil)
	set(ComponentColumn, c.Column != nil)
	set(ComponentList, c.List != nil)
	set(ComponentCard, c.Card != nil)
	set(ComponentTabs, c.Tabs != nil)
	set(ComponentModal, c.Modal != nil)
	set(ComponentDivider, c.Divider != nil)
	set(ComponentButton, c.Button != nil)
	set(ComponentTextField, c.TextField != nil)
	set(ComponentCheckBox, c.CheckBox != nil)
	set(ComponentChoicePicker, c.ChoicePicker != nil)
	set(ComponentSlider, c.Slider != nil)
	set(ComponentDateTimeInput, c.DateTimeInput != nil)
	set(ComponentProgress, c.Progress != nil)
	if count != 1 {
		return ""
	}
	return typ
}

// MarshalJSON delegates standard component encoding to the Google SDK and then
// merges the local extension fields.
func (c Component) MarshalJSON() ([]byte, error) {
	var raw map[string]json.RawMessage
	switch c.ComponentType() {
	case "":
		return nil, fmt.Errorf("a2ui: component has no concrete type set")
	case ComponentProgress:
		raw = make(map[string]json.RawMessage)
		if err := appendCommonFields(raw, c); err != nil {
			return nil, err
		}
		componentType, err := json.Marshal(ComponentProgress)
		if err != nil {
			return nil, err
		}
		raw["component"] = componentType
		if c.Progress != nil && c.Progress.Value != nil {
			value, err := json.Marshal(*c.Progress.Value)
			if err != nil {
				return nil, err
			}
			raw["value"] = value
		}
		if c.Progress != nil && c.Progress.Max != nil {
			max, err := json.Marshal(*c.Progress.Max)
			if err != nil {
				return nil, err
			}
			raw["max"] = max
		}
	default:
		std, err := json.Marshal(c.toSDK())
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(std, &raw); err != nil {
			return nil, err
		}
	}
	if err := appendExtensionFields(raw, c); err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

// UnmarshalJSON decodes either a standard Google SDK component or the local
// Progress extension and restores the local extension fields.
func (c *Component) UnmarshalJSON(data []byte) error {
	var disc struct {
		Component string `json:"component"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}

	if disc.Component == ComponentProgress {
		type progressWire struct {
			ID            string                   `json:"id"`
			Accessibility *AccessibilityAttributes `json:"accessibility,omitempty"`
			Weight        *float64                 `json:"weight,omitempty"`
			Checks        []CheckRule              `json:"checks,omitempty"`
			Value         *DynamicNumber           `json:"value,omitempty"`
			Max           *float64                 `json:"max,omitempty"`
		}
		var wire progressWire
		if err := json.Unmarshal(data, &wire); err != nil {
			return err
		}
		*c = Component{
			ID:            wire.ID,
			Accessibility: wire.Accessibility,
			Weight:        wire.Weight,
			Checks:        wire.Checks,
			Progress: &ProgressComponent{
				Value: wire.Value,
				Max:   wire.Max,
			},
		}
	} else {
		var std googlea2ui.Component
		if err := json.Unmarshal(data, &std); err != nil {
			return err
		}
		*c = FromSDK(std)
	}

	var ext struct {
		Action        *Action  `json:"action,omitempty"`
		Spacing       *float64 `json:"spacing,omitempty"`
		Padding       *float64 `json:"padding,omitempty"`
		Strikethrough *bool    `json:"strikethrough,omitempty"`
	}
	if err := json.Unmarshal(data, &ext); err != nil {
		return err
	}
	c.Action = ext.Action
	c.Spacing = ext.Spacing
	c.Padding = ext.Padding
	c.Strikethrough = ext.Strikethrough
	return nil
}

func appendCommonFields(dst map[string]json.RawMessage, c Component) error {
	id, err := json.Marshal(c.ID)
	if err != nil {
		return err
	}
	dst["id"] = id
	if c.Accessibility != nil {
		v, err := json.Marshal(c.Accessibility)
		if err != nil {
			return err
		}
		dst["accessibility"] = v
	}
	if c.Weight != nil {
		v, err := json.Marshal(*c.Weight)
		if err != nil {
			return err
		}
		dst["weight"] = v
	}
	if len(c.Checks) > 0 {
		v, err := json.Marshal(c.Checks)
		if err != nil {
			return err
		}
		dst["checks"] = v
	}
	return nil
}

func appendExtensionFields(dst map[string]json.RawMessage, c Component) error {
	if c.Action != nil {
		v, err := json.Marshal(c.Action)
		if err != nil {
			return err
		}
		dst["action"] = v
	}
	if c.Spacing != nil {
		v, err := json.Marshal(*c.Spacing)
		if err != nil {
			return err
		}
		dst["spacing"] = v
	}
	if c.Padding != nil {
		v, err := json.Marshal(*c.Padding)
		if err != nil {
			return err
		}
		dst["padding"] = v
	}
	if c.Strikethrough != nil {
		v, err := json.Marshal(*c.Strikethrough)
		if err != nil {
			return err
		}
		dst["strikethrough"] = v
	}
	return nil
}

func (c Component) toSDK() googlea2ui.Component {
	return googlea2ui.Component{
		ID:            c.ID,
		Accessibility: c.Accessibility,
		Weight:        c.Weight,
		Checks:        c.Checks,
		Text:          c.Text,
		Image:         c.Image,
		Icon:          c.Icon,
		Video:         c.Video,
		AudioPlayer:   c.AudioPlayer,
		Row:           c.Row,
		Column:        c.Column,
		List:          c.List,
		Card:          c.Card,
		Tabs:          c.Tabs,
		Modal:         c.Modal,
		Divider:       c.Divider,
		Button:        c.Button,
		TextField:     c.TextField,
		CheckBox:      c.CheckBox,
		ChoicePicker:  c.ChoicePicker,
		Slider:        c.Slider,
		DateTimeInput: c.DateTimeInput,
	}
}

package swiftui

import (
	"fmt"
	"math"
	"sort"
)

type layoutKind int

const (
	layoutKindVStack layoutKind = iota + 1
	layoutKindHStack
	layoutKindZStack
	layoutKindVGrid
	layoutKindHGrid
	layoutKindFlow
)

// LayoutSpec describes the supported v1 runtime-switchable layout families.
//
// Curated surface.
//
// LayoutSpec is the public layout surface to use today when the same child set
// needs to switch between curated layouts at runtime. It intentionally models
// only the constrained layouts the Go surface can represent cleanly: stacks,
// curated grids, and a simple flow layout.
//
// LayoutSpec is not a bridge for SwiftUI's protocol-heavy custom layout
// system. Raw Layout, AnyLayout, and LayoutValueKey parity remain separate
// runtime-adapter design work.
//
// The zero value is not useful. Use one of the constructor helpers.
type LayoutSpec struct {
	kind         layoutKind
	spacing      float64
	hAlign       HorizontalAlignment
	vAlign       VerticalAlignment
	tracks       []GridItem
	minItemWidth float64
}

// ResponsiveLayoutStep maps a minimum width to a constrained layout spec.
//
// Curated surface.
//
// Steps are evaluated in ascending MinWidth order. The layout with the largest
// MinWidth not exceeding the available width wins.
type ResponsiveLayoutStep struct {
	MinWidth float64
	Layout   LayoutSpec
}

// ResponsiveLayout chooses between constrained layout specs using available width.
//
// Curated surface.
//
// This is the strongest supported adaptive layout surface today. It remains a
// curated Go-side chooser over the existing constrained layout families rather
// than exposing SwiftUI's protocol-heavy custom Layout system.
//
// The zero value is not useful. Use NewResponsiveLayout.
type ResponsiveLayout struct {
	fallback LayoutSpec
	steps    []ResponsiveLayoutStep
}

// LayoutProposal describes the inputs available to a constrained layout model.
//
// Curated surface.
//
// This is intentionally narrower than SwiftUI's layout protocol surface. It
// exposes only the values the Go runtime can use safely today: proposed width,
// proposed height, and child count.
type LayoutProposal struct {
	Width      float64
	Height     float64
	ChildCount int
}

// LayoutContext is kept as a compatibility alias for LayoutProposal.
//
// Curated surface.
type LayoutContext = LayoutProposal

// NewLayoutProposal constructs a constrained layout proposal.
func NewLayoutProposal(width, height float64, childCount int) LayoutProposal {
	if childCount < 0 {
		childCount = 0
	}
	return LayoutProposal{
		Width:      width,
		Height:     height,
		ChildCount: childCount,
	}
}

// WithWidth returns a copy of p with an updated width.
func (p LayoutProposal) WithWidth(width float64) LayoutProposal {
	p.Width = width
	return p
}

// WithHeight returns a copy of p with an updated height.
func (p LayoutProposal) WithHeight(height float64) LayoutProposal {
	p.Height = height
	return p
}

// WithChildCount returns a copy of p with an updated child count.
func (p LayoutProposal) WithChildCount(childCount int) LayoutProposal {
	if childCount < 0 {
		childCount = 0
	}
	p.ChildCount = childCount
	return p
}

// IsZero reports whether the proposal carries no useful sizing information.
func (p LayoutProposal) IsZero() bool {
	return p.Width == 0 && p.Height == 0 && p.ChildCount == 0
}

// String reports a compact human-readable summary of the proposal.
func (p LayoutProposal) String() string {
	return fmt.Sprintf("proposal(width=%.0f height=%.0f children=%d)", p.Width, p.Height, p.ChildCount)
}

// LayoutTag names a semantic child role for constrained tagged layouts.
//
// Curated surface.
//
// Tags are a concrete alternative to SwiftUI's protocol-heavy LayoutValueKey
// system. They let layout models react to child roles without exposing
// arbitrary metadata propagation or per-child placement callbacks.
type LayoutTag string

// PlacementMetadata carries the concrete placement keys used by tagged layout
// models.
//
// Curated surface.
//
// This is a fixed metadata shape, not an open-ended key/value protocol. Span
// and Priority are the only placement hints the runtime reasons about today.
// The zero value is allowed and means "no placement hints".
type PlacementMetadata struct {
	Span     int
	Priority int
}

// PlacementHint creates normalized placement metadata for a tagged child.
func PlacementHint(span, priority int) PlacementMetadata {
	if span < 1 {
		span = 1
	}
	if priority < 0 {
		priority = 0
	}
	return PlacementMetadata{Span: span, Priority: priority}
}

// TaggedView pairs one semantic tag with one child view.
//
// Curated surface.
type TaggedView struct {
	Tag       LayoutTag
	Placement PlacementMetadata
	View      Viewable
}

// TaggedLayoutContext describes the inputs available to a tagged layout model.
//
// Curated surface.
//
// It extends LayoutContext with an ordered tag list for the provided children.
// The runtime still resolves to one of the existing LayoutSpec families.
type TaggedLayoutContext struct {
	Width      float64
	Height     float64
	ChildCount int
	Tags       []LayoutTag
	Placements []PlacementMetadata
}

// Proposal returns the size/count portion of the tagged layout context.
func (ctx TaggedLayoutContext) Proposal() LayoutProposal {
	return NewLayoutProposal(ctx.Width, ctx.Height, ctx.ChildCount)
}

// LayoutModel resolves a constrained layout spec from a runtime proposal.
//
// Curated surface.
//
// It does not mirror SwiftUI's protocol-heavy custom layout system. Models are
// expected to choose one of the existing LayoutSpec families rather than place
// child views directly.
type LayoutModel interface {
	ResolveLayout(LayoutProposal) LayoutSpec
}

// LayoutModelFunc adapts a function into a constrained layout model.
//
// Curated surface.
type LayoutModelFunc func(LayoutProposal) LayoutSpec

// ResolveLayout implements LayoutModel.
func (f LayoutModelFunc) ResolveLayout(ctx LayoutProposal) LayoutSpec {
	if f == nil {
		return LayoutSpec{}
	}
	return f(ctx)
}

// TaggedLayoutModel resolves a constrained layout spec from runtime proposal
// and semantic child tags.
//
// Curated surface.
//
// Tagged layout is the next metadata-aware step toward layout parity. It stays
// concrete and lowers into the existing LayoutSpec families instead of exposing
// raw Layout or LayoutValueKey protocol machinery.
type TaggedLayoutModel interface {
	ResolveTaggedLayout(TaggedLayoutContext) LayoutSpec
}

// TaggedLayoutModelFunc adapts a function into a tagged layout model.
//
// Curated surface.
type TaggedLayoutModelFunc func(TaggedLayoutContext) LayoutSpec

// ResolveTaggedLayout implements TaggedLayoutModel.
func (f TaggedLayoutModelFunc) ResolveTaggedLayout(ctx TaggedLayoutContext) LayoutSpec {
	if f == nil {
		return LayoutSpec{}
	}
	return f(ctx)
}

// LayoutBreakpoint selects a layout when the available width reaches MinWidth.
//
// Curated surface.
type LayoutBreakpoint struct {
	MinWidth float64
	Layout   LayoutSpec
}

// ResponsiveLayoutModel chooses a LayoutSpec from ordered width breakpoints.
//
// Curated surface.
//
// The zero value is not useful. Use NewResponsiveLayoutModel.
type ResponsiveLayoutModel struct {
	fallback    LayoutSpec
	breakpoints []LayoutBreakpoint
}

// AdaptiveGridModel chooses between a fallback stack and a computed grid.
//
// Curated surface.
//
// It is intentionally constrained: the model computes grid tracks from width
// and child count, but still lowers into the existing curated grid surface.
// This is the strongest safe step toward custom layout parity without exposing
// raw placement callbacks or SwiftUI's Layout protocol.
type AdaptiveGridModel struct {
	Fallback     LayoutSpec
	MinItemWidth float64
	MinColumns   int
	MaxColumns   int
	Spacing      float64
}

// LayoutRule chooses a constrained layout when the context matches.
//
// Curated surface.
//
// Zero-valued bounds are ignored. Rules are evaluated in order, and the first
// matching rule wins.
type LayoutRule struct {
	MinWidth    float64
	MaxWidth    float64
	MinHeight   float64
	MaxHeight   float64
	MinChildren int
	MaxChildren int
	Layout      LayoutSpec
}

// RuleBasedLayoutModel resolves constrained layouts from ordered rules.
//
// Curated surface.
//
// This is a content-aware step toward layout/runtime parity: the model can
// react to viewport size and child count together while still lowering into
// existing curated layout families.
type RuleBasedLayoutModel struct {
	Fallback LayoutSpec
	Rules    []LayoutRule
}

// TaggedLayoutRule chooses a constrained layout when viewport and tag metadata
// match.
//
// Curated surface.
type TaggedLayoutRule struct {
	MinWidth     float64
	MaxWidth     float64
	MinHeight    float64
	MaxHeight    float64
	MinChildren  int
	MaxChildren  int
	RequireTags  []LayoutTag
	TagCounts    []TagCountConstraint
	LeadingTags  []LayoutTag
	TrailingTags []LayoutTag
	Layout       LayoutSpec
}

// TaggedRuleBasedLayoutModel resolves constrained layouts from ordered viewport
// and tag-aware rules.
//
// Curated surface.
type TaggedRuleBasedLayoutModel struct {
	Fallback LayoutSpec
	Rules    []TaggedLayoutRule
}

// PlacementPreset names one semantic tagged-layout preset.
//
// Curated surface.
type PlacementPreset int

const (
	PlacementPresetPrimarySecondary PlacementPreset = iota + 1
	PlacementPresetFeaturedGrid
	PlacementPresetDashboardBoard
)

// PlacementLayoutModel chooses one constrained layout family from semantic
// child roles and viewport width.
//
// Curated surface.
type PlacementLayoutModel struct {
	Preset       PlacementPreset
	Spacing      float64
	Breakpoint   float64
	MinItemWidth float64
}

// TagCountConstraint bounds how many children may use one semantic tag.
//
// Curated surface.
//
// Zero-valued bounds are ignored. This is a concrete count-based extension for
// tagged layouts, not an open-ended metadata protocol.
type TagCountConstraint struct {
	Tag LayoutTag
	Min int
	Max int
}

// NewResponsiveLayoutModel creates a constrained layout model that resolves to
// fallback until a breakpoint threshold is reached.
func NewResponsiveLayoutModel(fallback LayoutSpec, breakpoints ...LayoutBreakpoint) ResponsiveLayoutModel {
	model := ResponsiveLayoutModel{
		fallback:    fallback,
		breakpoints: append([]LayoutBreakpoint(nil), breakpoints...),
	}
	sort.Slice(model.breakpoints, func(i, j int) bool {
		return model.breakpoints[i].MinWidth < model.breakpoints[j].MinWidth
	})
	return model
}

// NewAdaptiveGridModel creates a constrained model that stacks when space is
// narrow and resolves to a computed VGrid when multiple columns fit.
func NewAdaptiveGridModel(fallback LayoutSpec, minItemWidth, spacing float64) AdaptiveGridModel {
	if minItemWidth <= 0 {
		minItemWidth = 180
	}
	return AdaptiveGridModel{
		Fallback:     fallback,
		MinItemWidth: minItemWidth,
		MinColumns:   1,
		Spacing:      spacing,
	}
}

// NewRuleBasedLayoutModel creates an ordered content-aware constrained layout model.
func NewRuleBasedLayoutModel(fallback LayoutSpec, rules ...LayoutRule) RuleBasedLayoutModel {
	return RuleBasedLayoutModel{
		Fallback: fallback,
		Rules:    append([]LayoutRule(nil), rules...),
	}
}

// NewTaggedRuleBasedLayoutModel creates an ordered tag-aware constrained
// layout model.
func NewTaggedRuleBasedLayoutModel(fallback LayoutSpec, rules ...TaggedLayoutRule) TaggedRuleBasedLayoutModel {
	return TaggedRuleBasedLayoutModel{
		Fallback: fallback,
		Rules:    append([]TaggedLayoutRule(nil), rules...),
	}
}

// NewPlacementLayoutModel creates one semantic tagged-layout preset model.
func NewPlacementLayoutModel(preset PlacementPreset, spacing float64) PlacementLayoutModel {
	if spacing == 0 {
		spacing = 12
	}
	return PlacementLayoutModel{
		Preset:       preset,
		Spacing:      spacing,
		Breakpoint:   680,
		MinItemWidth: 180,
	}
}

// AtLeastWidth creates one responsive-layout breakpoint.
func AtLeastWidth(minWidth float64, layout LayoutSpec) LayoutBreakpoint {
	if minWidth < 0 {
		minWidth = 0
	}
	return LayoutBreakpoint{MinWidth: minWidth, Layout: layout}
}

// Tagged creates one tagged child wrapper for a constrained tagged layout.
func Tagged(tag LayoutTag, v Viewable) TaggedView {
	return TaggedView{Tag: tag, Placement: PlacementMetadata{}, View: v}
}

// TaggedWithPlacement creates one tagged child wrapper with concrete placement
// metadata.
func TaggedWithPlacement(tag LayoutTag, placement PlacementMetadata, v Viewable) TaggedView {
	return TaggedView{Tag: tag, Placement: placement, View: v}
}

// AtLeastTagCount creates one lower-bounded tagged count constraint.
func AtLeastTagCount(tag LayoutTag, min int) TagCountConstraint {
	if min < 0 {
		min = 0
	}
	return TagCountConstraint{Tag: tag, Min: min}
}

// AtMostTagCount creates one upper-bounded tagged count constraint.
func AtMostTagCount(tag LayoutTag, max int) TagCountConstraint {
	if max < 0 {
		max = 0
	}
	return TagCountConstraint{Tag: tag, Max: max}
}

// TagCountBetween creates one bounded tagged count constraint.
func TagCountBetween(tag LayoutTag, min, max int) TagCountConstraint {
	if min < 0 {
		min = 0
	}
	if max < 0 {
		max = 0
	}
	return TagCountConstraint{Tag: tag, Min: min, Max: max}
}

// ResolveLayout chooses the best matching constrained layout.
func (m ResponsiveLayoutModel) ResolveLayout(ctx LayoutProposal) LayoutSpec {
	spec := m.fallback
	for _, breakpoint := range m.breakpoints {
		if ctx.Width < breakpoint.MinWidth {
			break
		}
		spec = breakpoint.Layout
	}
	return spec
}

// ResolveLayout chooses a computed grid track set from width and child count.
func (m AdaptiveGridModel) ResolveLayout(ctx LayoutProposal) LayoutSpec {
	spec := m.Fallback
	if spec.kind == 0 {
		spec = VStackLayout(HorizontalAlignmentLeading, m.Spacing)
	}
	columns := m.columnCount(ctx)
	if columns <= 1 {
		return spec
	}
	tracks := make([]GridItem, columns)
	for i := range tracks {
		tracks[i] = FlexibleGridItem(m.normalizedMinItemWidth(), 1000000)
	}
	return VGridLayout(tracks, m.Spacing)
}

// ResolveLayout chooses the first matching rule and falls back when none match.
func (m RuleBasedLayoutModel) ResolveLayout(ctx LayoutProposal) LayoutSpec {
	for _, rule := range m.Rules {
		if !rule.matches(ctx) {
			continue
		}
		if rule.Layout.kind != 0 {
			return rule.Layout
		}
		break
	}
	if m.Fallback.kind != 0 {
		return m.Fallback
	}
	return VStackLayout(HorizontalAlignmentLeading, 0)
}

// ResolveTaggedLayout chooses the first matching tag-aware rule and falls back
// when none match.
func (m TaggedRuleBasedLayoutModel) ResolveTaggedLayout(ctx TaggedLayoutContext) LayoutSpec {
	for _, rule := range m.Rules {
		if !rule.matches(ctx) {
			continue
		}
		if rule.Layout.kind != 0 {
			return rule.Layout
		}
		break
	}
	if m.Fallback.kind != 0 {
		return m.Fallback
	}
	return VStackLayout(HorizontalAlignmentLeading, 0)
}

// ResolveTaggedLayout chooses one semantic preset from viewport width and child
// role tags.
func (m PlacementLayoutModel) ResolveTaggedLayout(ctx TaggedLayoutContext) LayoutSpec {
	spacing := m.Spacing
	if spacing == 0 {
		spacing = 12
	}
	breakpoint := m.Breakpoint
	if breakpoint <= 0 {
		breakpoint = 680
	}
	minItemWidth := m.MinItemWidth
	if minItemWidth <= 0 {
		minItemWidth = 180
	}

	switch m.Preset {
	case PlacementPresetPrimarySecondary:
		if ctx.Width >= breakpoint && ctx.HasTag("primary") && ctx.HasTag("secondary") {
			return HStackLayout(VerticalAlignmentTop, spacing)
		}
		return VStackLayout(HorizontalAlignmentLeading, spacing)
	case PlacementPresetFeaturedGrid:
		if ctx.Width >= breakpoint && ctx.HasTag("hero") {
			if ctx.ChildCount >= 3 {
				return VGridLayout([]GridItem{
					FlexibleGridItem(minItemWidth, 1000000),
					FlexibleGridItem(minItemWidth, 1000000),
				}, spacing)
			}
			return HStackLayout(VerticalAlignmentTop, spacing)
		}
		if ctx.ChildCount >= 3 {
			return FlowLayout(minItemWidth, spacing)
		}
		return VStackLayout(HorizontalAlignmentLeading, spacing)
	case PlacementPresetDashboardBoard:
		if ctx.Width >= breakpoint && ctx.HasLeadingTags("primary", "detail") {
			primaryPlacement := ctx.PlacementAt(0)
			detailPlacement := ctx.PlacementAt(1)
			if ctx.ChildCount >= 4 && ctx.TagCount("meta") >= 2 && primaryPlacement.Span >= 2 && detailPlacement.Priority >= 1 {
				return VGridLayout([]GridItem{
					FlexibleGridItem(minItemWidth, 1000000),
					FlexibleGridItem(minItemWidth, 1000000),
				}, spacing)
			}
			return HStackLayout(VerticalAlignmentTop, spacing)
		}
		if ctx.ChildCount >= 3 {
			return FlowLayout(minItemWidth, spacing)
		}
		return VStackLayout(HorizontalAlignmentLeading, spacing)
	default:
		return VStackLayout(HorizontalAlignmentLeading, spacing)
	}
}

// VStackLayout creates a vertical stack layout spec.
func VStackLayout(alignment HorizontalAlignment, spacing float64) LayoutSpec {
	return LayoutSpec{
		kind:    layoutKindVStack,
		spacing: spacing,
		hAlign:  alignment,
	}
}

// HStackLayout creates a horizontal stack layout spec.
func HStackLayout(alignment VerticalAlignment, spacing float64) LayoutSpec {
	return LayoutSpec{
		kind:    layoutKindHStack,
		spacing: spacing,
		vAlign:  alignment,
	}
}

// ZStackLayout creates a layered stack layout spec.
func ZStackLayout() LayoutSpec {
	return LayoutSpec{kind: layoutKindZStack}
}

// VGridLayout creates a vertical curated grid layout spec.
func VGridLayout(columns []GridItem, spacing float64) LayoutSpec {
	return LayoutSpec{
		kind:    layoutKindVGrid,
		spacing: spacing,
		tracks:  append([]GridItem(nil), columns...),
	}
}

// HGridLayout creates a horizontal curated grid layout spec.
func HGridLayout(rows []GridItem, spacing float64) LayoutSpec {
	return LayoutSpec{
		kind:    layoutKindHGrid,
		spacing: spacing,
		tracks:  append([]GridItem(nil), rows...),
	}
}

// FlowLayout creates a width-driven flow layout spec.
func FlowLayout(minItemWidth, spacing float64) LayoutSpec {
	if minItemWidth <= 0 {
		minItemWidth = 180
	}
	return LayoutSpec{
		kind:         layoutKindFlow,
		spacing:      spacing,
		minItemWidth: minItemWidth,
	}
}

// NewResponsiveLayout creates a width-driven chooser over constrained layout specs.
func NewResponsiveLayout(fallback LayoutSpec, steps ...ResponsiveLayoutStep) ResponsiveLayout {
	out := ResponsiveLayout{
		fallback: fallback,
		steps:    append([]ResponsiveLayoutStep(nil), steps...),
	}
	sort.SliceStable(out.steps, func(i, j int) bool {
		return out.steps[i].MinWidth < out.steps[j].MinWidth
	})
	return out
}

// Apply renders children using the constrained layout spec.
func (s LayoutSpec) Apply(children ...Viewable) View {
	switch s.kind {
	case layoutKindVStack:
		if s.hAlign == HorizontalAlignmentLeading && s.spacing == 0 {
			return VStack(children...)
		}
		if s.spacing == 0 {
			return VStackAligned(s.hAlign, children...)
		}
		return VStackAlignedSpaced(s.hAlign, s.spacing, children...)
	case layoutKindHStack:
		if s.vAlign == VerticalAlignmentCenter && s.spacing == 0 {
			return HStack(children...)
		}
		if s.spacing == 0 {
			return HStackAligned(s.vAlign, children...)
		}
		return HStackAlignedSpaced(s.vAlign, s.spacing, children...)
	case layoutKindZStack:
		return ZStack(children...)
	case layoutKindVGrid:
		return LazyVGrid(s.tracks, s.spacing, children...)
	case layoutKindHGrid:
		return LazyHGrid(s.tracks, s.spacing, children...)
	case layoutKindFlow:
		return s.applyFlow(children...)
	default:
		return VStack(children...)
	}
}

// AnyLayout applies a constrained v1 layout spec to the provided children.
//
// This is the supported layout-erasure entry point for the curated v1 layout
// surface. It does not expose SwiftUI's custom Layout protocol machinery.
func AnyLayout(spec LayoutSpec, children ...Viewable) View {
	return spec.Apply(children...)
}

// Apply renders children using a width-responsive constrained layout chooser.
func (r ResponsiveLayout) Apply(children ...Viewable) View {
	return GeometryReader(func(width, _ float64) View {
		return r.layoutForWidth(width).Apply(children...)
	})
}

// AnyResponsiveLayout applies a responsive constrained layout to the provided children.
func AnyResponsiveLayout(layout ResponsiveLayout, children ...Viewable) View {
	return layout.Apply(children...)
}

// CustomLayout applies a constrained Go-native layout model to children.
//
// This is the strongest currently supported step toward custom layout authoring.
// The model resolves to one of the existing LayoutSpec families from viewport
// and child-count context; it does not expose raw SwiftUI Layout protocol parity.
func CustomLayout(model LayoutModel, children ...Viewable) View {
	if model == nil {
		return VStack(children...)
	}
	if len(children) == 0 {
		return EmptyView()
	}
	return GeometryReader(func(width, height float64) View {
		spec := model.ResolveLayout(NewLayoutProposal(width, height, len(children)))
		return spec.Apply(children...).MaxFrame(-1, 0)
	})
}

// CustomLayoutTagged applies a tagged Go-native layout model to tagged
// children.
//
// This is the metadata-aware constrained layout step. The model can react to
// semantic child roles and concrete placement hints while still lowering into
// the existing LayoutSpec families.
func CustomLayoutTagged(model TaggedLayoutModel, children ...TaggedView) View {
	if model == nil {
		views := taggedViews(children)
		return VStack(views...)
	}
	if len(children) == 0 {
		return EmptyView()
	}
	views := taggedViews(children)
	tags := taggedNames(children)
	placements := taggedPlacements(children)
	return GeometryReader(func(width, height float64) View {
		spec := model.ResolveTaggedLayout(TaggedLayoutContext{
			Width:      width,
			Height:     height,
			ChildCount: len(children),
			Tags:       tags,
			Placements: placements,
		})
		return spec.Apply(views...).MaxFrame(-1, 0)
	})
}

func (m AdaptiveGridModel) normalizedMinItemWidth() float64 {
	if m.MinItemWidth <= 0 {
		return 180
	}
	return m.MinItemWidth
}

func (m AdaptiveGridModel) columnCount(ctx LayoutProposal) int {
	width := ctx.Width
	if width <= 0 {
		return 1
	}
	minWidth := m.normalizedMinItemWidth()
	columns := int(math.Floor((width + m.Spacing) / (minWidth + m.Spacing)))
	if columns < 1 {
		columns = 1
	}
	if ctx.ChildCount > 0 && columns > ctx.ChildCount {
		columns = ctx.ChildCount
	}
	if m.MinColumns > 1 && columns < m.MinColumns {
		columns = m.MinColumns
	}
	if m.MaxColumns > 0 && columns > m.MaxColumns {
		columns = m.MaxColumns
	}
	if columns < 1 {
		return 1
	}
	return columns
}

func (r LayoutRule) matches(ctx LayoutProposal) bool {
	if r.MinWidth > 0 && ctx.Width < r.MinWidth {
		return false
	}
	if r.MaxWidth > 0 && ctx.Width > r.MaxWidth {
		return false
	}
	if r.MinHeight > 0 && ctx.Height < r.MinHeight {
		return false
	}
	if r.MaxHeight > 0 && ctx.Height > r.MaxHeight {
		return false
	}
	if r.MinChildren > 0 && ctx.ChildCount < r.MinChildren {
		return false
	}
	if r.MaxChildren > 0 && ctx.ChildCount > r.MaxChildren {
		return false
	}
	return true
}

func (r TaggedLayoutRule) matches(ctx TaggedLayoutContext) bool {
	if r.MinWidth > 0 && ctx.Width < r.MinWidth {
		return false
	}
	if r.MaxWidth > 0 && ctx.Width > r.MaxWidth {
		return false
	}
	if r.MinHeight > 0 && ctx.Height < r.MinHeight {
		return false
	}
	if r.MaxHeight > 0 && ctx.Height > r.MaxHeight {
		return false
	}
	if r.MinChildren > 0 && ctx.ChildCount < r.MinChildren {
		return false
	}
	if r.MaxChildren > 0 && ctx.ChildCount > r.MaxChildren {
		return false
	}
	for _, tag := range r.RequireTags {
		if !ctx.HasTag(tag) {
			return false
		}
	}
	for _, constraint := range r.TagCounts {
		count := ctx.TagCount(constraint.Tag)
		if constraint.Min > 0 && count < constraint.Min {
			return false
		}
		if constraint.Max > 0 && count > constraint.Max {
			return false
		}
	}
	if !ctx.HasLeadingTags(r.LeadingTags...) {
		return false
	}
	if !ctx.HasTrailingTags(r.TrailingTags...) {
		return false
	}
	return true
}

func (s LayoutSpec) applyFlow(children ...Viewable) View {
	if len(children) == 0 {
		return EmptyView()
	}
	return GeometryReader(func(width, _ float64) View {
		columns := s.flowColumns(width)
		rows := make([]Viewable, 0, (len(children)+columns-1)/columns)
		for start := 0; start < len(children); start += columns {
			end := start + columns
			if end > len(children) {
				end = len(children)
			}
			rows = append(rows, HStackSpaced(s.spacing, children[start:end]...).MaxFrame(-1, 0))
		}
		return VStackSpaced(s.spacing, rows...).MaxFrame(-1, 0)
	})
}

func (s LayoutSpec) flowColumns(width float64) int {
	if width <= 0 {
		return 1
	}
	minWidth := s.minItemWidth
	if minWidth <= 0 {
		minWidth = 180
	}
	columns := int(math.Floor((width + s.spacing) / (minWidth + s.spacing)))
	if columns < 1 {
		return 1
	}
	return columns
}

func (r ResponsiveLayout) layoutForWidth(width float64) LayoutSpec {
	layout := r.fallback
	for _, step := range r.steps {
		if width >= step.MinWidth {
			layout = step.Layout
		}
	}
	return layout
}

// HasTag reports whether the tagged context includes the given semantic role.
func (ctx TaggedLayoutContext) HasTag(tag LayoutTag) bool {
	return ctx.TagCount(tag) > 0
}

// TagCount reports how many tagged children use the given semantic role.
func (ctx TaggedLayoutContext) TagCount(tag LayoutTag) int {
	count := 0
	for _, have := range ctx.Tags {
		if have == tag {
			count++
		}
	}
	return count
}

// HasLeadingTags reports whether the ordered tag list starts with tags.
func (ctx TaggedLayoutContext) HasLeadingTags(tags ...LayoutTag) bool {
	if len(tags) == 0 {
		return true
	}
	if len(ctx.Tags) < len(tags) {
		return false
	}
	for i, tag := range tags {
		if ctx.Tags[i] != tag {
			return false
		}
	}
	return true
}

// HasTrailingTags reports whether the ordered tag list ends with tags.
func (ctx TaggedLayoutContext) HasTrailingTags(tags ...LayoutTag) bool {
	if len(tags) == 0 {
		return true
	}
	if len(ctx.Tags) < len(tags) {
		return false
	}
	start := len(ctx.Tags) - len(tags)
	for i, tag := range tags {
		if ctx.Tags[start+i] != tag {
			return false
		}
	}
	return true
}

func taggedViews(children []TaggedView) []Viewable {
	views := make([]Viewable, 0, len(children))
	for _, child := range children {
		views = append(views, child.View)
	}
	return views
}

func taggedNames(children []TaggedView) []LayoutTag {
	tags := make([]LayoutTag, 0, len(children))
	for _, child := range children {
		tags = append(tags, child.Tag)
	}
	return tags
}

func taggedPlacements(children []TaggedView) []PlacementMetadata {
	placements := make([]PlacementMetadata, 0, len(children))
	for _, child := range children {
		placements = append(placements, child.Placement)
	}
	return placements
}

// PlacementAt reports the placement metadata at index.
func (ctx TaggedLayoutContext) PlacementAt(index int) PlacementMetadata {
	if index < 0 || index >= len(ctx.Placements) {
		return PlacementMetadata{}
	}
	return ctx.Placements[index]
}

// PlacementAtTag reports the first placement metadata associated with tag.
func (ctx TaggedLayoutContext) PlacementAtTag(tag LayoutTag) PlacementMetadata {
	for i, have := range ctx.Tags {
		if have == tag {
			return ctx.PlacementAt(i)
		}
	}
	return PlacementMetadata{}
}

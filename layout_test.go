package swiftui

import (
	"reflect"
	"testing"
)

func TestLayoutSpecConstructors(t *testing.T) {
	tests := []struct {
		name string
		got  LayoutSpec
		want LayoutSpec
	}{
		{
			name: "vstack",
			got:  VStackLayout(HorizontalAlignmentTrailing, 12),
			want: LayoutSpec{kind: layoutKindVStack, hAlign: HorizontalAlignmentTrailing, spacing: 12},
		},
		{
			name: "hstack",
			got:  HStackLayout(VerticalAlignmentBottom, 8),
			want: LayoutSpec{kind: layoutKindHStack, vAlign: VerticalAlignmentBottom, spacing: 8},
		},
		{
			name: "zstack",
			got:  ZStackLayout(),
			want: LayoutSpec{kind: layoutKindZStack},
		},
		{
			name: "vgrid",
			got:  VGridLayout([]GridItem{FlexibleGridItem(120, 240)}, 10),
			want: LayoutSpec{kind: layoutKindVGrid, spacing: 10, tracks: []GridItem{FlexibleGridItem(120, 240)}},
		},
		{
			name: "hgrid",
			got:  HGridLayout([]GridItem{AdaptiveGridItem(80, 160)}, 6),
			want: LayoutSpec{kind: layoutKindHGrid, spacing: 6, tracks: []GridItem{AdaptiveGridItem(80, 160)}},
		},
		{
			name: "flow",
			got:  FlowLayout(220, 14),
			want: LayoutSpec{kind: layoutKindFlow, minItemWidth: 220, spacing: 14},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.kind != tt.want.kind || tt.got.hAlign != tt.want.hAlign || tt.got.vAlign != tt.want.vAlign || tt.got.spacing != tt.want.spacing || tt.got.minItemWidth != tt.want.minItemWidth || len(tt.got.tracks) != len(tt.want.tracks) {
				t.Fatalf("layout = %#v, want %#v", tt.got, tt.want)
			}
		})
	}
}

func TestTaggedView(t *testing.T) {
	child := Tagged("hero", Text("overview"))
	if got, want := child.Tag, LayoutTag("hero"); got != want {
		t.Fatalf("Tag = %q, want %q", got, want)
	}
	if child.View == nil {
		t.Fatal("View = nil, want non-nil")
	}
}

func TestTaggedViewPlacementMetadata(t *testing.T) {
	meta := PlacementHint(2, 3)
	child := TaggedWithPlacement("hero", meta, Text("overview"))
	if got, want := child.Tag, LayoutTag("hero"); got != want {
		t.Fatalf("Tag = %q, want %q", got, want)
	}
	if got, want := child.Placement, meta; got != want {
		t.Fatalf("Placement = %#v, want %#v", got, want)
	}
}

func TestFlowLayoutColumns(t *testing.T) {
	spec := FlowLayout(220, 12)
	tests := []struct {
		width float64
		want  int
	}{
		{width: 0, want: 1},
		{width: 180, want: 1},
		{width: 452, want: 2},
		{width: 700, want: 3},
	}
	for _, tc := range tests {
		if got := spec.flowColumns(tc.width); got != tc.want {
			t.Fatalf("flowColumns(%v) = %d, want %d", tc.width, got, tc.want)
		}
	}
}

func TestFlowLayoutDefaultsMinWidth(t *testing.T) {
	spec := FlowLayout(0, 12)
	if got, want := spec.minItemWidth, 180.0; got != want {
		t.Fatalf("minItemWidth = %v, want %v", got, want)
	}
}

func TestLayoutProposalHelpers(t *testing.T) {
	proposal := NewLayoutProposal(640, 320, 2)
	if proposal.IsZero() {
		t.Fatal("proposal should not be zero")
	}
	if got, want := proposal.WithWidth(720).WithHeight(400).WithChildCount(3), (LayoutProposal{Width: 720, Height: 400, ChildCount: 3}); got != want {
		t.Fatalf("proposal helpers = %#v, want %#v", got, want)
	}
	if got, want := (TaggedLayoutContext{Width: 640, Height: 320, ChildCount: 3}).Proposal(), proposal.WithChildCount(3); got != want {
		t.Fatalf("tagged proposal = %#v, want %#v", got, want)
	}
	if got, want := proposal.String(), "proposal(width=640 height=320 children=2)"; got != want {
		t.Fatalf("proposal.String() = %q, want %q", got, want)
	}
}

func TestPlacementHintNormalizes(t *testing.T) {
	if got, want := PlacementHint(0, -3), (PlacementMetadata{Span: 1, Priority: 0}); got != want {
		t.Fatalf("PlacementHint = %#v, want %#v", got, want)
	}
}

func TestTaggedLayoutContextPlacementAt(t *testing.T) {
	ctx := TaggedLayoutContext{
		Placements: []PlacementMetadata{
			PlacementHint(2, 1),
			PlacementHint(1, 0),
		},
	}
	if got, want := ctx.PlacementAt(0), (PlacementMetadata{Span: 2, Priority: 1}); got != want {
		t.Fatalf("PlacementAt(0) = %#v, want %#v", got, want)
	}
	if got, want := ctx.PlacementAt(1), (PlacementMetadata{Span: 1, Priority: 0}); got != want {
		t.Fatalf("PlacementAt(1) = %#v, want %#v", got, want)
	}
	if got, want := ctx.PlacementAt(2), (PlacementMetadata{}); got != want {
		t.Fatalf("PlacementAt(2) = %#v, want %#v", got, want)
	}
}

func TestGridLayoutCopiesTracks(t *testing.T) {
	columns := []GridItem{FlexibleGridItem(100, 200)}
	spec := VGridLayout(columns, 8)
	columns[0] = AdaptiveGridItem(40, 60)
	if got, want := spec.tracks[0].Kind, GridItemFlexible; got != want {
		t.Fatalf("tracks[0].Kind = %v, want %v", got, want)
	}
}

func TestLayoutSpecZeroValueFallsBackToVStack(t *testing.T) {
	var spec LayoutSpec
	view := spec.Apply(Text("fallback"))
	if view.ptr == 0 {
		t.Fatal("zero LayoutSpec should still render a fallback view")
	}
	view.Release()
}

func TestFlowLayoutNegativeMinWidthUsesDefault(t *testing.T) {
	spec := FlowLayout(-10, 5)
	if got, want := spec.minItemWidth, 180.0; got != want {
		t.Fatalf("minItemWidth = %v, want %v", got, want)
	}
}

func TestResponsiveLayoutChoosesLargestMatchingBreakpoint(t *testing.T) {
	layout := NewResponsiveLayout(
		VStackLayout(HorizontalAlignmentLeading, 8),
		ResponsiveLayoutStep{MinWidth: 480, Layout: HStackLayout(VerticalAlignmentTop, 10)},
		ResponsiveLayoutStep{MinWidth: 720, Layout: VGridLayout([]GridItem{FlexibleGridItem(160, 260), FlexibleGridItem(160, 260)}, 12)},
	)
	tests := []struct {
		width float64
		want  layoutKind
	}{
		{width: 320, want: layoutKindVStack},
		{width: 520, want: layoutKindHStack},
		{width: 760, want: layoutKindVGrid},
	}
	for _, tc := range tests {
		if got := layout.layoutForWidth(tc.width).kind; got != tc.want {
			t.Fatalf("layoutForWidth(%v) = %v, want %v", tc.width, got, tc.want)
		}
	}
}

func TestResponsiveLayoutSortsBreakpoints(t *testing.T) {
	layout := NewResponsiveLayout(
		ZStackLayout(),
		ResponsiveLayoutStep{MinWidth: 720, Layout: VGridLayout([]GridItem{FlexibleGridItem(100, 200)}, 8)},
		ResponsiveLayoutStep{MinWidth: 480, Layout: HStackLayout(VerticalAlignmentCenter, 6)},
	)
	if got, want := layout.steps[0].MinWidth, 480.0; got != want {
		t.Fatalf("steps[0].MinWidth = %v, want %v", got, want)
	}
	if got, want := layout.steps[1].MinWidth, 720.0; got != want {
		t.Fatalf("steps[1].MinWidth = %v, want %v", got, want)
	}
}

func TestResponsiveLayoutModelSortsBreakpoints(t *testing.T) {
	model := NewResponsiveLayoutModel(
		VStackLayout(HorizontalAlignmentLeading, 8),
		AtLeastWidth(900, HGridLayout([]GridItem{AdaptiveGridItem(120, 220)}, 12)),
		AtLeastWidth(480, HStackLayout(VerticalAlignmentTop, 10)),
	)
	got := make([]float64, 0, len(model.breakpoints))
	for _, breakpoint := range model.breakpoints {
		got = append(got, breakpoint.MinWidth)
	}
	want := []float64{480, 900}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("breakpoints = %v, want %v", got, want)
	}
}

func TestResponsiveLayoutModelResolveLayout(t *testing.T) {
	fallback := VStackLayout(HorizontalAlignmentLeading, 8)
	row := HStackLayout(VerticalAlignmentTop, 10)
	grid := VGridLayout([]GridItem{FlexibleGridItem(160, 240), FlexibleGridItem(160, 240)}, 12)
	model := NewResponsiveLayoutModel(
		fallback,
		AtLeastWidth(480, row),
		AtLeastWidth(960, grid),
	)
	tests := []struct {
		name  string
		width float64
		want  LayoutSpec
	}{
		{name: "fallback", width: 320, want: fallback},
		{name: "row", width: 640, want: row},
		{name: "grid", width: 1200, want: grid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ResolveLayout(NewLayoutProposal(tt.width, 0, 4)); !equalLayoutSpec(got, tt.want) {
				t.Fatalf("ResolveLayout(%v) = %#v, want %#v", tt.width, got, tt.want)
			}
		})
	}
}

func TestAtLeastWidthClampsNegativeWidths(t *testing.T) {
	breakpoint := AtLeastWidth(-20, ZStackLayout())
	if got, want := breakpoint.MinWidth, 0.0; got != want {
		t.Fatalf("MinWidth = %v, want %v", got, want)
	}
}

func TestCustomLayoutNilModelFallsBackToVStack(t *testing.T) {
	view := CustomLayout(nil, Text("fallback"))
	if view.ptr == 0 {
		t.Fatal("CustomLayout(nil) should still render a fallback view")
	}
	view.Release()
}

func TestCustomLayoutTaggedNilModelFallsBackToVStack(t *testing.T) {
	view := CustomLayoutTagged(nil, Tagged("hero", Text("fallback")))
	if view.ptr == 0 {
		t.Fatal("CustomLayoutTagged(nil) should still render a fallback view")
	}
	view.Release()
}

func TestLayoutModelFunc(t *testing.T) {
	model := LayoutModelFunc(func(ctx LayoutProposal) LayoutSpec {
		if ctx.ChildCount > 2 && ctx.Height >= 200 {
			return FlowLayout(200, 10)
		}
		return HStackLayout(VerticalAlignmentTop, 8)
	})
	got := model.ResolveLayout(NewLayoutProposal(500, 240, 3))
	if got.kind != layoutKindFlow {
		t.Fatalf("ResolveLayout kind = %v, want %v", got.kind, layoutKindFlow)
	}
}

func TestTaggedLayoutModelFunc(t *testing.T) {
	model := TaggedLayoutModelFunc(func(ctx TaggedLayoutContext) LayoutSpec {
		if ctx.HasTag("hero") && ctx.Width >= 640 {
			return HStackLayout(VerticalAlignmentTop, 12)
		}
		return VStackLayout(HorizontalAlignmentLeading, 10)
	})
	got := model.ResolveTaggedLayout(TaggedLayoutContext{
		Width:      720,
		ChildCount: 3,
		Tags:       []LayoutTag{"hero", "detail", "meta"},
	})
	if got.kind != layoutKindHStack {
		t.Fatalf("ResolveTaggedLayout kind = %v, want %v", got.kind, layoutKindHStack)
	}
}

func TestAdaptiveGridModelResolveLayout(t *testing.T) {
	model := NewAdaptiveGridModel(VStackLayout(HorizontalAlignmentLeading, 8), 180, 12)
	tests := []struct {
		name       string
		ctx        LayoutProposal
		wantKind   layoutKind
		wantTracks int
	}{
		{
			name:       "fallback stack",
			ctx:        NewLayoutProposal(220, 0, 4),
			wantKind:   layoutKindVStack,
			wantTracks: 0,
		},
		{
			name:       "two column grid",
			ctx:        NewLayoutProposal(420, 0, 4),
			wantKind:   layoutKindVGrid,
			wantTracks: 2,
		},
		{
			name:       "child count caps tracks",
			ctx:        NewLayoutProposal(900, 0, 2),
			wantKind:   layoutKindVGrid,
			wantTracks: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.ResolveLayout(tt.ctx)
			if got.kind != tt.wantKind {
				t.Fatalf("kind = %v, want %v", got.kind, tt.wantKind)
			}
			if got.kind == layoutKindVGrid && len(got.tracks) != tt.wantTracks {
				t.Fatalf("tracks = %d, want %d", len(got.tracks), tt.wantTracks)
			}
		})
	}
}

func TestAdaptiveGridModelRespectsColumnBounds(t *testing.T) {
	model := NewAdaptiveGridModel(VStackLayout(HorizontalAlignmentLeading, 8), 160, 10)
	model.MinColumns = 2
	model.MaxColumns = 3
	if got, want := model.columnCount(NewLayoutProposal(1200, 0, 8)), 3; got != want {
		t.Fatalf("columnCount(max) = %d, want %d", got, want)
	}
	if got, want := model.columnCount(NewLayoutProposal(200, 0, 8)), 2; got != want {
		t.Fatalf("columnCount(min) = %d, want %d", got, want)
	}
}

func TestLayoutRuleMatches(t *testing.T) {
	rule := LayoutRule{
		MinWidth:    400,
		MaxWidth:    900,
		MinHeight:   180,
		MaxHeight:   320,
		MinChildren: 2,
		MaxChildren: 4,
		Layout:      HStackLayout(VerticalAlignmentTop, 8),
	}
	if !rule.matches(NewLayoutProposal(640, 240, 3)) {
		t.Fatal("matches = false, want true")
	}
	if rule.matches(NewLayoutProposal(320, 240, 3)) {
		t.Fatal("matches(width) = true, want false")
	}
	if rule.matches(NewLayoutProposal(640, 240, 5)) {
		t.Fatal("matches(child count) = true, want false")
	}
	if rule.matches(NewLayoutProposal(640, 120, 3)) {
		t.Fatal("matches(height) = true, want false")
	}
}

func TestRuleBasedLayoutModelResolveLayout(t *testing.T) {
	fallback := VStackLayout(HorizontalAlignmentLeading, 10)
	compact := FlowLayout(160, 8)
	dense := VGridLayout([]GridItem{
		FlexibleGridItem(160, 260),
		FlexibleGridItem(160, 260),
	}, 12)
	model := NewRuleBasedLayoutModel(
		fallback,
		LayoutRule{MaxChildren: 2, Layout: HStackLayout(VerticalAlignmentTop, 8)},
		LayoutRule{MaxWidth: 420, MinChildren: 3, Layout: compact},
		LayoutRule{MinWidth: 700, MinHeight: 220, MinChildren: 3, Layout: dense},
	)
	tests := []struct {
		name string
		ctx  LayoutProposal
		want LayoutSpec
	}{
		{
			name: "small child count uses row",
			ctx:  NewLayoutProposal(800, 260, 2),
			want: HStackLayout(VerticalAlignmentTop, 8),
		},
		{
			name: "compact width uses flow",
			ctx:  NewLayoutProposal(360, 260, 4),
			want: compact,
		},
		{
			name: "wide width uses grid",
			ctx:  NewLayoutProposal(900, 260, 4),
			want: dense,
		},
		{
			name: "short viewport falls back",
			ctx:  NewLayoutProposal(900, 180, 4),
			want: fallback,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ResolveLayout(tt.ctx); !equalLayoutSpec(got, tt.want) {
				t.Fatalf("ResolveLayout(%#v) = %#v, want %#v", tt.ctx, got, tt.want)
			}
		})
	}
}

func TestTaggedLayoutContextTagCount(t *testing.T) {
	ctx := TaggedLayoutContext{
		Tags: []LayoutTag{"hero", "meta", "meta"},
	}
	if got, want := ctx.TagCount("meta"), 2; got != want {
		t.Fatalf("TagCount(meta) = %d, want %d", got, want)
	}
	if !ctx.HasTag("hero") {
		t.Fatal("HasTag(hero) = false, want true")
	}
	if ctx.HasTag("detail") {
		t.Fatal("HasTag(detail) = true, want false")
	}
}

func TestTaggedLayoutContextLeadingAndTrailingTags(t *testing.T) {
	ctx := TaggedLayoutContext{
		Tags: []LayoutTag{"hero", "detail", "meta", "meta"},
	}
	if !ctx.HasLeadingTags("hero", "detail") {
		t.Fatal("HasLeadingTags(hero, detail) = false, want true")
	}
	if ctx.HasLeadingTags("detail") {
		t.Fatal("HasLeadingTags(detail) = true, want false")
	}
	if !ctx.HasTrailingTags("meta", "meta") {
		t.Fatal("HasTrailingTags(meta, meta) = false, want true")
	}
	if ctx.HasTrailingTags("detail", "meta") {
		t.Fatal("HasTrailingTags(detail, meta) = true, want false")
	}
}

func TestTaggedLayoutRuleMatches(t *testing.T) {
	rule := TaggedLayoutRule{
		MinWidth:    480,
		MaxWidth:    920,
		MinChildren: 3,
		MaxChildren: 4,
		RequireTags: []LayoutTag{"hero", "detail"},
		TagCounts: []TagCountConstraint{
			AtLeastTagCount("meta", 1),
		},
		LeadingTags:  []LayoutTag{"hero"},
		TrailingTags: []LayoutTag{"meta"},
		Layout:       HStackLayout(VerticalAlignmentTop, 10),
	}
	if !rule.matches(TaggedLayoutContext{
		Width:      720,
		Height:     240,
		ChildCount: 3,
		Tags:       []LayoutTag{"hero", "detail", "meta"},
	}) {
		t.Fatal("matches = false, want true")
	}
	if rule.matches(TaggedLayoutContext{
		Width:      720,
		ChildCount: 3,
		Tags:       []LayoutTag{"hero", "meta", "meta"},
	}) {
		t.Fatal("matches(missing tag) = true, want false")
	}
	if rule.matches(TaggedLayoutContext{
		Width:      720,
		ChildCount: 2,
		Tags:       []LayoutTag{"hero", "detail"},
	}) {
		t.Fatal("matches(tag count) = true, want false")
	}
	if rule.matches(TaggedLayoutContext{
		Width:      720,
		ChildCount: 3,
		Tags:       []LayoutTag{"detail", "hero", "meta"},
	}) {
		t.Fatal("matches(leading tags) = true, want false")
	}
}

func TestTagCountConstraintHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  TagCountConstraint
		want TagCountConstraint
	}{
		{
			name: "at least clamps negative",
			got:  AtLeastTagCount("meta", -2),
			want: TagCountConstraint{Tag: "meta", Min: 0},
		},
		{
			name: "at most clamps negative",
			got:  AtMostTagCount("meta", -1),
			want: TagCountConstraint{Tag: "meta", Max: 0},
		},
		{
			name: "between preserves bounds",
			got:  TagCountBetween("detail", 1, 3),
			want: TagCountConstraint{Tag: "detail", Min: 1, Max: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("constraint = %#v, want %#v", tt.got, tt.want)
			}
		})
	}
}

func TestTaggedRuleBasedLayoutModelResolveLayout(t *testing.T) {
	fallback := VStackLayout(HorizontalAlignmentLeading, 10)
	row := HStackLayout(VerticalAlignmentTop, 12)
	grid := VGridLayout([]GridItem{
		FlexibleGridItem(180, 260),
		FlexibleGridItem(180, 260),
	}, 12)
	model := NewTaggedRuleBasedLayoutModel(
		fallback,
		TaggedLayoutRule{
			MinWidth:    900,
			MinChildren: 4,
			RequireTags: []LayoutTag{"hero"},
			TagCounts: []TagCountConstraint{
				AtLeastTagCount("meta", 2),
			},
			LeadingTags: []LayoutTag{"hero"},
			Layout:      grid,
		},
		TaggedLayoutRule{
			MinWidth:    640,
			RequireTags: []LayoutTag{"hero", "detail"},
			TagCounts: []TagCountConstraint{
				TagCountBetween("meta", 1, 2),
			},
			LeadingTags:  []LayoutTag{"hero", "detail"},
			TrailingTags: []LayoutTag{"meta"},
			Layout:       row,
		},
	)
	tests := []struct {
		name string
		ctx  TaggedLayoutContext
		want LayoutSpec
	}{
		{
			name: "fallback without detail tag",
			ctx: TaggedLayoutContext{
				Width:      720,
				ChildCount: 3,
				Tags:       []LayoutTag{"hero", "meta", "meta"},
			},
			want: fallback,
		},
		{
			name: "row with hero and detail",
			ctx: TaggedLayoutContext{
				Width:      720,
				ChildCount: 3,
				Tags:       []LayoutTag{"hero", "detail", "meta"},
			},
			want: row,
		},
		{
			name: "grid on wider viewport",
			ctx: TaggedLayoutContext{
				Width:      980,
				ChildCount: 4,
				Tags:       []LayoutTag{"hero", "detail", "meta", "meta"},
			},
			want: grid,
		},
		{
			name: "fallback when meta count too high for row",
			ctx: TaggedLayoutContext{
				Width:      720,
				ChildCount: 5,
				Tags:       []LayoutTag{"hero", "detail", "meta", "meta", "meta"},
			},
			want: fallback,
		},
		{
			name: "fallback when tag order does not match",
			ctx: TaggedLayoutContext{
				Width:      720,
				ChildCount: 4,
				Tags:       []LayoutTag{"detail", "hero", "meta", "meta"},
			},
			want: fallback,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ResolveTaggedLayout(tt.ctx); !equalLayoutSpec(got, tt.want) {
				t.Fatalf("ResolveTaggedLayout(%#v) = %#v, want %#v", tt.ctx, got, tt.want)
			}
		})
	}
}

func TestPlacementLayoutModelPrimarySecondary(t *testing.T) {
	model := NewPlacementLayoutModel(PlacementPresetPrimarySecondary, 14)
	tests := []struct {
		name string
		ctx  TaggedLayoutContext
		want layoutKind
	}{
		{
			name: "wide primary-secondary shell",
			ctx: TaggedLayoutContext{
				Width:      760,
				ChildCount: 3,
				Tags:       []LayoutTag{"primary", "secondary", "meta"},
			},
			want: layoutKindHStack,
		},
		{
			name: "narrow primary-secondary shell",
			ctx: TaggedLayoutContext{
				Width:      560,
				ChildCount: 3,
				Tags:       []LayoutTag{"primary", "secondary", "meta"},
			},
			want: layoutKindVStack,
		},
		{
			name: "missing tags falls back",
			ctx: TaggedLayoutContext{
				Width:      760,
				ChildCount: 3,
				Tags:       []LayoutTag{"detail", "meta", "meta"},
			},
			want: layoutKindVStack,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ResolveTaggedLayout(tt.ctx); got.kind != tt.want {
				t.Fatalf("kind = %v, want %v", got.kind, tt.want)
			}
		})
	}
}

func TestPlacementLayoutModelFeaturedGrid(t *testing.T) {
	model := NewPlacementLayoutModel(PlacementPresetFeaturedGrid, 12)
	got := model.ResolveTaggedLayout(TaggedLayoutContext{
		Width:      820,
		ChildCount: 3,
		Tags:       []LayoutTag{"hero", "detail", "meta"},
	})
	if got.kind != layoutKindVGrid {
		t.Fatalf("kind = %v, want %v", got.kind, layoutKindVGrid)
	}
}

func TestPlacementLayoutModelDashboardBoard(t *testing.T) {
	model := NewPlacementLayoutModel(PlacementPresetDashboardBoard, 14)
	model.Breakpoint = 700
	model.MinItemWidth = 200
	tests := []struct {
		name string
		ctx  TaggedLayoutContext
		want LayoutSpec
	}{
		{
			name: "wide board uses grid for primary detail meta",
			ctx: TaggedLayoutContext{
				Width:      880,
				ChildCount: 4,
				Tags:       []LayoutTag{"primary", "detail", "meta", "meta"},
				Placements: []PlacementMetadata{
					PlacementHint(2, 2),
					PlacementHint(1, 1),
					PlacementHint(1, 0),
					PlacementHint(1, 0),
				},
			},
			want: VGridLayout([]GridItem{
				FlexibleGridItem(200, 1000000),
				FlexibleGridItem(200, 1000000),
			}, 14),
		},
		{
			name: "wide board with weak placement uses row",
			ctx: TaggedLayoutContext{
				Width:      880,
				ChildCount: 4,
				Tags:       []LayoutTag{"primary", "detail", "meta", "meta"},
				Placements: []PlacementMetadata{
					PlacementHint(1, 0),
					PlacementHint(1, 0),
					PlacementHint(1, 0),
					PlacementHint(1, 0),
				},
			},
			want: HStackLayout(VerticalAlignmentTop, 14),
		},
		{
			name: "tag order mismatch falls back",
			ctx: TaggedLayoutContext{
				Width:      880,
				ChildCount: 4,
				Tags:       []LayoutTag{"detail", "primary", "meta", "meta"},
				Placements: []PlacementMetadata{
					PlacementHint(2, 2),
					PlacementHint(1, 1),
					PlacementHint(1, 0),
					PlacementHint(1, 0),
				},
			},
			want: FlowLayout(200, 14),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.ResolveTaggedLayout(tt.ctx); !equalLayoutSpec(got, tt.want) {
				t.Fatalf("ResolveTaggedLayout(%#v) = %#v, want %#v", tt.ctx, got, tt.want)
			}
		})
	}
}

func equalLayoutSpec(got, want LayoutSpec) bool {
	if got.kind != want.kind || got.hAlign != want.hAlign || got.vAlign != want.vAlign || got.spacing != want.spacing || got.minItemWidth != want.minItemWidth {
		return false
	}
	if len(got.tracks) != len(want.tracks) {
		return false
	}
	for i := range got.tracks {
		if got.tracks[i] != want.tracks[i] {
			return false
		}
	}
	return true
}

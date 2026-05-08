package swiftui

// NavigationStackPath rebuilds a navigation stack from the current route tokens.
//
// When the bridge is available, the stack body refreshes whenever the path
// revision changes. Without the bridge, it renders the current path once.
func NavigationStackPath(path *NavigationPathState, root func(path []string) View) View {
	if root == nil {
		return EmptyView()
	}
	if path == nil || path.RevisionState() == nil {
		return NavigationStack(root(nilSafePath(path)))
	}
	return DynamicView(path.RevisionState(), func(_ int) View {
		return NavigationStack(root(path.Get()))
	})
}

// NavigationSplitViewPreferredCompactColumn binds split navigation to a compact-column state.
func NavigationSplitViewPreferredCompactColumn(compact *CompactColumnState, sidebar View, detail View) View {
	if compact == nil || compact.VisibilityState() == nil {
		return NavigationSplitView(sidebar, detail)
	}
	return NavigationSplitViewVisibility(compact.VisibilityState(), sidebar, detail)
}

// NavigationSplitViewTriplePreferredCompactColumn binds a three-column split view to a compact-column state.
func NavigationSplitViewTriplePreferredCompactColumn(compact *CompactColumnState, sidebar View, content View, detail View) View {
	if compact == nil || compact.VisibilityState() == nil {
		return NavigationSplitViewTriple(sidebar, content, detail)
	}
	return NavigationSplitViewTripleVisibility(compact.VisibilityState(), sidebar, content, detail)
}

func nilSafePath(path *NavigationPathState) []string {
	if path == nil {
		return nil
	}
	return path.Get()
}

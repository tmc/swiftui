package swiftui

import "strings"

// Markdown creates a read-only markdown view.
//
// The supported block subset is paragraphs, headings, blockquotes, ordered
// lists, unordered lists, and fenced code blocks. Inline emphasis, code spans,
// and links are rendered by the platform markdown parser. Apply
// EnvironmentOpenURL to intercept tapped links. Tables, task lists, raw HTML,
// and images remain plain text.
func Markdown(source string) View {
	blocks := markdownBlocks(source)
	parts := make([]Viewable, 0, len(blocks))
	for _, block := range blocks {
		if block.code {
			parts = append(parts, markdownCodeBlock(block.lang, block.text))
			continue
		}
		parts = appendMarkdownProseViews(parts, block.text)
	}
	if len(parts) == 0 {
		trimmed := strings.TrimSpace(source)
		if trimmed == "" {
			return EmptyView()
		}
		parts = append(parts, markdownParagraphView(trimmed))
	}
	return VStackAlignedSpaced(HorizontalAlignmentLeading, 8, parts...).MaxFrame(-1, 0)
}

type markdownBlock struct {
	text string
	lang string
	code bool
}

func markdownBlocks(source string) []markdownBlock {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	var out []markdownBlock
	remainder := source
	for {
		open := strings.Index(remainder, "```")
		if open < 0 {
			if strings.TrimSpace(remainder) != "" {
				out = append(out, markdownBlock{text: remainder})
			}
			return out
		}
		before := remainder[:open]
		if strings.TrimSpace(before) != "" {
			out = append(out, markdownBlock{text: before})
		}
		afterOpen := remainder[open+3:]
		close := strings.Index(afterOpen, "```")
		if close < 0 {
			if strings.TrimSpace(afterOpen) != "" {
				out = append(out, markdownBlock{text: afterOpen})
			}
			return out
		}
		block := afterOpen[:close]
		lang := ""
		code := block
		if nl := strings.IndexByte(block, '\n'); nl >= 0 {
			lang = strings.TrimSpace(block[:nl])
			code = block[nl+1:]
		}
		code = strings.TrimSuffix(code, "\n")
		out = append(out, markdownBlock{text: code, lang: lang, code: true})
		remainder = afterOpen[close+3:]
	}
}

func appendMarkdownProseViews(dst []Viewable, text string) []Viewable {
	for _, paragraph := range markdownParagraphs(text) {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed == "" {
			continue
		}
		if level, heading, ok := markdownHeading(trimmed); ok {
			dst = append(dst, markdownHeadingView(level, heading))
			continue
		}
		if items := markdownListItems(trimmed); len(items) != 0 {
			dst = append(dst, markdownListView(items))
			continue
		}
		if quote, ok := markdownBlockquote(trimmed); ok {
			dst = append(dst, markdownQuoteView(quote))
			continue
		}
		dst = append(dst, markdownParagraphView(trimmed))
	}
	return dst
}

func markdownParagraphs(text string) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	parts := make([]string, 0, len(lines))
	current := make([]string, 0, len(lines))
	flush := func() {
		if len(current) == 0 {
			return
		}
		parts = append(parts, strings.Join(current, "\n"))
		current = current[:0]
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return parts
}

func markdownHeading(text string) (level int, body string, ok bool) {
	count := 0
	for count < len(text) && text[count] == '#' {
		count++
	}
	if count == 0 || count > 6 || len(text) <= count || text[count] != ' ' {
		return 0, "", false
	}
	body = strings.TrimSpace(text[count+1:])
	if body == "" {
		return 0, "", false
	}
	return count, body, true
}

func markdownHeadingView(level int, text string) View {
	font := FontBody
	switch {
	case level <= 2:
		font = FontTitle3
	case level == 3:
		font = FontHeadline
	}
	return markdownInlineText(text).
		Font(font).
		FontWeight(WeightSemibold).
		MaxFrame(-1, 0)
}

type markdownListItem struct {
	marker string
	text   string
}

func markdownListItems(text string) []markdownListItem {
	lines := strings.Split(text, "\n")
	items := make([]markdownListItem, 0, len(lines))
	for _, line := range lines {
		marker, body, ok := markdownListMarker(line)
		if !ok {
			return nil
		}
		items = append(items, markdownListItem{marker: marker, text: body})
	}
	return items
}

func markdownListMarker(line string) (marker, body string, ok bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "- "):
		return "•", strings.TrimSpace(trimmed[2:]), true
	case strings.HasPrefix(trimmed, "* "):
		return "•", strings.TrimSpace(trimmed[2:]), true
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] >= '0' && trimmed[i] <= '9' {
			continue
		}
		if i == 0 || trimmed[i] != '.' || len(trimmed) <= i+1 || trimmed[i+1] != ' ' {
			return "", "", false
		}
		return trimmed[:i+1], strings.TrimSpace(trimmed[i+2:]), true
	}
	return "", "", false
}

func markdownListView(items []markdownListItem) View {
	rows := make([]Viewable, 0, len(items))
	for _, item := range items {
		rows = append(rows, HStackAlignedSpaced(VerticalAlignmentTop, 8,
			Text(item.marker).
				Font(FontCallout).
				FontWeight(WeightSemibold).
				AsView(),
			markdownInlineText(item.text).
				Font(FontCallout).
				MaxFrame(-1, 0),
		))
	}
	return VStackAlignedSpaced(HorizontalAlignmentLeading, 6, rows...)
}

func markdownBlockquote(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	quote := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ">") {
			return "", false
		}
		quote = append(quote, strings.TrimSpace(strings.TrimPrefix(trimmed, ">")))
	}
	return strings.Join(quote, "\n"), len(quote) != 0
}

func markdownQuoteView(text string) View {
	return HStackAlignedSpaced(VerticalAlignmentTop, 10,
		Text(">").Font(FontHeadline).AsView(),
		markdownInlineText(text).
			Font(FontCallout).
			ForegroundStyleNamed("secondary").
			MaxFrame(-1, 0),
	)
}

func markdownParagraphView(text string) View {
	return markdownInlineText(text).Font(FontCallout).MaxFrame(-1, 0)
}

func markdownCodeBlock(lang, code string) View {
	children := make([]Viewable, 0, 2)
	if strings.TrimSpace(lang) != "" {
		children = append(children, Text(strings.TrimSpace(lang)).
			Font(FontCaption2).
			FontDesign(DesignMonospaced).
			ForegroundStyleNamed("secondary"))
	}
	children = append(children, SelectableText(code).
		Font(FontSystemDesign(12, WeightRegular, DesignMonospaced)).
		MaxFrame(-1, 0))
	return VStackAlignedSpaced(HorizontalAlignmentLeading, 8, children...).
		Padding(10).
		BackgroundRoundedRect(0.14, 0.15, 0.18, 0.98, 10).
		ClipRoundedRect(10)
}

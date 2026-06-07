package notion

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

func MarkdownToBlocks(markdown string) ([]Block, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.Linkify),
	)
	source := []byte(markdown)
	doc := md.Parser().Parse(text.NewReader(source))

	var blocks []Block
	if err := walkNode(doc, source, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func walkNode(n gast.Node, source []byte, blocks *[]Block) error {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		block, err := convertNode(child, source)
		if err != nil {
			return err
		}
		if block != nil {
			*blocks = append(*blocks, *block)
		}

		if child.HasChildren() {
			skip := child.Kind() == extensionast.KindTable ||
				child.Kind() == gast.KindBlockquote ||
				child.Kind() == gast.KindListItem
			if skip {
				continue
			}
			if err := walkNode(child, source, blocks); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertNode(n gast.Node, source []byte) (*Block, error) {
	switch n.Kind() {
	case gast.KindDocument:
		return nil, nil

	case gast.KindHeading:
		heading := n.(*gast.Heading)
		var btype BlockType
		switch heading.Level {
		case 1:
			btype = BlockHeading1
		case 2:
			btype = BlockHeading2
		case 3:
			btype = BlockHeading3
		default:
			btype = BlockHeading3
		}
		rt := extractRichText(n, source)
		return &Block{
			Type: btype,
			Heading1: func() *TextBlock {
				tb := &TextBlock{RichText: rt}
				if btype == BlockHeading1 { return tb }
				return nil
			}(),
			Heading2: func() *TextBlock {
				if btype == BlockHeading2 { return &TextBlock{RichText: rt} }
				return nil
			}(),
			Heading3: func() *TextBlock {
				if btype == BlockHeading3 { return &TextBlock{RichText: rt} }
				return nil
			}(),
		}, nil

	case gast.KindParagraph:
		rt := extractRichText(n, source)
		if len(rt) == 0 {
			return nil, nil
		}
		return &Block{Type: BlockParagraph, Paragraph: &TextBlock{RichText: rt}}, nil

	case gast.KindList:
		return nil, nil

	case gast.KindListItem:
		listItem := n.(*gast.ListItem)
		parent := n.Parent()
		if parent == nil {
			return nil, nil
		}
		list, isList := parent.(*gast.List)
		if !isList {
			block := &Block{Type: BlockBulletedListItem, BulletedItem: &TextBlock{RichText: extractRichText(n, source)}}
			if err := collectListChildren(n, source, block); err != nil {
				return nil, err
			}
			return block, nil
		}

		rt := extractRichText(n, source)
		cb := extractCheckbox(listItem, source)
		if cb != nil {
			block := &Block{Type: BlockToDo, ToDo: &ToDoBlock{RichText: rt, Checked: *cb}}
			if err := collectListChildren(n, source, block); err != nil {
				return nil, err
			}
			return block, nil
		}

		block := &Block{}
		if list.IsOrdered() {
			block.Type = BlockNumberedListItem
			block.NumberedItem = &TextBlock{RichText: rt}
		} else {
			block.Type = BlockBulletedListItem
			block.BulletedItem = &TextBlock{RichText: rt}
		}
		if err := collectListChildren(n, source, block); err != nil {
			return nil, err
		}
		return block, nil

	case gast.KindCodeBlock:
		content := string(n.Text(nil))
		rt := []RichText{{Type: "text", Text: &TextContent{Content: strings.TrimSuffix(content, "\n")}}}
		return &Block{Type: BlockCode, Code: &CodeBlock{RichText: rt, Language: ""}}, nil

	case gast.KindFencedCodeBlock:
		fcb := n.(*gast.FencedCodeBlock)
		lang := string(fcb.Language(source))
		content := string(n.Text(source))
		rt := []RichText{{Type: "text", Text: &TextContent{Content: strings.TrimSuffix(content, "\n")}}}
		return &Block{Type: BlockCode, Code: &CodeBlock{RichText: rt, Language: lang}}, nil

	case gast.KindBlockquote:
		rt := extractRichText(n, source)
		content := string(n.Text(source))
		if strings.HasPrefix(strings.TrimSpace(content), "[!") {
			return &Block{Type: BlockCallout, Callout: &CalloutBlock{RichText: rt, Icon: &Icon{Type: "emoji", Emoji: "💡"}}}, nil
		}
		return &Block{Type: BlockQuote, Quote: &TextBlock{RichText: rt}}, nil

	case gast.KindThematicBreak:
		return &Block{Type: BlockDivider, Divider: &DividerBlock{}}, nil

	case extensionast.KindTable:
		return convertTable(n, source), nil

	default:
		return nil, nil
	}
}

func convertTable(n gast.Node, source []byte) *Block {
	tbl := &TableBlock{
		Children: []Block{},
	}

	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		isHeader := row.Kind() == extensionast.KindTableHeader
		if isHeader {
			tbl.HasColumnHeader = true
		}

		if row.Kind() != extensionast.KindTableHeader && row.Kind() != extensionast.KindTableRow {
			continue
		}

		cells := [][]RichText{}
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if cell.Kind() != extensionast.KindTableCell {
				continue
			}
			rt := extractRichText(cell, source)
			cells = append(cells, rt)
		}

		if len(cells) > tbl.TableWidth {
			tbl.TableWidth = len(cells)
		}

		tbl.Children = append(tbl.Children, Block{
			Type:     BlockTableRow,
			TableRow: &TableRowBlock{Cells: cells},
		})
	}

	return &Block{Type: BlockTable, Table: tbl}
}

func collectListChildren(n gast.Node, source []byte, parent *Block) error {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != gast.KindList {
			continue
		}
		for li := child.FirstChild(); li != nil; li = li.NextSibling() {
			if li.Kind() != gast.KindListItem {
				continue
			}
			sub, err := convertNode(li, source)
			if err != nil {
				return err
			}
			if sub == nil {
				continue
			}
			parent.Children = append(parent.Children, *sub)
		}
	}
	return nil
}

func extractRichText(n gast.Node, source []byte) []RichText {
	var result []RichText
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		rt := convertInline(child, source)
		if rt != nil {
			result = append(result, rt...)
		} else if child.HasChildren() {
			inner := extractRichText(child, source)
			result = append(result, inner...)
		}
	}
	if len(result) == 0 {
		content := string(n.Text(source))
		if strings.TrimSpace(content) != "" {
			result = append(result, RichText{Type: "text", Text: &TextContent{Content: content}})
		} else {
			return []RichText{}
		}
	}
	return result
}

func convertInline(n gast.Node, source []byte) []RichText {
	switch n.Kind() {
	case gast.KindText:
		content := string(n.Text(source))
		if strings.TrimSpace(content) == "" {
			return nil
		}
		return []RichText{{Type: "text", Text: &TextContent{Content: content}}}

	case gast.KindEmphasis:
		em := n.(*gast.Emphasis)
		children := extractRichText(n, source)
		for i := range children {
			if children[i].Annotations == nil {
				children[i].Annotations = &Annotations{}
			}
			if em.Level == 1 {
				children[i].Annotations.Italic = true
			} else {
				children[i].Annotations.Bold = true
			}
		}
		return children

	case gast.KindCodeSpan:
		content := string(n.Text(source))
		return []RichText{{
			Type:        "text",
			Text:        &TextContent{Content: content},
			Annotations: &Annotations{Code: true},
		}}

	case gast.KindLink:
		link := n.(*gast.Link)
		children := extractRichText(n, source)
		url := string(link.Destination)
		for i := range children {
			children[i].Href = url
		}
		return children

	case extensionast.KindStrikethrough:
		children := extractRichText(n, source)
		for i := range children {
			if children[i].Annotations == nil {
				children[i].Annotations = &Annotations{}
			}
			children[i].Annotations.Strikethrough = true
		}
		return children

	default:
		return nil
	}
}

func extractCheckbox(listItem *gast.ListItem, source []byte) *bool {
	first := listItem.FirstChild()
	if first == nil {
		return nil
	}

	text := strings.TrimSpace(string(first.Text(source)))
	if strings.HasPrefix(text, "[ ] ") || strings.HasPrefix(text, "[x] ") || strings.HasPrefix(text, "[X] ") {
		checked := strings.HasPrefix(text, "[x]") || strings.HasPrefix(text, "[X]")
		return &checked
	}
	return nil
}

func BlocksToMarkdown(blocks []Block) (string, error) {
	var buf bytes.Buffer
	for _, block := range blocks {
		s, err := blockToMarkdown(block)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
		buf.WriteString("\n")
	}
	return buf.String(), nil
}

func blockToMarkdown(b Block) (string, error) {
	line, err := blockToMarkdownLine(b)
	if err != nil {
		return "", err
	}

	if len(b.Children) == 0 || b.Type == BlockTable || b.Type == BlockCode || b.Type == BlockQuote || b.Type == BlockCallout {
		return line, nil
	}

	var buf bytes.Buffer
	buf.WriteString(strings.TrimRight(line, "\n"))

	for _, child := range b.Children {
		childContent, err := blockToMarkdown(child)
		if err != nil {
			return "", err
		}
		lines := strings.Split(strings.TrimRight(childContent, "\n"), "\n")
		for _, l := range lines {
			buf.WriteString("\n  " + l)
		}
	}
	buf.WriteString("\n")
	return buf.String(), nil
}

func blockToMarkdownLine(b Block) (string, error) {
	switch b.Type {
	case BlockParagraph:
		if b.Paragraph == nil {
			return "\n", nil
		}
		return richTextToAnnotated(b.Paragraph.RichText) + "\n", nil
	case BlockHeading1:
		if b.Heading1 == nil {
			return "\n", nil
		}
		return "# " + richTextToAnnotated(b.Heading1.RichText) + "\n", nil
	case BlockHeading2:
		if b.Heading2 == nil {
			return "\n", nil
		}
		return "## " + richTextToAnnotated(b.Heading2.RichText) + "\n", nil
	case BlockHeading3:
		if b.Heading3 == nil {
			return "\n", nil
		}
		return "### " + richTextToAnnotated(b.Heading3.RichText) + "\n", nil
	case BlockBulletedListItem:
		if b.BulletedItem == nil {
			return "\n", nil
		}
		return "- " + richTextToAnnotated(b.BulletedItem.RichText) + "\n", nil
	case BlockNumberedListItem:
		if b.NumberedItem == nil {
			return "\n", nil
		}
		return "1. " + richTextToAnnotated(b.NumberedItem.RichText) + "\n", nil
	case BlockToDo:
		if b.ToDo == nil {
			return "\n", nil
		}
		prefix := "- [ ] "
		if b.ToDo.Checked {
			prefix = "- [x] "
		}
		return prefix + richTextToAnnotated(b.ToDo.RichText) + "\n", nil
	case BlockCode:
		if b.Code == nil {
			return "\n", nil
		}
		lang := b.Code.Language
		if lang == "" {
			lang = "text"
		}
		content := richTextToPlain(b.Code.RichText)
		return "```" + lang + "\n" + content + "\n```\n", nil
	case BlockQuote:
		if b.Quote == nil {
			return "\n", nil
		}
		return "> " + richTextToAnnotated(b.Quote.RichText) + "\n", nil
	case BlockCallout:
		if b.Callout == nil {
			return "\n", nil
		}
		return "> [!NOTE]\n> " + richTextToAnnotated(b.Callout.RichText) + "\n", nil
	case BlockDivider:
		return "---\n", nil
	case BlockTable:
		if len(b.Children) > 0 && (b.Table == nil || len(b.Table.Children) == 0) {
			if b.Table == nil {
				b.Table = &TableBlock{}
			}
			b.Table.Children = b.Children
			width := 0
			for _, row := range b.Table.Children {
				if row.TableRow != nil && len(row.TableRow.Cells) > width {
					width = len(row.TableRow.Cells)
				}
			}
			b.Table.TableWidth = width
		}
		return tableToMarkdown(b.Table), nil
	case BlockChildPage:
		if b.ChildPage != nil {
			return "[child page: " + b.ChildPage.Title + "]\n", nil
		}
		return "[child page]\n", nil
	default:
		return "", nil
	}
}

func tableToMarkdown(t *TableBlock) string {
	if len(t.Children) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for i, row := range t.Children {
		if row.TableRow == nil {
			continue
		}
		var cells []string
		for _, cell := range row.TableRow.Cells {
			cells = append(cells, richTextToAnnotated(cell))
		}
		buf.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		if i == 0 {
			var sep []string
			for range cells {
				sep = append(sep, "---")
			}
			buf.WriteString("| " + strings.Join(sep, " | ") + " |\n")
		}
	}
	return buf.String()
}

func richTextToPlain(rt []RichText) string {
	var b strings.Builder
	for _, r := range rt {
		if r.Text != nil {
			b.WriteString(r.Text.Content)
		}
	}
	return b.String()
}

func richTextToAnnotated(rt []RichText) string {
	var b strings.Builder
	for _, r := range rt {
		if r.Text != nil {
			content := r.Text.Content
			linkURL := r.Href
			if linkURL == "" && r.Text.Link != nil {
				linkURL = r.Text.Link.URL
			}

			annotated := content
			if r.Annotations != nil {
				if r.Annotations.Code {
					b.WriteString("`" + content + "`")
					continue
				}
				if r.Annotations.Bold && r.Annotations.Italic {
					annotated = "***" + annotated + "***"
				} else if r.Annotations.Bold {
					annotated = "**" + annotated + "**"
				} else if r.Annotations.Italic {
					annotated = "*" + annotated + "*"
				}
				if r.Annotations.Strikethrough {
					annotated = "~~" + annotated + "~~"
				}
			}
			if linkURL != "" {
				annotated = "[" + annotated + "](" + linkURL + ")"
			}
			b.WriteString(annotated)
		}
	}
	return b.String()
}

func MarkdownToBlocksRaw(markdown string) []Block {
	blocks, err := MarkdownToBlocks(markdown)
	if err != nil {
		return nil
	}
	return blocks
}

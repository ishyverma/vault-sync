package notion

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func MarkdownToBlocks(markdown string) ([]Block, error) {
	md := goldmark.New()
	source := []byte(markdown)
	doc := md.Parser().Parse(text.NewReader(source))

	var blocks []Block
	if err := walkNode(doc, source, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

func walkNode(n ast.Node, source []byte, blocks *[]Block) error {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		block, err := convertNode(child, source)
		if err != nil {
			return err
		}
		if block != nil {
			*blocks = append(*blocks, *block)
		}

		if child.HasChildren() {
			if err := walkNode(child, source, blocks); err != nil {
				return err
			}
		}
	}
	return nil
}

func convertNode(n ast.Node, source []byte) (*Block, error) {
	switch n.Kind() {
	case ast.KindDocument:
		return nil, nil

	case ast.KindHeading:
		heading := n.(*ast.Heading)
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

	case ast.KindParagraph:
		rt := extractRichText(n, source)
		if len(rt) == 0 {
			return nil, nil
		}
		return &Block{Type: BlockParagraph, Paragraph: &TextBlock{RichText: rt}}, nil

	case ast.KindList:
		return nil, nil

	case ast.KindListItem:
		listItem := n.(*ast.ListItem)
		parent := n.Parent()
		if parent == nil {
			return nil, nil
		}
		list, ok := parent.(*ast.List)
		if !ok {
			return &Block{Type: BlockBulletedListItem, BulletedItem: &TextBlock{RichText: extractRichText(n, source)}}, nil
		}

		rt := extractRichText(n, source)
		cb := extractCheckbox(listItem, source)
		if cb != nil {
			return &Block{Type: BlockToDo, ToDo: &ToDoBlock{RichText: rt, Checked: *cb}}, nil
		}

		if list.IsOrdered() {
			return &Block{Type: BlockNumberedListItem, NumberedItem: &TextBlock{RichText: rt}}, nil
		}
		return &Block{Type: BlockBulletedListItem, BulletedItem: &TextBlock{RichText: rt}}, nil

	case ast.KindCodeBlock:
		content := string(n.Text(nil))
		rt := []RichText{{Type: "text", Text: &TextContent{Content: strings.TrimSuffix(content, "\n")}}}
		return &Block{Type: BlockCode, Code: &CodeBlock{RichText: rt, Language: ""}}, nil

	case ast.KindFencedCodeBlock:
		fcb := n.(*ast.FencedCodeBlock)
		lang := string(fcb.Language(source))
		content := string(n.Text(source))
		rt := []RichText{{Type: "text", Text: &TextContent{Content: strings.TrimSuffix(content, "\n")}}}
		return &Block{Type: BlockCode, Code: &CodeBlock{RichText: rt, Language: lang}}, nil

	case ast.KindBlockquote:
		rt := extractRichText(n, source)
		content := string(n.Text(source))
		if strings.HasPrefix(strings.TrimSpace(content), "[!") {
			return &Block{Type: BlockCallout, Callout: &CalloutBlock{RichText: rt, Icon: &Icon{Type: "emoji", Emoji: "💡"}}}, nil
		}
		return &Block{Type: BlockQuote, Quote: &TextBlock{RichText: rt}}, nil

	case ast.KindThematicBreak:
		return &Block{Type: BlockDivider, Divider: &DividerBlock{}}, nil

	default:
		return nil, nil
	}
}

func extractRichText(n ast.Node, source []byte) []RichText {
	content := string(n.Text(source))
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return []RichText{{Type: "text", Text: &TextContent{Content: content}}}
}

func extractCheckbox(listItem *ast.ListItem, source []byte) *bool {
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
	switch b.Type {
	case BlockParagraph:
		return richTextToPlain(b.Paragraph.RichText) + "\n", nil
	case BlockHeading1:
		return "# " + richTextToPlain(b.Heading1.RichText) + "\n", nil
	case BlockHeading2:
		return "## " + richTextToPlain(b.Heading2.RichText) + "\n", nil
	case BlockHeading3:
		return "### " + richTextToPlain(b.Heading3.RichText) + "\n", nil
	case BlockBulletedListItem:
		return "- " + richTextToPlain(b.BulletedItem.RichText) + "\n", nil
	case BlockNumberedListItem:
		return "1. " + richTextToPlain(b.NumberedItem.RichText) + "\n", nil
	case BlockToDo:
		prefix := "- [ ] "
		if b.ToDo.Checked {
			prefix = "- [x] "
		}
		return prefix + richTextToPlain(b.ToDo.RichText) + "\n", nil
	case BlockCode:
		lang := b.Code.Language
		if lang == "" {
			lang = "text"
		}
		content := richTextToPlain(b.Code.RichText)
		return "```" + lang + "\n" + content + "\n```\n", nil
	case BlockQuote:
		return "> " + richTextToPlain(b.Quote.RichText) + "\n", nil
	case BlockCallout:
		return "> [!NOTE]\n> " + richTextToPlain(b.Callout.RichText) + "\n", nil
	case BlockDivider:
		return "---\n", nil
	case BlockTable:
		return tableToMarkdown(b.Table), nil
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
			cells = append(cells, richTextToPlain(cell))
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

func MarkdownToBlocksRaw(markdown string) []Block {
	blocks, err := MarkdownToBlocks(markdown)
	if err != nil {
		return nil
	}
	return blocks
}

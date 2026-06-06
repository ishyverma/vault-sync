package notion

type BlockType string

const (
	BlockParagraph       BlockType = "paragraph"
	BlockHeading1        BlockType = "heading_1"
	BlockHeading2        BlockType = "heading_2"
	BlockHeading3        BlockType = "heading_3"
	BlockBulletedListItem BlockType = "bulleted_list_item"
	BlockNumberedListItem BlockType = "numbered_list_item"
	BlockToDo            BlockType = "to_do"
	BlockCode            BlockType = "code"
	BlockQuote           BlockType = "quote"
	BlockCallout         BlockType = "callout"
	BlockDivider         BlockType = "divider"
	BlockTable           BlockType = "table"
	BlockTableRow        BlockType = "table_row"
	BlockImage           BlockType = "image"
)

type Block struct {
	ID           string       `json:"id,omitempty"`
	Object       string       `json:"object,omitempty"`
	Type         BlockType    `json:"type"`
	Paragraph    *TextBlock   `json:"paragraph,omitempty"`
	Heading1     *TextBlock   `json:"heading_1,omitempty"`
	Heading2     *TextBlock   `json:"heading_2,omitempty"`
	Heading3     *TextBlock   `json:"heading_3,omitempty"`
	BulletedItem *TextBlock   `json:"bulleted_list_item,omitempty"`
	NumberedItem *TextBlock   `json:"numbered_list_item,omitempty"`
	ToDo         *ToDoBlock   `json:"to_do,omitempty"`
	Code         *CodeBlock   `json:"code,omitempty"`
	Quote        *TextBlock   `json:"quote,omitempty"`
	Callout      *CalloutBlock `json:"callout,omitempty"`
	Divider      *DividerBlock `json:"divider,omitempty"`
	Table        *TableBlock  `json:"table,omitempty"`
	TableRow     *TableRowBlock `json:"table_row,omitempty"`
	Children     []Block      `json:"children,omitempty"`
}

type TextBlock struct {
	RichText []RichText `json:"rich_text"`
	Color    string     `json:"color,omitempty"`
	IsToggle bool       `json:"is_toggle,omitempty"`
}

type ToDoBlock struct {
	RichText []RichText `json:"rich_text"`
	Checked  bool       `json:"checked"`
	Color    string     `json:"color,omitempty"`
}

type CodeBlock struct {
	RichText []RichText `json:"rich_text"`
	Language string     `json:"language"`
}

type CalloutBlock struct {
	RichText []RichText `json:"rich_text"`
	Icon     *Icon      `json:"icon,omitempty"`
	Color    string     `json:"color,omitempty"`
}

type DividerBlock struct{}

type TableBlock struct {
	TableWidth int          `json:"table_width"`
	HasColumnHeader bool   `json:"has_column_header"`
	HasRowHeader    bool   `json:"has_row_header"`
	Children    []Block     `json:"children"`
}

type TableRowBlock struct {
	Cells [][]RichText `json:"cells"`
}

type Icon struct {
	Type  string `json:"type"`
	Emoji string `json:"emoji,omitempty"`
}

type RichText struct {
	Type        string       `json:"type"`
	Text        *TextContent `json:"text,omitempty"`
	Mention     *Mention     `json:"mention,omitempty"`
	Equation    *Equation    `json:"equation,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
	Href        string       `json:"href,omitempty"`
}

type TextContent struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

type Mention struct {
	Type string     `json:"type"`
	Page *PageRef   `json:"page,omitempty"`
}

type PageRef struct {
	ID string `json:"id"`
}

type Link struct {
	URL string `json:"url"`
}

type Equation struct {
	Expression string `json:"expression"`
}

type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

type Page struct {
	ID         string              `json:"id"`
	Object     string              `json:"object"`
	CreatedAt  string              `json:"created_time"`
	ModifiedAt string              `json:"last_edited_time"`
	Archived   bool                `json:"archived"`
	URL        string              `json:"url"`
	Properties map[string]Property `json:"properties"`
	Parent     *Parent             `json:"parent,omitempty"`
}

type Parent struct {
	Type       string `json:"type"`
	PageID     string `json:"page_id,omitempty"`
	DatabaseID string `json:"database_id,omitempty"`
}

type Property struct {
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	Title   []RichText  `json:"title,omitempty"`
	RichText []RichText `json:"rich_text,omitempty"`
	Number  *Number     `json:"number,omitempty"`
	Select  *Select     `json:"select,omitempty"`
	MultiSelect []Select `json:"multi_select,omitempty"`
	Date    *DateValue  `json:"date,omitempty"`
	Checkbox *bool       `json:"checkbox,omitempty"`
	URL     string      `json:"url,omitempty"`
	Status  *Select     `json:"status,omitempty"`
}

type Number struct {
	Number float64 `json:"number,omitempty"`
}

type Select struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

type DateValue struct {
	Start string `json:"start"`
	End   string `json:"end,omitempty"`
}

type CreatePageRequest struct {
	Parent     Parent              `json:"parent"`
	Properties map[string]Property `json:"properties"`
	Children   []Block             `json:"children,omitempty"`
}

type UpdatePageRequest struct {
	Properties map[string]Property `json:"properties,omitempty"`
	Archived   *bool               `json:"archived,omitempty"`
}

type UpdateBlockRequest struct {
	Archived *bool `json:"archived,omitempty"`
}

type AppendBlocksRequest struct {
	Children []Block `json:"children"`
}

type ListBlocksResponse struct {
	Results    []Block `json:"results"`
	HasMore    bool    `json:"has_more"`
	NextCursor string  `json:"next_cursor"`
}

type SearchRequest struct {
	Query      string `json:"query"`
	Sort       *Sort  `json:"sort,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	StartCursor string `json:"start_cursor,omitempty"`
}

type Sort struct {
	Direction string `json:"direction"`
	Timestamp string `json:"timestamp"`
}

type SearchResponse struct {
	Results    []Page  `json:"results"`
	HasMore    bool    `json:"has_more"`
	NextCursor string  `json:"next_cursor"`
}

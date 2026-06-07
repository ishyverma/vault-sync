package notion

type BlockType string

const (
	BlockParagraph        BlockType = "paragraph"
	BlockHeading1         BlockType = "heading_1"
	BlockHeading2         BlockType = "heading_2"
	BlockHeading3         BlockType = "heading_3"
	BlockBulletedListItem BlockType = "bulleted_list_item"
	BlockNumberedListItem BlockType = "numbered_list_item"
	BlockToDo             BlockType = "to_do"
	BlockCode             BlockType = "code"
	BlockQuote            BlockType = "quote"
	BlockCallout          BlockType = "callout"
	BlockDivider          BlockType = "divider"
	BlockTable            BlockType = "table"
	BlockTableRow         BlockType = "table_row"
	BlockImage            BlockType = "image"
	BlockBookmark         BlockType = "bookmark"
	BlockEquation         BlockType = "equation"
	BlockEmbed            BlockType = "embed"
	BlockVideo            BlockType = "video"
	BlockFile             BlockType = "file"
	BlockPDF              BlockType = "pdf"
	BlockChildPage        BlockType = "child_page"
	BlockChildDatabase    BlockType = "child_database"
	BlockBreadcrumb       BlockType = "breadcrumb"
	BlockColumnList       BlockType = "column_list"
	BlockColumn           BlockType = "column"
	BlockLinkPreview      BlockType = "link_preview"
	BlockLinkToPage       BlockType = "link_to_page"
	BlockSyncedBlock      BlockType = "synced_block"
	BlockTemplate         BlockType = "template"
	BlockTableOfContents  BlockType = "table_of_contents"
)

type Block struct {
	ID           string          `json:"id,omitempty"`
	Object       string          `json:"object,omitempty"`
	Type         BlockType       `json:"type"`
	HasChildren  bool            `json:"has_children,omitempty"`
	Archived     bool            `json:"archived,omitempty"`
	Paragraph    *TextBlock      `json:"paragraph,omitempty"`
	Heading1     *TextBlock      `json:"heading_1,omitempty"`
	Heading2     *TextBlock      `json:"heading_2,omitempty"`
	Heading3     *TextBlock      `json:"heading_3,omitempty"`
	BulletedItem *TextBlock      `json:"bulleted_list_item,omitempty"`
	NumberedItem *TextBlock      `json:"numbered_list_item,omitempty"`
	ToDo         *ToDoBlock      `json:"to_do,omitempty"`
	Code         *CodeBlock      `json:"code,omitempty"`
	Quote        *TextBlock      `json:"quote,omitempty"`
	Callout      *CalloutBlock   `json:"callout,omitempty"`
	Divider      *DividerBlock   `json:"divider,omitempty"`
	Table        *TableBlock     `json:"table,omitempty"`
	TableRow     *TableRowBlock  `json:"table_row,omitempty"`
	Image        *FileBlock      `json:"image,omitempty"`
	Video        *FileBlock      `json:"video,omitempty"`
	File         *FileBlock      `json:"file,omitempty"`
	PDF          *FileBlock      `json:"pdf,omitempty"`
	Bookmark     *BookmarkBlock  `json:"bookmark,omitempty"`
	Embed        *EmbedBlock     `json:"embed,omitempty"`
	Equation     *EquationBlock  `json:"equation,omitempty"`
	ChildPage    *ChildPageBlock `json:"child_page,omitempty"`
	ChildDatabase *ChildPageBlock `json:"child_database,omitempty"`
	Breadcrumb   *BreadcrumbBlock `json:"breadcrumb,omitempty"`
	ColumnList   *ColumnListBlock `json:"column_list,omitempty"`
	Column       *ColumnBlock     `json:"column,omitempty"`
	LinkPreview  *LinkPreviewBlock `json:"link_preview,omitempty"`
	LinkToPage   *LinkToPageBlock `json:"link_to_page,omitempty"`
	SyncedBlock  *SyncedBlock     `json:"synced_block,omitempty"`
	Template     *TemplateBlock   `json:"template,omitempty"`
	TableOfContents *TableOfContentsBlock `json:"table_of_contents,omitempty"`
	Children     []Block         `json:"children,omitempty"`
}

type ChildPageBlock struct {
	Title string `json:"title"`
}

type FileBlock struct {
	Type     string       `json:"type"`
	External *FileURL     `json:"external,omitempty"`
	File     *FileURL     `json:"file,omitempty"`
	Caption  []RichText   `json:"caption,omitempty"`
}

type FileURL struct {
	URL        string `json:"url"`
	ExpiryTime string `json:"expiry_time,omitempty"`
}

type BookmarkBlock struct {
	URL     string    `json:"url"`
	Caption []RichText `json:"caption,omitempty"`
}

type EmbedBlock struct {
	URL string `json:"url"`
}

type EquationBlock struct {
	Expression string `json:"expression"`
}

type BreadcrumbBlock struct{}

type ColumnListBlock struct{}

type ColumnBlock struct{}

type LinkPreviewBlock struct {
	URL string `json:"url"`
}

type LinkToPageBlock struct {
	Type       string `json:"type"`
	PageID     string `json:"page_id,omitempty"`
	DatabaseID string `json:"database_id,omitempty"`
}

type SyncedBlock struct {
	SyncedFrom *SyncedFrom `json:"synced_from,omitempty"`
}

type SyncedFrom struct {
	Type    string `json:"type"`
	BlockID string `json:"block_id,omitempty"`
}

type TemplateBlock struct {
	RichText []RichText `json:"rich_text"`
}

type TableOfContentsBlock struct {
	Color string `json:"color,omitempty"`
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
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Color         string `json:"color,omitempty"`
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

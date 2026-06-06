package notion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
)

type Connector struct {
	client      *Client
	token       string
	targetPageID string
	databaseID  string
	notesDir    string
}

func NewConnector(token, targetPageID, databaseID, notesDir string) *Connector {
	return &Connector{
		token:       token,
		targetPageID: targetPageID,
		databaseID:  databaseID,
		notesDir:    notesDir,
	}
}

func (c *Connector) Name() string {
	return "notion"
}

func (c *Connector) Connect() error {
	if c.token == "" {
		return fmt.Errorf("notion token not configured: run 'vault connect notion --token <key>'")
	}
	c.client = NewClient(c.token)
	return c.client.Status()
}

func (c *Connector) Status() (bool, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return false, err
		}
	}
	if err := c.client.Status(); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Connector) Push(note *storage.Note, content string) (string, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	_, body, _ := core.ParseFrontmatter(content)
	if body == "" {
		body = content
	}

	blocks, err := MarkdownToBlocks(body)
	if err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}

	properties := buildProperties(note)

	// Look up existing remote_id via sync_state
	// If caller already resolved remoteID we'd use it; here we rely on the engine to pass context
	// For now: always create a new page under the target page
	if c.targetPageID == "" {
		return "", fmt.Errorf("notion target page not configured: set target_page_id in config")
	}

	page, err := c.client.CreatePage(&CreatePageRequest{
		Parent:     Parent{Type: "page_id", PageID: c.targetPageID},
		Properties: properties,
		Children:   blocks,
	})
	if err != nil {
		return "", fmt.Errorf("create notion page: %w", err)
	}

	return page.ID, nil
}

func (c *Connector) Pull(remoteID string) (string, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	page, err := c.client.GetPage(remoteID)
	if err != nil {
		return "", fmt.Errorf("get notion page: %w", err)
	}

	blocks, err := c.client.GetBlocks(remoteID)
	if err != nil {
		return "", fmt.Errorf("get notion blocks: %w", err)
	}

	body, err := BlocksToMarkdown(blocks)
	if err != nil {
		return "", fmt.Errorf("convert blocks to markdown: %w", err)
	}

	fm := propertiesToFrontmatter(page.Properties)
	content := core.BuildNoteContent(fm, body)

	return content, nil
}

func (c *Connector) Delete(remoteID string) error {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return err
		}
	}
	return c.client.DeletePage(remoteID)
}

func buildProperties(note *storage.Note) map[string]Property {
	props := map[string]Property{
		"title": {
			Type:  "title",
			Title: []RichText{{Type: "text", Text: &TextContent{Content: note.Title}}},
		},
	}

	if len(note.Tags) > 0 {
		opts := make([]Select, len(note.Tags))
		for i, tag := range note.Tags {
			opts[i] = Select{Name: tag}
		}
		props["Tags"] = Property{Type: "multi_select", MultiSelect: opts}
	}

	if !note.CreatedAt.IsZero() {
		props["Created"] = Property{
			Type: "date",
			Date: &DateValue{Start: note.CreatedAt.Format("2006-01-02")},
		}
	}

	return props
}

func frontmatterFromDisk(notesDir, filename string) (string, error) {
	path := filepath.Join(notesDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func propertiesToFrontmatter(props map[string]Property) core.Frontmatter {
	var fm core.Frontmatter
	for _, p := range props {
		switch p.Type {
		case "title":
			fm.Title = richTextToPlain(p.Title)
		case "multi_select":
			for _, s := range p.MultiSelect {
				fm.Tags = append(fm.Tags, s.Name)
			}
		case "date":
			if p.Date != nil {
				fm.Date = p.Date.Start
			}
		}
	}
	return fm
}

func richTextToAnnotated(rt []RichText) string {
	var b strings.Builder
	for _, r := range rt {
		if r.Text != nil {
			content := r.Text.Content
			if r.Annotations != nil {
				if r.Annotations.Code {
					content = "`" + content + "`"
				}
				if r.Annotations.Bold {
					content = "**" + content + "**"
				}
				if r.Annotations.Italic {
					content = "*" + content + "*"
				}
			}
			b.WriteString(content)
		}
	}
	return b.String()
}

package notion

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ishyverma/vault-sync/internal/connectors"
	"github.com/ishyverma/vault-sync/internal/core"
	"github.com/ishyverma/vault-sync/internal/storage"
)

var ErrNotFound = errors.New("notion: page not found")

var _ connectors.Connector = (*Connector)(nil)

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

func (c *Connector) Push(note *storage.Note, content string, remoteID string) (string, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	fm, body, fmErr := core.ParseFrontmatter(content)
	if fmErr != nil {
		body = content
	} else if body == "" {
		body = content
	}

	body = embedTags(body, fm.Tags)

	// Use frontmatter title if available, fall back to DB title
	title := note.Title
	if fm.Title != "" {
		title = fm.Title
	}

	blocks, err := MarkdownToBlocks(body)
	if err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}

	properties := buildProperties(title)

	if c.targetPageID == "" {
		return "", fmt.Errorf("notion target page not configured: set target_page_id in config")
	}

	// Update existing page
	if remoteID != "" {
		updated, err := c.client.UpdatePage(remoteID, &UpdatePageRequest{Properties: properties})
		if err != nil {
			if strings.Contains(err.Error(), "archived") {
				// Page was archived in Notion. Fall through to create a new one.
				remoteID = ""
			} else {
				return "", fmt.Errorf("update notion page: %w", err)
			}
		} else {
			_ = updated
			// Replace blocks: delete all existing, then append new ones
			if err := c.replaceBlocks(remoteID, blocks); err != nil {
				return "", fmt.Errorf("replace blocks: %w", err)
			}
			return remoteID, nil
		}
	}

	// Create new page (without children — append them separately)
	page, err := c.client.CreatePage(&CreatePageRequest{
		Parent:     Parent{Type: "page_id", PageID: c.targetPageID},
		Properties: properties,
	})
	if err != nil {
		return "", fmt.Errorf("create notion page: %w", err)
	}

	if len(blocks) > 0 {
		if err := c.appendBlocksRecursive(page.ID, blocks); err != nil {
			return "", fmt.Errorf("append blocks: %w", err)
		}
	}

	return page.ID, nil
}

func embedTags(body string, tags []string) string {
	if len(tags) == 0 {
		return body
	}
	var b strings.Builder
	b.WriteString("**Tags:** ")
	for i, tag := range tags {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("`" + tag + "`")
	}
	b.WriteString("\n\n")
	b.WriteString(body)
	return b.String()
}

func (c *Connector) replaceBlocks(pageID string, newBlocks []Block) error {
	existing, err := c.client.GetBlocks(pageID)
	if err != nil {
		return fmt.Errorf("get existing blocks: %w", err)
	}

	for _, b := range existing {
		if err := c.client.DeleteBlock(b.ID); err != nil {
			return fmt.Errorf("delete block %s: %w", b.ID, err)
		}
	}

	if len(newBlocks) > 0 {
		if err := c.appendBlocksRecursive(pageID, newBlocks); err != nil {
			return fmt.Errorf("append blocks: %w", err)
		}
	}

	return nil
}

func (c *Connector) appendBlocksRecursive(parentID string, blocks []Block) error {
	// Strip children for the initial append
	stripped := make([]Block, len(blocks))
	childMap := make(map[int][]Block)
	for i, b := range blocks {
		stripped[i] = b
		stripped[i].Children = nil
		if len(b.Children) > 0 {
			childMap[i] = b.Children
		}
	}

	resp, err := c.client.AppendBlocksWithResponse(parentID, &AppendBlocksRequest{Children: stripped})
	if err != nil {
		return err
	}

	// Append nested children to each created parent block
	for i, children := range childMap {
		if i < len(resp) && resp[i].ID != "" {
			if err := c.appendBlocksRecursive(resp[i].ID, children); err != nil {
				return err
			}
		}
	}
	return nil
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

func buildProperties(title string) map[string]Property {
	if title == "" {
		return map[string]Property{}
	}
	return map[string]Property{
		"title": {
			Type:  "title",
			Title: []RichText{{Type: "text", Text: &TextContent{Content: title}}},
		},
	}
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

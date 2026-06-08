package obsidian

import (
	"path/filepath"
	"strings"
)

// ConvertToWikilink converts a markdown link to an Obsidian WikiLink.
// [text](../other-note.md) -> [[other-note|text]]
func ConvertToWikilink(text, href, notesDir string) string {
	if strings.HasPrefix(href, "[[") || strings.HasPrefix(text, "[[") {
		return text
	}

	if !strings.HasSuffix(href, ".md") {
		return ""
	}

	linkName := strings.TrimSuffix(filepath.Base(href), ".md")

	if text == "" || text == linkName || text == href {
		return "[[" + linkName + "]]"
	}

	return "[[" + linkName + "|" + text + "]]"
}

// ExtractWikilinks finds all [[WikiLink]] references in content.
func ExtractWikilinks(content string) []string {
	var links []string
	remaining := content
	for {
		start := strings.Index(remaining, "[[")
		if start == -1 {
			break
		}
		remaining = remaining[start+2:]
		end := strings.Index(remaining, "]]")
		if end == -1 {
			break
		}
		link := remaining[:end]
		if pipeIdx := strings.Index(link, "|"); pipeIdx != -1 {
			link = link[:pipeIdx]
		}
		links = append(links, link)
		remaining = remaining[end+2:]
	}
	return links
}

// ResolveWikilink resolves a WikiLink to a note filename.
func ResolveWikilink(link, notesDir string) string {
	safe := strings.ReplaceAll(link, " ", "-")
	safe = strings.ReplaceAll(safe, "/", "-")
	safe = strings.ToLower(safe)
	return filepath.Join(notesDir, safe+".md")
}

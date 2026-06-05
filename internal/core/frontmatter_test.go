package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter(t *testing.T) {
	content := `---
title: My Note
date: 2026-06-05
tags:
  - go
  - test
---

Note body here
`
	fm, body, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, "My Note", fm.Title)
	assert.Equal(t, "2026-06-05", fm.Date)
	assert.Equal(t, []string{"go", "test"}, fm.Tags)
	assert.Equal(t, "Note body here", body)
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := "Just a simple note without frontmatter"
	fm, body, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Empty(t, fm.Title)
	assert.Equal(t, content, body)
}

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	content := "---\n---\n\nBody"
	fm, body, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Empty(t, fm.Title)
	assert.Equal(t, "Body", strings.TrimSpace(body))
}

func TestParseFrontmatter_MissingCloseTag(t *testing.T) {
	content := "---\ntitle: Broken\nNo closing dashes"
	fm, body, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Empty(t, fm.Title)
	assert.Equal(t, content, body)
}

func TestBuildFrontmatter(t *testing.T) {
	fm := Frontmatter{
		Title: "Test",
		Date:  "2026-06-05",
		Tags:  []string{"go"},
	}
	result := BuildFrontmatter(fm)
	assert.Contains(t, result, "title: Test")
	assert.Contains(t, result, "date: 2026-06-05")
	assert.Contains(t, result, "  - go")
	assert.Contains(t, result, "---")
}

func TestBuildFrontmatter_NoDate(t *testing.T) {
	fm := Frontmatter{Title: "Untitled"}
	result := BuildFrontmatter(fm)
	assert.Contains(t, result, "title: Untitled")
	assert.NotContains(t, result, "date:")
}

func TestBuildFrontmatter_NoTags(t *testing.T) {
	fm := Frontmatter{Title: "Untitled"}
	result := BuildFrontmatter(fm)
	assert.NotContains(t, result, "tags:")
}

func TestBuildNoteContent(t *testing.T) {
	fm := Frontmatter{Title: "Note", Date: "2026-06-05"}
	content := BuildNoteContent(fm, "Hello world")
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "title: Note")
	assert.Contains(t, content, "Hello world")
}

func TestWordCount(t *testing.T) {
	assert.Equal(t, 0, WordCount(""))
	assert.Equal(t, 1, WordCount("hello"))
	assert.Equal(t, 3, WordCount("hello world foo"))
	assert.Equal(t, 3, WordCount("  spaced   out  words  "))
}

func TestParseFrontmatter_ExtraFields(t *testing.T) {
	content := `---
title: Extra
custom_field: value
another: thing
---

Body
`
	fm, body, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, "Extra", fm.Title)
	assert.Equal(t, "Body", body)
}

func TestParseFrontmatter_MultilineTags(t *testing.T) {
	content := `---
title: Tags
tags:
  - tag one
  - tag two
  - tag three
---

Body
`
	fm, _, err := ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, []string{"tag one", "tag two", "tag three"}, fm.Tags)
}

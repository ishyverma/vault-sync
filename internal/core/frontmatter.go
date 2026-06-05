package core

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Frontmatter struct {
	Title string   `yaml:"title"`
	Date  string   `yaml:"date"`
	Tags  []string `yaml:"tags"`
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n?---\n?(.*)$`)

func ParseFrontmatter(content string) (Frontmatter, string, error) {
	matches := frontmatterRe.FindStringSubmatch(content)
	if len(matches) < 3 {
		return Frontmatter{}, content, nil
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(matches[1]), &fm); err != nil {
		return Frontmatter{}, content, fmt.Errorf("parse frontmatter: %w", err)
	}

	return fm, strings.TrimSpace(matches[2]), nil
}

func BuildFrontmatter(fm Frontmatter) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %s\n", fm.Title))
	if fm.Date != "" {
		b.WriteString(fmt.Sprintf("date: %s\n", fm.Date))
	}
	if len(fm.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, tag := range fm.Tags {
			b.WriteString(fmt.Sprintf("  - %s\n", tag))
		}
	}
	b.WriteString("---\n")
	return b.String()
}

func BuildNoteContent(fm Frontmatter, body string) string {
	return BuildFrontmatter(fm) + "\n" + body
}

func WordCount(s string) int {
	words := strings.Fields(s)
	return len(words)
}

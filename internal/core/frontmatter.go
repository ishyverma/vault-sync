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

var frontmatterRe = regexp.MustCompile(`(?s)^---[\r]?\n(.*?)[\r]?\n?---[\r]?\n?(.*)$`)

var frontmatterReCRLF = regexp.MustCompile(`(?s)^---\r\n(.*?)\r?\n?---\r?\n?(.*)$`)

func ParseFrontmatter(content string) (Frontmatter, string, error) {
	matches := frontmatterRe.FindStringSubmatch(content)
	if matches == nil {
		matches = frontmatterReCRLF.FindStringSubmatch(content)
	}
	if len(matches) < 3 {
		return Frontmatter{}, content, nil
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(matches[1]), &fm); err != nil {
		return Frontmatter{}, content, fmt.Errorf("parse frontmatter: %w", err)
	}

	return fm, strings.TrimSpace(matches[2]), nil
}

type buildFrontmatterData struct {
	Title string   `yaml:"title"`
	Date  string   `yaml:"date,omitempty"`
	Tags  []string `yaml:"tags,omitempty"`
}

func BuildFrontmatter(fm Frontmatter) string {
	b, err := yaml.Marshal(buildFrontmatterData{
		Title: fm.Title,
		Date:  fm.Date,
		Tags:  fm.Tags,
	})
	if err != nil {
		return fmt.Sprintf("---\ntitle: %s\n---\n", fm.Title)
	}
	return "---\n" + string(b) + "---\n"
}

func BuildNoteContent(fm Frontmatter, body string) string {
	return BuildFrontmatter(fm) + "\n" + body
}

func WordCount(s string) int {
	words := strings.Fields(s)
	return len(words)
}

package core

import (
	"strings"
	"text/template"
	"time"
)

type TemplateVars struct {
	Title string
	Date  string
}

type TemplateEngine struct {
	templates map[string]string
}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		templates: map[string]string{
			"blank": BlankTemplate,
			"daily": DailyTemplate,
			"meeting": MeetingTemplate,
			"project": ProjectTemplate,
		},
	}
}

func (e *TemplateEngine) Render(name, title string) (string, error) {
	tmplStr, ok := e.templates[name]
	if !ok {
		tmplStr = e.templates["blank"]
	}

	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	vars := TemplateVars{
		Title: title,
		Date:  time.Now().Format("2006-01-02"),
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (e *TemplateEngine) Names() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}

const BlankTemplate = `---
title: "{{.Title}}"
date: {{.Date}}
tags: []
---

`

const DailyTemplate = `---
title: "Daily Note - {{.Date}}"
date: {{.Date}}
tags: [daily]
---

## Today's Focus

## Tasks

- [ ]

## Notes

`

const MeetingTemplate = `---
title: "{{.Title}}"
date: {{.Date}}
tags: [meeting]
---

# {{.Title}}

**Date:** {{.Date}}
**Attendees:**

## Agenda

1.

## Notes

## Action Items

- [ ]

`

const ProjectTemplate = `---
title: "{{.Title}}"
date: {{.Date}}
tags: [project]
---

# {{.Title}}

## Overview

## Goals

## Tasks

- [ ]

## Resources

`

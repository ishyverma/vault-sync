package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateEngine_Render(t *testing.T) {
	e := NewTemplateEngine()
	content, err := e.Render("blank", "My Note")
	require.NoError(t, err)
	assert.Contains(t, content, `title: "My Note"`)
	assert.Contains(t, content, "---")
}

func TestTemplateEngine_Daily(t *testing.T) {
	e := NewTemplateEngine()
	content, err := e.Render("daily", "Today")
	require.NoError(t, err)
	assert.Contains(t, content, "Daily Note")
	assert.Contains(t, content, "[daily]")
	assert.Contains(t, content, "Today's Focus")
}

func TestTemplateEngine_Meeting(t *testing.T) {
	e := NewTemplateEngine()
	content, err := e.Render("meeting", "Sprint Review")
	require.NoError(t, err)
	assert.Contains(t, content, "Sprint Review")
	assert.Contains(t, content, "[meeting]")
}

func TestTemplateEngine_Project(t *testing.T) {
	e := NewTemplateEngine()
	content, err := e.Render("project", "VaultSync")
	require.NoError(t, err)
	assert.Contains(t, content, "VaultSync")
	assert.Contains(t, content, "[project]")
}

func TestTemplateEngine_UnknownName(t *testing.T) {
	e := NewTemplateEngine()
	_, err := e.Render("nonexistent", "Fallback")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown template")
}

func TestTemplateEngine_Names(t *testing.T) {
	e := NewTemplateEngine()
	names := e.Names()
	assert.Contains(t, names, "blank")
	assert.Contains(t, names, "daily")
	assert.Contains(t, names, "meeting")
	assert.Contains(t, names, "project")
}

func TestTemplateEngine_RenderVarSubstitution(t *testing.T) {
	e := NewTemplateEngine()
	content, err := e.Render("blank", "Custom Title Here")
	require.NoError(t, err)
	assert.Contains(t, content, `title: "Custom Title Here"`)
}

package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Esc          key.Binding
	Quit         key.Binding
	Help         key.Binding
	TabNext      key.Binding
	TabPrev      key.Binding
	Dashboard    key.Binding
	Browser      key.Binding
	Search       key.Binding
	Sync         key.Binding
	Settings     key.Binding
	Conflicts    key.Binding
	Open           key.Binding
	Refresh        key.Binding
	Filter         key.Binding
	SortToggle     key.Binding
	ResolveLocal   key.Binding
	ResolveRemote  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help, k.TabNext, k.Dashboard, k.Browser, k.Search, k.Sync}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Dashboard, k.Browser, k.Search, k.Sync},
		{k.Settings, k.Conflicts, k.SortToggle, k.Filter},
		{k.Open, k.Refresh, k.Quit, k.Help},
		{k.ResolveLocal, k.ResolveRemote},
	}
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Esc:           key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:          key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
		TabNext:       key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
		TabPrev:       key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S+tab", "prev tab")),
		Dashboard:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "dashboard")),
		Browser:       key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "notes")),
		Search:        key.NewBinding(key.WithKeys("3", "/"), key.WithHelp("3", "search")),
		Sync:          key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "sync")),
		Settings:      key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "settings")),
		Conflicts:     key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "conflicts")),
		Open:          key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open note")),
		Refresh:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		SortToggle:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
		ResolveLocal:  key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "keep local")),
		ResolveRemote: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "keep remote")),
	}
}

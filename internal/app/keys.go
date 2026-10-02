package app

import "charm.land/bubbles/v2/key"

// KeyMap holds global bindings.
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	Pages    key.Binding
	NextPane key.Binding
	PrevPane key.Binding
	Refresh  key.Binding
	Procs    key.Binding
	Settings key.Binding
	Alerts   key.Binding
	Window   key.Binding
	Back     key.Binding
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
}

// DefaultKeyMap returns the global key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Pages:    key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"), key.WithHelp("1-9", "sections")),
		NextPane: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		PrevPane: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Procs:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "processes")),
		Settings: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
		Alerts:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "alerts")),
		Window:   key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "graph window")),
		Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	}
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Pages, k.NextPane, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Pages, k.NextPane, k.PrevPane, k.Back},
		{k.Procs, k.Settings, k.Alerts, k.Window},
		{k.Refresh, k.Help, k.Quit},
	}
}

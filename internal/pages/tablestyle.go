package pages

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/styles"
)

// newNavTable returns a Bubbles table with MIFO key bindings.
func newNavTable() table.Model {
	km := table.DefaultKeyMap()
	km.LineUp = key.NewBinding(key.WithKeys("up", "k"))
	km.LineDown = key.NewBinding(key.WithKeys("down", "j"))
	km.PageUp = key.NewBinding(key.WithKeys("pgup"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown"))
	km.HalfPageUp = key.NewBinding(key.WithKeys("ctrl+u"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"))
	km.GotoTop = key.NewBinding(key.WithKeys("home", "g"))
	km.GotoBottom = key.NewBinding(key.WithKeys("end", "G"))
	return table.New(table.WithKeyMap(km), table.WithFocused(true))
}

// tableStyles returns table styles for a theme. Cells carry no colour so
// the selection background spans the full row.
func tableStyles(t styles.Theme, focused bool) table.Styles {
	st := table.DefaultStyles()
	b := lipgloss.NormalBorder()
	if !t.Sym.Unicode {
		b = lipgloss.ASCIIBorder()
	}
	st.Header = lipgloss.NewStyle().Foreground(t.P.Muted).Bold(true).Padding(0, 1).
		BorderStyle(b).BorderBottom(true).BorderForeground(t.P.Border)
	st.Cell = lipgloss.NewStyle().Padding(0, 1)
	if focused {
		st.Selected = lipgloss.NewStyle().Background(t.P.AccentBg).Foreground(t.P.Text).Bold(true)
	} else {
		st.Selected = lipgloss.NewStyle().Foreground(t.P.Accent)
	}
	return st
}

// placeholderCols gives a table its column count before the first layout;
// the Bubbles table requires columns before rows are set.
func placeholderCols(n int) []table.Column {
	cols := make([]table.Column, n)
	for i := range cols {
		cols[i] = table.Column{Width: 8}
	}
	return cols
}

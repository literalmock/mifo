package components

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/styles"
)

// NavItem is one sidebar entry.
type NavItem struct {
	Num   int
	Title string
	Short string
}

// FilterValue implements list.Item.
func (i NavItem) FilterValue() string { return i.Title }

type navDelegate struct {
	theme   *styles.Theme
	compact bool
	focused bool
	width   int
}

func (d *navDelegate) Height() int                         { return 1 }
func (d *navDelegate) Spacing() int                        { return 0 }
func (d *navDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d *navDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(NavItem)
	if !ok {
		return
	}
	t := *d.theme
	label := it.Title
	if d.compact {
		label = it.Short
	}
	sel := index == m.Index()
	num := fmt.Sprintf("%d", it.Num)
	inner := d.width - 1
	text := fmt.Sprintf(" %s  %s", num, label)
	if pad := inner - lipgloss.Width(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	if sel {
		bar := lipgloss.NewStyle().Foreground(t.P.Accent).Background(t.P.AccentBg).Render(t.Sym.Select)
		if !d.focused {
			bar = lipgloss.NewStyle().Foreground(t.P.Faint).Background(t.P.AccentBg).Render(t.Sym.Select)
		}
		body := lipgloss.NewStyle().Background(t.P.AccentBg).Foreground(t.P.Text).Bold(true)
		numS := body.Foreground(t.P.Accent).Render(" " + num)
		rest := body.Render("  " + label + strings.Repeat(" ", max(inner-lipgloss.Width(" "+num+"  "+label), 0)))
		fmt.Fprint(w, bar+numS+rest)
		return
	}
	fmt.Fprint(w, " "+t.Faint.Render(" "+num)+t.Muted.Render("  "+label)+strings.Repeat(" ", max(inner-lipgloss.Width(" "+num+"  "+label), 0)))
}

// Sidebar is the navigation menu, built on the Bubbles list.
type Sidebar struct {
	List list.Model
	del  *navDelegate
}

// NewSidebar creates a sidebar for the given items.
func NewSidebar(theme *styles.Theme, items []NavItem) Sidebar {
	li := make([]list.Item, len(items))
	for i, it := range items {
		li[i] = it
	}
	d := &navDelegate{theme: theme, width: 18}
	l := list.New(li, d, 18, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.InfiniteScrolling = true
	return Sidebar{List: l, del: d}
}

// SetCompact toggles the abbreviated layout.
func (s *Sidebar) SetCompact(v bool) { s.del.compact = v }

// SetFocused marks the sidebar as having keyboard focus.
func (s *Sidebar) SetFocused(v bool) { s.del.focused = v }

// Focused reports focus.
func (s *Sidebar) Focused() bool { return s.del.focused }

// Update forwards messages to the list.
func (s *Sidebar) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.List, cmd = s.List.Update(msg)
	return cmd
}

// View renders the sidebar at the given size with an optional footer block
// (live mini stats) pinned to the bottom.
func (s *Sidebar) View(width, height int, footer string) string {
	t := *s.del.theme
	s.del.width = width
	items := len(s.List.Items())
	listH := min(items, height)
	s.List.SetSize(width-1, listH)
	menu := s.List.View()
	var body string
	fh := lipgloss.Height(footer)
	if footer != "" && height-listH-1 >= fh {
		gap := height - listH - fh
		body = menu + strings.Repeat("\n", gap) + footer
	} else {
		body = menu
	}
	body = Fit(body, width-1, height)
	sep := lipgloss.NewStyle().Foreground(t.P.Border)
	lines := strings.Split(body, "\n")
	for i := range lines {
		lines[i] += sep.Render(t.Sym.Border.Left)
	}
	return strings.Join(lines, "\n")
}

// SidebarWidth returns the width for a mode at a terminal width.
func SidebarWidth(compact bool) int {
	if compact {
		return 12
	}
	return 18
}

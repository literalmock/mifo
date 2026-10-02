package app

import (
	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/pages"
)

func navItems(ps []pages.Page) []components.NavItem {
	shorts := map[string]string{
		"overview": "Home", "cpu": "CPU", "memory": "Mem", "processes": "Proc",
		"disk": "Disk", "network": "Net", "sensors": "Temp", "battery": "Batt", "settings": "Prefs",
	}
	items := make([]components.NavItem, len(ps))
	for i, p := range ps {
		items[i] = components.NavItem{Num: i + 1, Title: p.Title(), Short: shorts[p.ID()]}
	}
	return items
}

func (m *Model) pageIndex(id string) int {
	for i, p := range m.pages {
		if p.ID() == id {
			return i
		}
	}
	return -1
}

// navigate switches to page i, recording history for Esc.
func (m *Model) navigate(i int, focusContent bool) {
	if i < 0 || i >= len(m.pages) {
		return
	}
	if i != m.active {
		m.navHist = append(m.navHist, m.active)
		if len(m.navHist) > 32 {
			m.navHist = m.navHist[1:]
		}
	}
	m.active = i
	m.sidebar.List.Select(i)
	if focusContent {
		m.setFocus(focusContent)
	}
}

// back returns to the previous page. It reports whether it moved.
func (m *Model) back() bool {
	if len(m.navHist) == 0 {
		return false
	}
	prev := m.navHist[len(m.navHist)-1]
	m.navHist = m.navHist[:len(m.navHist)-1]
	m.active = prev
	m.sidebar.List.Select(prev)
	return true
}

func (m *Model) setFocus(content bool) {
	m.contentFocus = content
	m.sidebar.SetFocused(!content)
}

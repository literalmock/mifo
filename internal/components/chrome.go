package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/styles"
)

// HeaderData is everything the header shows.
type HeaderData struct {
	Model    string
	OS       string
	Uptime   string
	Clock    string
	Health   string
	HealthLv int // 0 ok, 1 warn, 2 crit
	Reason   string
	Refresh  string
	Busy     bool
}

// Header renders the two-line header plus separator (3 lines total).
func Header(t styles.Theme, d HeaderData, width int) string {
	logo := t.Accent.Bold(true).Render(t.Sym.Logo + " MIFO")
	tag := t.Faint.Italic(true).Render("Your Mac. Every heartbeat.")
	hc := t.Accent
	switch d.HealthLv {
	case 1:
		hc = t.Warn
	case 2:
		hc = t.Crit
	}
	health := hc.Render(t.Sym.Dot + " " + d.Health)
	if d.HealthLv > 0 && d.Reason != "" && width >= 110 {
		health += t.Muted.Render(" " + t.Sym.Sep + " " + d.Reason)
	}
	clock := t.Value.Bold(true).Render(d.Clock)
	left1 := logo + "  " + tag
	if width < 70 {
		left1 = logo
	}
	right1 := health + "   " + clock
	line1 := spread(left1, right1, width)

	sep := " " + t.Sym.Sep + " "
	info := d.Model
	if d.OS != "" {
		info += sep + d.OS
	}
	if d.Uptime != "" {
		info += sep + "up " + d.Uptime
	}
	left2 := t.Muted.Render(info)
	right2 := t.Faint.Render(d.Refresh)
	if d.Busy {
		right2 = t.Accent.Render(t.Sym.Dot+" ") + right2
	}
	line2 := spread(left2, right2, width)
	rule := lipgloss.NewStyle().Foreground(t.P.Border).Render(strings.Repeat(t.Sym.Border.Top, max(width, 0)))
	return Fit(" "+line1, width, 1) + "\n" + Fit(" "+line2, width, 1) + "\n" + rule
}

// spread places left and right on one line, truncating left if needed.
func spread(left, right string, width int) string {
	width-- // leading space
	gap := width - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if gap < 1 {
		return Fit(left, max(width-lipgloss.Width(right)-2, 0), 1) + "  " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

// Footer renders a separator and a single status line (2 lines total).
func Footer(t styles.Theme, help, status string, width int) string {
	rule := lipgloss.NewStyle().Foreground(t.P.Border).Render(strings.Repeat(t.Sym.Border.Bottom, max(width, 0)))
	return rule + "\n" + Fit(" "+spread(help, status, width), width, 1)
}

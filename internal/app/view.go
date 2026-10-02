package app

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/alerts"
	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/format"
)

const (
	minWidth  = 64
	minHeight = 20
	headerH   = 3
	footerH   = 2
)

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView(m.Render())
	v.AltScreen = true
	v.WindowTitle = "MIFO " + m.pages[m.active].Title()
	// Paint the terminal background for explicit themes. "System" follows
	// the terminal's own colours (and painting would skew detection).
	if m.cfg.Theme != config.ThemeSystem {
		v.BackgroundColor = m.theme.P.Bg
		v.ForegroundColor = m.theme.P.Text
	}
	return v
}

// Render returns the full screen as a string (exposed for tests).
func (m Model) Render() string {
	t := *m.theme
	w, h := m.width, m.height
	if w < minWidth || h < minHeight {
		msg := t.Title.Render("Terminal too small") + "\n" +
			t.Muted.Render(fmt.Sprintf("MIFO needs at least %d×%d (now %d×%d)", minWidth, minHeight, w, h))
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg)
	}
	compact := m.cfg.Sidebar == config.SidebarCompact || (m.cfg.Sidebar == config.SidebarAuto && w < 110)
	m.sidebar.SetCompact(compact)
	sbW := components.SidebarWidth(compact)
	bodyH := h - headerH - footerH
	mainW := w - sbW - 1

	page := m.pages[m.active]
	ctx := m.pageContext()
	var mini string
	if !compact {
		mini = m.miniStats(sbW - 1)
	}
	side := m.sidebar.View(sbW, bodyH, mini)
	main := components.Fit(page.View(ctx, mainW-1, bodyH, m.contentFocus), mainW-1, bodyH)
	body := lipgloss.JoinHorizontal(lipgloss.Top, side, " ", main, " ")

	screen := m.header(w) + "\n" + body + "\n" + m.footer(w)
	switch {
	case m.showHelp:
		screen = overlay(screen, m.helpView(min(w-8, 96)), w, h)
	case m.showAlerts:
		screen = overlay(screen, m.alertsView(min(w-8, 90), min(h-6, 26)), w, h)
	}
	return screen
}

func (m Model) header(w int) string {
	t := *m.theme
	d := components.HeaderData{Clock: m.now.Format("15:04:05")}
	if m.hostReady {
		d.Model = m.host.ModelName
		if m.host.Chip != "" && w >= 100 {
			d.Model += " " + t.Sym.Sep + " " + m.host.Chip
		}
		d.OS = strings.TrimSpace(m.host.OSName + " " + m.host.OSVersion)
		if !m.host.BootTime.IsZero() {
			d.Uptime = format.Uptime(m.now.Sub(m.host.BootTime))
		}
	} else {
		d.Model = "Identifying this Mac…"
	}
	if m.hasSnap {
		sev, reason := alerts.Health(m.snap, m.cfg)
		d.Health, d.HealthLv, d.Reason = sev.String(), int(sev), reason
	} else {
		d.Health = "Starting"
	}
	d.Refresh = m.refreshText()
	d.Busy = m.collecting
	return components.Header(t, d, w)
}

func (m Model) refreshText() string {
	iv := fmt.Sprintf("every %ds", m.cfg.IntervalSecs)
	if !m.hasSnap {
		return "sampling " + m.theme.Sym.Sep + " " + iv
	}
	age := m.now.Sub(m.snap.Time)
	if age < 0 {
		age = 0
	}
	stale := age > 3*m.cfg.Interval()+2*time.Second
	ago := fmt.Sprintf("updated %ds ago", int(age.Seconds()))
	if age < time.Second {
		ago = "updated just now"
	}
	if stale {
		return m.theme.Warn.Render("stale "+m.theme.Sym.Sep+" "+ago) + m.theme.Faint.Render(" "+m.theme.Sym.Sep+" "+iv)
	}
	return ago + " " + m.theme.Sym.Sep + " " + iv
}

func (m Model) footer(w int) string {
	t := *m.theme
	page := m.pages[m.active]
	var binds []key.Binding
	if m.contentFocus {
		binds = append(binds, page.Help(m.pageContext())...)
	} else {
		binds = append(binds, m.keys.Up, m.keys.Down, m.keys.Enter)
	}
	binds = append(binds, m.keys.NextPane, m.keys.Help, m.keys.Quit)
	m.help.SetWidth(w * 2 / 3)
	helpS := m.help.ShortHelpView(binds)

	var status string
	if m.status != "" {
		st := t.Accent
		if m.statusErr {
			st = t.Warn
		}
		status = st.Render(m.status)
	} else {
		status = t.Muted.Render(page.Title()) + t.Faint.Render(" "+t.Sym.Sep+" ")
		if !m.contentFocus {
			status += t.Faint.Render("menu " + t.Sym.Sep + " ")
		}
		if m.collecting {
			status += t.Accent.Render(t.Sym.Dot + " sampling")
		} else {
			status += t.Accent.Render(t.Sym.Dot) + t.Muted.Render(" live")
		}
	}
	maxStatus := w - lipgloss.Width(helpS) - 4
	if lipgloss.Width(status) > maxStatus && maxStatus > 10 {
		status = components.Fit(status, maxStatus, 1)
	}
	return components.Footer(t, helpS, status, w)
}

// miniStats renders compact live meters for the bottom of the sidebar.
func (m Model) miniStats(w int) string {
	if !m.hasSnap || w < 14 {
		return ""
	}
	t := *m.theme
	row := func(label string, pct float64, ok bool) string {
		if !ok {
			return " " + t.Faint.Render(fmt.Sprintf("%-4s", label)) + t.Faint.Render(" n/a")
		}
		val := fmt.Sprintf("%3.0f%%", pct)
		barW := w - 1 - 4 - 1 - len(val) - 1
		return " " + t.Faint.Render(fmt.Sprintf("%-4s", label)) + " " +
			components.Bar(t, pct, max(barW, 1), t.LevelColor(pct, 70, 90)) + " " + t.Muted.Render(val)
	}
	s := m.snap
	lines := []string{
		row("CPU", s.CPU.Total, s.CPU.Valid),
		row("MEM", s.Memory.UsedPercent, s.Memory.Valid),
	}
	if s.Battery.Present {
		lines = append(lines, row("BAT", s.Battery.Percent, true))
	}
	return strings.Join(lines, "\n")
}

// overlay composites a dialog centred over the screen using Lip Gloss layers.
func overlay(base, fg string, w, h int) string {
	fw, fh := lipgloss.Size(fg)
	x := max((w-fw)/2, 0)
	y := max((h-fh)/2, 0)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(fg).X(x).Y(y).Z(1),
	)
	return comp.Render()
}

func (m Model) dialogBox(title, body string, width int) string {
	t := *m.theme
	st := lipgloss.NewStyle().
		Border(t.Sym.Border).BorderForeground(t.P.BorderHi).
		Background(t.P.Surface).Padding(1, 2).Width(width)
	return st.Render(t.Accent.Bold(true).Render(title) + "\n\n" + body)
}

func (m Model) helpView(width int) string {
	t := *m.theme
	page := m.pages[m.active]
	groups := m.keys.FullHelp()
	if pb := page.Help(m.pageContext()); len(pb) > 0 {
		groups = append(groups, pb)
	}
	m.help.SetWidth(width - 6)
	body := m.help.FullHelpView(groups)
	body += "\n\n" + t.Faint.Render("1 Overview  2 CPU  3 Memory  4 Processes  5 Disk") +
		"\n" + t.Faint.Render("6 Network  7 Sensors  8 Battery  9 Settings")
	body += "\n" + t.Faint.Render("Press ? or esc to close")
	return m.dialogBox("Keyboard shortcuts "+t.Sym.Sep+" "+page.Title(), body, width)
}

func (m Model) alertsView(width, height int) string {
	t := *m.theme
	var lines []string
	active := m.alerts.Active()
	lines = append(lines, t.Title.Render("Active"))
	if len(active) == 0 {
		lines = append(lines, t.Accent.Render(t.Sym.Check+" ")+t.Muted.Render("No active alerts"))
	}
	for i, a := range active {
		st := t.Warn
		if a.Severity == alerts.Critical {
			st = t.Crit
		}
		line := st.Render(t.Sym.Dot+" ") + t.Value.Render(a.Title) + t.Muted.Render("  "+a.Detail)
		if i == m.alertCursor {
			line = t.Selected.Render(t.Sym.Select+" ") + line
		} else {
			line = "  " + line
		}
		lines = append(lines, components.Fit(line, width-6, 1))
	}
	lines = append(lines, "", t.Title.Render("History"))
	hist := m.alerts.History()
	if len(hist) == 0 {
		lines = append(lines, t.Muted.Render("Nothing yet this session"))
	}
	room := max(height-len(lines)-6, 1)
	for i, a := range hist {
		if i >= room {
			lines = append(lines, t.Faint.Render(fmt.Sprintf("… %d more", len(hist)-room)))
			break
		}
		state := t.Warn.Render("active")
		if !a.Active() {
			state = t.Faint.Render("resolved " + a.Resolved.Format("15:04"))
		} else if a.Dismissed {
			state = t.Muted.Render("dismissed")
		}
		lines = append(lines, components.Fit(t.Faint.Render(a.At.Format("15:04:05"))+"  "+t.Value.Render(a.Title)+"  "+state, width-6, 1))
	}
	lines = append(lines, "", t.Faint.Render("j/k select · d dismiss · D dismiss all · esc close"))
	if !m.cfg.AlertsEnabled {
		lines = append(lines, t.Warn.Render("Alerts are disabled in Settings"))
	}
	return m.dialogBox("Alerts", strings.Join(lines, "\n"), width)
}

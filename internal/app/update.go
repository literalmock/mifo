package app

import (
	"slices"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/pages"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.BackgroundColorMsg:
		m.darkBG, m.bgKnown = msg.IsDark(), true
		m.applyTheme()
		return m, nil

	case hostMsg:
		m.host, m.hostReady = msg.host, true
		return m, nil

	case tickMsg:
		if msg.gen != m.tickGen {
			return m, nil // stale tick from a previous interval
		}
		return m, tea.Batch(m.collectCmd(), m.scheduleTick())

	case clockMsg:
		m.now = m.nowFn()
		if !m.statusUntil.IsZero() && m.now.After(m.statusUntil) {
			m.status, m.statusUntil = "", time.Time{}
		}
		return m, nil

	case snapshotMsg:
		m.collecting = false
		m.snap, m.hasSnap = msg.snap, true
		m.now = m.nowFn()
		m.hist.Record(msg.snap)
		raised := m.alerts.Evaluate(msg.snap, m.cfg, m.now)
		if m.cfg.NativeNotify {
			for _, a := range raised {
				cmds = append(cmds, notifyCmd(m.ctx, a))
			}
		}
		// Let pages react (e.g. keep table selection stable).
		ctx := m.pageContext()
		for _, p := range m.pages {
			cmds = append(cmds, p.Update(ctx, msg.snap))
		}
		return m, tea.Batch(cmds...)

	case configSaveMsg:
		if msg.err != nil {
			m.setStatus("Could not save settings: "+msg.err.Error(), true)
		}
		return m, nil

	case pages.NavigateMsg:
		if i := m.pageIndex(msg.Page); i >= 0 {
			m.navigate(i, true)
		}
		return m, nil

	case pages.StatusMsg:
		m.setStatus(msg.Text, msg.Error)
		return m, nil

	case pages.ConfigChangedMsg:
		return m, m.applyConfig(msg.Cfg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Route everything else (async results, cursor blinks) to all pages.
	ctx := m.pageContext()
	for _, p := range m.pages {
		cmds = append(cmds, p.Update(ctx, msg))
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) setStatus(s string, isErr bool) {
	m.status, m.statusErr = s, isErr
	m.statusUntil = m.nowFn().Add(5 * time.Second)
}

// applyConfig adopts a new configuration, restarting the sampling timer
// when the interval changed, and persists it.
func (m *Model) applyConfig(c config.Config) tea.Cmd {
	c.Validate()
	old := m.cfg
	m.cfg = c
	var cmds []tea.Cmd
	if old.Theme != c.Theme || old.Symbols != c.Symbols {
		m.applyTheme()
	}
	if old.IntervalSecs != c.IntervalSecs {
		m.savedInterval = 0 // the user chose an interval explicitly
		m.tickGen++
		cmds = append(cmds, m.scheduleTick())
	}
	persist := c
	if m.savedInterval > 0 {
		persist.IntervalSecs = m.savedInterval
	}
	cmds = append(cmds, m.saveConfigCmd(persist))
	return tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	page := m.pages[m.active]

	// Overlays take precedence.
	if m.showHelp {
		if key.Matches(msg, m.keys.Help, m.keys.Back) {
			m.showHelp = false
		} else if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}
	if m.showAlerts {
		return m.handleAlertsKey(msg)
	}

	// A page with an active text input or dialog receives every key.
	if m.contentFocus && page.Capturing() {
		cmd, _ := page.HandleKey(m.pageContext(), msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	case key.Matches(msg, m.keys.Pages):
		i := int(msg.String()[0] - '1')
		m.navigate(i, true)
		return m, nil
	case key.Matches(msg, m.keys.NextPane), key.Matches(msg, m.keys.PrevPane):
		m.setFocus(!m.contentFocus)
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		if m.collecting {
			m.setStatus("A collection is already in progress", false)
			return m, nil
		}
		m.setStatus("Refreshing…", false)
		return m, m.collectCmd()
	case key.Matches(msg, m.keys.Procs):
		m.navigate(m.pageIndex("processes"), true)
		return m, nil
	case key.Matches(msg, m.keys.Settings):
		m.navigate(m.pageIndex("settings"), true)
		return m, nil
	case key.Matches(msg, m.keys.Alerts):
		m.showAlerts, m.alertCursor = true, 0
		return m, nil
	case key.Matches(msg, m.keys.Window):
		c := m.cfg
		i := slices.Index(config.Windows, c.WindowSecs)
		c.WindowSecs = config.Windows[(i+1)%len(config.Windows)]
		m.setStatus("Graph window: "+windowName(c.WindowSecs), false)
		return m, m.applyConfig(c)
	}

	if !m.contentFocus {
		switch {
		case key.Matches(msg, m.keys.Enter), msg.String() == "right", msg.String() == "l":
			m.setFocus(true)
			return m, nil
		case key.Matches(msg, m.keys.Back):
			if m.back() {
				return m, nil
			}
			return m, nil
		}
		if !m.cfg.VimKeys && (msg.String() == "j" || msg.String() == "k") {
			return m, nil
		}
		before := m.sidebar.List.Index()
		cmd := m.sidebar.Update(msg)
		if after := m.sidebar.List.Index(); after != before {
			m.navHist = append(m.navHist, m.active)
			m.active = after
		}
		return m, cmd
	}

	if !m.cfg.VimKeys && (msg.String() == "j" || msg.String() == "k") {
		return m, nil
	}
	cmd, handled := page.HandleKey(m.pageContext(), msg)
	if handled {
		return m, cmd
	}
	if key.Matches(msg, m.keys.Back) {
		if !m.back() {
			m.setFocus(false)
		}
		return m, nil
	}
	if msg.String() == "left" || msg.String() == "h" {
		m.setFocus(false)
	}
	return m, nil
}

func (m Model) handleAlertsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	active := m.alerts.Active()
	switch msg.String() {
	case "esc", "a", "q":
		m.showAlerts = false
	case "up", "k":
		m.alertCursor = max(m.alertCursor-1, 0)
	case "down", "j":
		m.alertCursor = min(m.alertCursor+1, max(len(active)-1, 0))
	case "d", "enter":
		if m.alertCursor < len(active) {
			m.alerts.Dismiss(active[m.alertCursor].ID)
			m.alertCursor = max(min(m.alertCursor, len(active)-2), 0)
		}
	case "D":
		m.alerts.DismissAll()
		m.alertCursor = 0
	}
	return m, nil
}

func windowName(secs int) string {
	switch secs {
	case 60:
		return "1 minute"
	case 300:
		return "5 minutes"
	case 900:
		return "15 minutes"
	case 3600:
		return "1 hour"
	}
	return ""
}

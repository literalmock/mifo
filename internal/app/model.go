// Package app wires metrics, pages and chrome into a Bubble Tea program.
package app

import (
	"context"
	"os"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"

	"github.com/literalmock/mifo/internal/alerts"
	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/metrics"
	"github.com/literalmock/mifo/internal/pages"
	"github.com/literalmock/mifo/internal/styles"
)

// Options configures a new Model.
type Options struct {
	Ctx        context.Context
	Source     metrics.Source
	Config     config.Config
	ConfigPath string
	// Warning is shown once at startup (e.g. a repaired config file).
	Warning string
	// Ops overrides side-effecting operations (tests).
	Ops pages.Ops
	// Now overrides the clock (tests).
	Now func() time.Time
	// NoSave disables config persistence (tests, demo).
	NoSave bool
	// StartPage overrides the configured start page for this session only.
	StartPage string
	// Interval overrides the refresh interval for this session only; it is
	// not written to disk unless the user changes the interval in Settings.
	Interval int
}

// Model is the root Bubble Tea model.
type Model struct {
	ctx     context.Context
	src     metrics.Source
	ops     pages.Ops
	nowFn   func() time.Time
	cfg     config.Config
	cfgPath string
	noSave  bool
	// savedInterval is the on-disk interval while a session override is
	// active (0 when there is no override).
	savedInterval int

	theme   *styles.Theme
	darkBG  bool
	bgKnown bool

	host      metrics.HostInfo
	hostReady bool
	snap      metrics.Snapshot
	hasSnap   bool
	hist      *metrics.History
	alerts    *alerts.Engine

	pages        []pages.Page
	active       int
	navHist      []int
	contentFocus bool
	sidebar      components.Sidebar
	help         help.Model
	keys         KeyMap

	showHelp    bool
	showAlerts  bool
	alertCursor int

	width, height int
	collecting    bool
	tickGen       int
	now           time.Time

	status      string
	statusErr   bool
	statusUntil time.Time
}

// New constructs the root model.
func New(o Options) Model {
	if o.Ctx == nil {
		o.Ctx = context.Background()
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	m := Model{
		ctx:     o.Ctx,
		src:     o.Source,
		nowFn:   o.Now,
		cfg:     o.Config,
		cfgPath: o.ConfigPath,
		noSave:  o.NoSave,
		theme:   new(styles.Theme),
		darkBG:  true,
		hist:    metrics.NewHistory(metrics.HistoryCapacity),
		alerts:  alerts.NewEngine(),
		keys:    DefaultKeyMap(),
		help:    help.New(),
		width:   100,
		height:  30,
	}
	m.cfg.Validate()
	if o.Interval > 0 && o.Interval != m.cfg.IntervalSecs {
		m.savedInterval = m.cfg.IntervalSecs
		m.cfg.IntervalSecs = o.Interval
		m.cfg.Validate()
	}
	m.ops = o.Ops
	if m.ops == nil {
		m.ops = realOps{ctx: o.Ctx}
	}
	m.now = m.nowFn()
	m.applyTheme()
	m.pages = []pages.Page{
		pages.NewOverview(),
		pages.NewCPU(),
		pages.NewMemory(),
		pages.NewProcesses(),
		pages.NewDisk(),
		pages.NewNetwork(),
		pages.NewSensors(),
		pages.NewBattery(),
		pages.NewSettings(),
	}
	m.sidebar = components.NewSidebar(m.theme, navItems(m.pages))
	start := m.cfg.StartPage
	if o.StartPage != "" {
		start = o.StartPage
	}
	if i := m.pageIndex(start); i >= 0 {
		m.active = i
		m.sidebar.List.Select(i)
	}
	m.setFocus(m.active != 0)
	if o.Warning != "" {
		m.status, m.statusErr, m.statusUntil = o.Warning, true, m.now.Add(10*time.Second)
	}
	return m
}

// applyTheme rebuilds styles from the current config.
func (m *Model) applyTheme() {
	p := styles.DarkPalette
	switch m.cfg.Theme {
	case config.ThemeLight:
		p = styles.LightPalette
	case config.ThemeSystem:
		if !m.darkBG {
			p = styles.LightPalette
		}
	}
	sym := styles.UnicodeSymbols
	switch m.cfg.Symbols {
	case config.SymbolsASCII:
		sym = styles.ASCIISymbols
	case config.SymbolsAuto:
		if !styles.DetectUnicode() {
			sym = styles.ASCIISymbols
		}
	}
	*m.theme = styles.New(p, sym)
	m.help.Styles = help.DefaultStyles(p.Dark)
	m.help.Styles.ShortKey = m.theme.Key
	m.help.Styles.ShortDesc = m.theme.Muted
	m.help.Styles.ShortSeparator = m.theme.Faint
	m.help.Styles.FullKey = m.theme.Accent
	m.help.Styles.FullDesc = m.theme.Muted
	m.help.Styles.FullSeparator = m.theme.Faint
	if sym.Unicode {
		m.help.ShortSeparator = " · "
	} else {
		m.help.ShortSeparator = " | "
	}
}

// pageContext assembles the shared page context.
func (m *Model) pageContext() *pages.Context {
	return &pages.Context{
		Ctx:     m.ctx,
		Theme:   *m.theme,
		Snap:    m.snap,
		HasSnap: m.hasSnap,
		Hist:    m.hist,
		Host:    m.host,
		Cfg:     m.cfg,
		CfgPath: m.cfgPath,
		Alerts:  m.alerts,
		Now:     m.now,
		SelfPID: os.Getpid(),
		Ops:     m.ops,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.hostCmd(), m.collectCmd(), m.scheduleTick(), clockTick(), tea.RequestBackgroundColor)
}

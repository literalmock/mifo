// Package pages implements MIFO's screens. Each page is a small state
// machine driven by the app model; rendering is a pure function of page state
// and the shared Context.
package pages

import (
	"context"
	"fmt"
	"image/color"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/literalmock/mifo/internal/alerts"
	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
	"github.com/literalmock/mifo/internal/styles"
)

// Context is the shared, read-mostly state pages render from.
type Context struct {
	Ctx     context.Context
	Theme   styles.Theme
	Snap    metrics.Snapshot
	HasSnap bool
	Hist    *metrics.History
	Host    metrics.HostInfo
	Cfg     config.Config
	CfgPath string
	Alerts  *alerts.Engine
	Now     time.Time
	SelfPID int
	Ops     Ops
}

// Ops are side-effecting operations pages may request. They return
// commands so the UI loop never blocks; tests substitute fakes.
type Ops interface {
	Signal(key metrics.ProcKey, force bool) tea.Cmd
	ProcessDetail(key metrics.ProcKey) tea.Cmd
	Diagnostics() tea.Cmd
	NetTop() tea.Cmd
}

// Page is one screen.
type Page interface {
	ID() string
	Title() string
	// HandleKey processes a key press while the page has focus and reports
	// whether it consumed the key.
	HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool)
	// Update receives every non-key message.
	Update(c *Context, msg tea.Msg) tea.Cmd
	View(c *Context, width, height int, focused bool) string
	// Help returns contextual key bindings for the footer.
	Help(c *Context) []key.Binding
	// Capturing is true when a text input or dialog needs all keys.
	Capturing() bool
}

// NavigateMsg asks the app to switch pages.
type NavigateMsg struct{ Page string }

// ConfigChangedMsg carries an updated configuration to persist and apply.
type ConfigChangedMsg struct{ Cfg config.Config }

// StatusMsg shows a transient message in the footer.
type StatusMsg struct {
	Text  string
	Error bool
}

// SignalResultMsg reports the outcome of terminating a process.
type SignalResultMsg struct {
	Key   metrics.ProcKey
	Name  string
	Force bool
	Err   error
}

// ProcessDetailMsg delivers on-demand process details.
type ProcessDetailMsg struct{ Detail metrics.ProcessDetail }

// DiagnosticsMsg delivers network diagnostics results.
type DiagnosticsMsg struct{ Result metrics.DiagResult }

// NetTopMsg delivers a per-process network snapshot.
type NetTopMsg struct{ Result metrics.NetTopResult }

// Keys used across pages.
var (
	keyUp    = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	keyDown  = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	keyEnter = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "inspect"))
	keyEsc   = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
	keySlash = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search"))
)

// ---- shared rendering helpers ----

// series returns the values of ring within the configured window.
func series(c *Context, r *metrics.Ring, buf []float64) []float64 {
	if r == nil {
		return buf[:0]
	}
	return r.Since(c.Now.Add(-c.Cfg.Window()), buf[:0])
}

func windowLabel(secs int) string {
	if secs >= 3600 {
		return fmt.Sprintf("%dh", secs/3600)
	}
	return fmt.Sprintf("%dm", secs/60)
}

func pctLabel(v float64) string { return fmt.Sprintf("%.0f%%", v) }

func rateLabel(v float64) string { return format.Rate(v) }

func cpuColor(t styles.Theme, pct float64) color.Color { return t.LevelColor(pct, 60, 85) }

func pressureColor(t styles.Theme, p metrics.PressureLevel) color.Color {
	switch p {
	case metrics.PressureWarning:
		return t.P.Warn
	case metrics.PressureCritical:
		return t.P.Crit
	case metrics.PressureNormal:
		return t.P.Accent
	}
	return t.P.Muted
}

func thermalColor(t styles.Theme, p metrics.ThermalPressure) color.Color {
	switch p {
	case metrics.ThermalNominal:
		return t.P.Accent
	case metrics.ThermalModerate:
		return t.P.Warn
	case metrics.ThermalUnknown:
		return t.P.Muted
	}
	return t.P.Crit
}

func tempStr(t styles.Theme, c float64) string {
	if t.Sym.Unicode {
		return format.Temp(c)
	}
	return format.TempASCII(c)
}

// waiting renders the placeholder shown before the first snapshot arrives.
func waiting(c *Context, w, h int) string {
	return components.Fit(c.Theme.Muted.Render("  Collecting first sample…"), w, h)
}

// errLine renders a collection error compactly.
func errLine(t styles.Theme, what string, err error) string {
	if err == nil {
		return t.Muted.Render(what + " unavailable")
	}
	return t.Warn.Render(what+" unavailable: ") + t.Muted.Render(err.Error())
}

func max0(v int) int { return max(v, 0) }

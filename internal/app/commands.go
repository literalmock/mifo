package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/literalmock/mifo/internal/alerts"
	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/metrics"
	"github.com/literalmock/mifo/internal/pages"
)

type (
	snapshotMsg   struct{ snap metrics.Snapshot }
	hostMsg       struct{ host metrics.HostInfo }
	tickMsg       struct{ gen int }
	clockMsg      time.Time
	configSaveMsg struct{ err error }
)

func (m *Model) collectCmd() tea.Cmd {
	if m.collecting || m.src == nil {
		return nil
	}
	m.collecting = true
	src, ctx := m.src, m.ctx
	return func() tea.Msg {
		return snapshotMsg{snap: src.Collect(ctx)}
	}
}

func (m Model) hostCmd() tea.Cmd {
	src, ctx := m.src, m.ctx
	if src == nil {
		return nil
	}
	return func() tea.Msg { return hostMsg{host: src.Host(ctx)} }
}

func (m *Model) scheduleTick() tea.Cmd {
	gen := m.tickGen
	return tea.Tick(m.cfg.Interval(), func(time.Time) tea.Msg { return tickMsg{gen: gen} })
}

func clockTick() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg { return clockMsg(t) })
}

func (m Model) saveConfigCmd(c config.Config) tea.Cmd {
	if m.noSave || m.cfgPath == "" {
		return nil
	}
	path := m.cfgPath
	return func() tea.Msg { return configSaveMsg{err: config.Save(path, c)} }
}

// appleScriptQuote escapes a string for an AppleScript string literal.
func appleScriptQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// notifyCmd posts a native notification via osascript. macOS may ask the
// user to allow notifications for the terminal app the first time.
func notifyCmd(ctx context.Context, a alerts.Alert) tea.Cmd {
	return func() tea.Msg {
		script := fmt.Sprintf("display notification %s with title %s subtitle %s",
			appleScriptQuote(a.Detail), appleScriptQuote("MIFO"), appleScriptQuote(a.Title))
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := exec.CommandContext(cctx, "/usr/bin/osascript", "-e", script).Run(); err != nil {
			return pages.StatusMsg{Text: "Native notification failed: " + err.Error(), Error: true}
		}
		return nil
	}
}

// realOps performs real side effects.
type realOps struct{ ctx context.Context }

func (o realOps) Signal(k metrics.ProcKey, force bool) tea.Cmd {
	return func() tea.Msg {
		sig := syscall.SIGTERM
		if force {
			sig = syscall.SIGKILL
		}
		err := metrics.Signal(o.ctx, k, sig, syscall.Getpid())
		return pages.SignalResultMsg{Key: k, Force: force, Err: err}
	}
}

func (o realOps) ProcessDetail(k metrics.ProcKey) tea.Cmd {
	return func() tea.Msg { return pages.ProcessDetailMsg{Detail: metrics.FetchProcessDetail(o.ctx, k)} }
}

func (o realOps) Diagnostics() tea.Cmd {
	return func() tea.Msg { return pages.DiagnosticsMsg{Result: metrics.RunDiagnostics(o.ctx)} }
}

func (o realOps) NetTop() tea.Cmd {
	return func() tea.Msg { return pages.NetTopMsg{Result: metrics.RunNetTop(o.ctx)} }
}

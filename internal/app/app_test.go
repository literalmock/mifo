package app

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/metrics"
	"github.com/literalmock/mifo/internal/pages"
)

var fixedNow = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

type fakeOps struct {
	mu      sync.Mutex
	signals []metrics.ProcKey
	forced  []bool
}

func (f *fakeOps) Signal(k metrics.ProcKey, force bool) tea.Cmd {
	f.mu.Lock()
	f.signals = append(f.signals, k)
	f.forced = append(f.forced, force)
	f.mu.Unlock()
	return func() tea.Msg { return pages.SignalResultMsg{Key: k, Force: force} }
}
func (f *fakeOps) ProcessDetail(k metrics.ProcKey) tea.Cmd {
	return func() tea.Msg {
		return pages.ProcessDetailMsg{Detail: metrics.ProcessDetail{Key: k, Args: "/bin/test --flag", Fetched: fixedNow}}
	}
}
func (f *fakeOps) Diagnostics() tea.Cmd { return nil }
func (f *fakeOps) NetTop() tea.Cmd      { return nil }

func newModel(t *testing.T, src metrics.Source) (Model, *fakeOps) {
	t.Helper()
	ops := &fakeOps{}
	cfg := config.Default()
	m := New(Options{
		Ctx: context.Background(), Source: src, Config: cfg, Ops: ops, NoSave: true,
		Now: func() time.Time { return fixedNow },
	})
	return m, ops
}

func mockSource() *metrics.MockSource {
	return &metrics.MockSource{
		HostInfo: metrics.SampleHost(),
		Next:     func(n int) metrics.Snapshot { return metrics.SampleSnapshot(fixedNow, n) },
	}
}

// update applies msg and synchronously drains any resulting commands that
// complete quickly (timers are skipped).
func update(m Model, msg tea.Msg) Model {
	tm, cmd := m.Update(msg)
	m = tm.(Model)
	return drain(m, cmd, 0)
}

func drain(m Model, cmd tea.Cmd, depth int) Model {
	if cmd == nil || depth > 5 {
		return m
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(30 * time.Millisecond):
		return m // a timer; ignore
	}
	switch msg := msg.(type) {
	case nil:
		return m
	case tea.BatchMsg:
		for _, c := range msg {
			m = drain(m, c, depth+1)
		}
		return m
	case tea.QuitMsg:
		return m
	}
	tm, next := m.Update(msg)
	return drain(tm.(Model), next, depth+1)
}

func press(s string) tea.KeyPressMsg {
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func ready(t *testing.T, w, h int) Model {
	m, _ := newModel(t, mockSource())
	m = update(m, tea.WindowSizeMsg{Width: w, Height: h})
	m = update(m, hostMsg{host: metrics.SampleHost()})
	m.collecting = true
	m = update(m, snapshotMsg{snap: metrics.SampleSnapshot(fixedNow, 0)})
	return m
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestNumberKeysNavigate(t *testing.T) {
	m := ready(t, 120, 40)
	for i, id := range []string{"overview", "cpu", "memory", "processes", "disk", "network", "sensors", "battery", "settings"} {
		m = update(m, press(string(rune('1'+i))))
		if m.pages[m.active].ID() != id {
			t.Fatalf("key %d -> %s, want %s", i+1, m.pages[m.active].ID(), id)
		}
		if m.sidebar.List.Index() != i {
			t.Errorf("sidebar not synced for %s", id)
		}
	}
}

func TestShortcutsAndBack(t *testing.T) {
	m := ready(t, 120, 40)
	m = update(m, press("p"))
	if m.pages[m.active].ID() != "processes" || !m.contentFocus {
		t.Fatal("p should open focused process manager")
	}
	m = update(m, press("s"))
	if m.pages[m.active].ID() != "settings" {
		t.Fatal("s should open settings")
	}
	m = update(m, press("esc"))
	if m.pages[m.active].ID() != "processes" {
		t.Fatalf("esc should go back, at %s", m.pages[m.active].ID())
	}
}

func TestTabTogglesFocus(t *testing.T) {
	m := ready(t, 120, 40)
	start := m.contentFocus
	m = update(m, press("tab"))
	if m.contentFocus == start {
		t.Fatal("tab did not move focus")
	}
	m = update(m, press("shift+tab"))
	if m.contentFocus != start {
		t.Fatal("shift+tab did not move focus back")
	}
}

func TestSidebarArrowNavigation(t *testing.T) {
	m := ready(t, 120, 40)
	m.setFocus(false)
	m = update(m, press("down"))
	if m.pages[m.active].ID() != "cpu" {
		t.Fatalf("down in sidebar -> %s", m.pages[m.active].ID())
	}
	m = update(m, press("j"))
	if m.pages[m.active].ID() != "memory" {
		t.Fatalf("j in sidebar -> %s", m.pages[m.active].ID())
	}
	m = update(m, press("enter"))
	if !m.contentFocus {
		t.Fatal("enter should focus content")
	}
}

func TestQuitKeys(t *testing.T) {
	m := ready(t, 120, 40)
	if _, cmd := m.Update(press("q")); !isQuit(cmd) {
		t.Error("q should quit")
	}
	if _, cmd := m.Update(press("ctrl+c")); !isQuit(cmd) {
		t.Error("ctrl+c should quit")
	}
}

func TestHelpOverlayToggle(t *testing.T) {
	m := ready(t, 120, 40)
	m = update(m, press("?"))
	if !m.showHelp || !strings.Contains(m.Render(), "Keyboard shortcuts") {
		t.Fatal("help not shown")
	}
	m = update(m, press("3"))
	if m.pages[m.active].ID() == "memory" {
		t.Fatal("keys must not leak through the help overlay")
	}
	m = update(m, press("esc"))
	if m.showHelp {
		t.Fatal("esc should close help")
	}
}

func TestSearchCapturesGlobalKeys(t *testing.T) {
	m := ready(t, 140, 40)
	m = update(m, press("p"))
	m = update(m, press("/"))
	for _, r := range "qs1" {
		tm, cmd := m.Update(press(string(r)))
		if isQuit(cmd) {
			t.Fatal("q quit while typing in search")
		}
		m = drain(tm.(Model), cmd, 0)
	}
	if m.pages[m.active].ID() != "processes" {
		t.Fatalf("global key navigated away while searching: %s", m.pages[m.active].ID())
	}
	if !strings.Contains(m.Render(), "qs1") {
		t.Error("typed text not shown in search box")
	}
	m = update(m, press("esc"))
	if m.pages[m.active].Capturing() {
		t.Error("esc should leave search")
	}
}

func TestSearchFiltersTable(t *testing.T) {
	m := ready(t, 140, 40)
	m = update(m, press("p"))
	m = update(m, press("/"))
	for _, r := range "xcode" {
		m = update(m, press(string(r)))
	}
	m = update(m, press("enter"))
	out := m.Render()
	if !strings.Contains(out, "Xcode") || strings.Contains(out, "launchd") {
		t.Error("filter not applied")
	}
	if !strings.Contains(out, "1 of 5") {
		t.Error("count not shown")
	}
}

func TestTerminateRequiresConfirmation(t *testing.T) {
	m := ready(t, 140, 40)
	ops := m.ops.(*fakeOps)
	m = update(m, press("p"))
	// Default sort is CPU desc: Xcode (PID 777) first.
	m = update(m, press("x"))
	if len(ops.signals) != 0 {
		t.Fatal("signal sent without confirmation")
	}
	if !strings.Contains(m.Render(), "Terminate process?") {
		t.Fatal("confirmation dialog not shown")
	}
	m = update(m, press("n"))
	if len(ops.signals) != 0 || m.pages[m.active].Capturing() {
		t.Fatal("cancel did not close dialog cleanly")
	}
	m = update(m, press("X"))
	if !strings.Contains(m.Render(), "Force quit process?") {
		t.Fatal("force dialog not shown")
	}
	m = update(m, press("y"))
	if len(ops.signals) != 1 || ops.signals[0].PID != 777 || !ops.forced[0] {
		t.Fatalf("signals = %+v forced=%v", ops.signals, ops.forced)
	}
	if ops.signals[0].Start != fixedNow.Add(-time.Hour).Unix() {
		t.Error("signal must carry the process start time for PID-reuse safety")
	}
}

func TestProcessDetailPanel(t *testing.T) {
	m := ready(t, 160, 44)
	m = update(m, press("p"))
	m = update(m, press("enter"))
	out := m.Render()
	if !strings.Contains(out, "Details") || !strings.Contains(out, "/bin/test --flag") {
		t.Error("detail panel or fetched args missing")
	}
	m = update(m, press("esc"))
	if strings.Contains(m.Render(), "Command line") {
		t.Error("esc should close details")
	}
}

func TestSortKeys(t *testing.T) {
	m := ready(t, 140, 40)
	m = update(m, press("p"))
	m = update(m, press("n"))
	if !strings.Contains(ansi.Strip(m.Render()), "sort Name") {
		t.Error("name sort not applied")
	}
	m = update(m, press("m"))
	if !strings.Contains(ansi.Strip(m.Render()), "sort Memory") {
		t.Error("memory sort not applied")
	}
}

func TestWindowKeyCyclesGraphWindow(t *testing.T) {
	m := ready(t, 120, 40)
	before := m.cfg.WindowSecs
	m = update(m, press("w"))
	if m.cfg.WindowSecs == before {
		t.Fatal("window not changed")
	}
}

func TestSettingsChangePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	ops := &fakeOps{}
	m := New(Options{Source: mockSource(), Config: config.Default(), ConfigPath: path, Ops: ops, Now: func() time.Time { return fixedNow }})
	m = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = update(m, press("s"))
	m = update(m, press("enter")) // refresh interval 2s -> 5s
	if m.cfg.IntervalSecs != 5 {
		t.Fatalf("interval = %d", m.cfg.IntervalSecs)
	}
	if got := config.Load(path).Config.IntervalSecs; got != 5 {
		t.Fatalf("persisted interval = %d", got)
	}
}

func TestSessionIntervalOverrideNotPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := New(Options{Source: mockSource(), Config: config.Default(), ConfigPath: path, Ops: &fakeOps{}, Interval: 10, Now: func() time.Time { return fixedNow }})
	if m.cfg.IntervalSecs != 10 {
		t.Fatal("override not applied")
	}
	m = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = update(m, press("w")) // any config change triggers a save
	if got := config.Load(path).Config.IntervalSecs; got != 2 {
		t.Fatalf("session override leaked to disk: %d", got)
	}
}

func TestIntervalChangeInvalidatesOldTicks(t *testing.T) {
	m := ready(t, 120, 40)
	gen := m.tickGen
	c := m.cfg
	c.IntervalSecs = 5
	m.applyConfig(c)
	if m.tickGen == gen {
		t.Fatal("tick generation not bumped")
	}
	m.collecting = false
	tm, _ := m.Update(tickMsg{gen: gen})
	if tm.(Model).collecting {
		t.Fatal("stale tick started a collection")
	}
}

func TestNoOverlappingCollections(t *testing.T) {
	src := mockSource()
	m, _ := newModel(t, src)
	cmd := m.collectCmd()
	if cmd == nil || !m.collecting {
		t.Fatal("first collection should start")
	}
	if m.collectCmd() != nil {
		t.Fatal("second collection must not start while one is running")
	}
}

func TestRenderAllPagesAllSizes(t *testing.T) {
	sizes := [][2]int{{64, 20}, {80, 24}, {100, 30}, {120, 40}, {160, 48}, {220, 60}}
	for _, sz := range sizes {
		m := ready(t, sz[0], sz[1])
		for i := range m.pages {
			m = update(m, press(string(rune('1'+i))))
			assertScreen(t, m.Render(), sz[0], sz[1], m.pages[m.active].ID())
		}
	}
}

func TestRenderASCIIAndLight(t *testing.T) {
	m := ready(t, 120, 40)
	c := m.cfg
	c.Symbols = config.SymbolsASCII
	c.Theme = config.ThemeLight
	m.applyConfig(c)
	for i := range m.pages {
		m = update(m, press(string(rune('1'+i))))
		out := m.Render()
		assertScreen(t, out, 120, 40, m.pages[m.active].ID())
		for _, r := range out {
			if (r >= 0x2800 && r <= 0x28FF) || r == '╭' || r == '━' {
				t.Fatalf("%s: unicode glyph %q in ASCII mode", m.pages[m.active].ID(), r)
			}
		}
	}
}

func TestMissingMetricsRenderGracefully(t *testing.T) {
	src := &metrics.MockSource{HostInfo: metrics.SampleHost(), Next: func(int) metrics.Snapshot { return metrics.EmptySnapshot(fixedNow) }}
	m, _ := newModel(t, src)
	m = update(m, tea.WindowSizeMsg{Width: 140, Height: 44})
	m = update(m, snapshotMsg{snap: metrics.EmptySnapshot(fixedNow)})
	for i := range m.pages {
		m = update(m, press(string(rune('1'+i))))
		out := m.Render()
		assertScreen(t, out, 140, 44, m.pages[m.active].ID())
		if strings.Contains(out, "NaN") {
			t.Errorf("%s: NaN rendered", m.pages[m.active].ID())
		}
	}
	m = update(m, press("1"))
	if !strings.Contains(m.Render(), "unavailable") {
		t.Error("overview should state that metrics are unavailable")
	}
	if len(m.alerts.Active()) != 0 {
		t.Error("alerts raised from missing data")
	}
}

func TestDesktopMacBatteryPage(t *testing.T) {
	m := ready(t, 120, 40)
	s := metrics.SampleSnapshot(fixedNow, 1)
	s.Battery = metrics.BatteryStats{Valid: true}
	m = update(m, snapshotMsg{snap: s})
	m = update(m, press("8"))
	if !strings.Contains(m.Render(), "This Mac has no battery") {
		t.Error("desktop battery state missing")
	}
}

func TestFanlessSensorsPage(t *testing.T) {
	m := ready(t, 140, 44)
	s := metrics.SampleSnapshot(fixedNow, 1)
	s.Sensors.Fans, s.Sensors.FanState = nil, metrics.FansNone
	s.Sensors.Power = nil
	m = update(m, snapshotMsg{snap: s})
	m = update(m, press("7"))
	out := m.Render()
	if !strings.Contains(out, "No fans") || !strings.Contains(out, "Unavailable on this Mac") {
		t.Error("fanless / unavailable states missing")
	}
	if strings.Contains(out, "0 rpm") {
		t.Error("fake zero fan speed rendered")
	}
}

func TestTooSmallTerminal(t *testing.T) {
	m := ready(t, 40, 10)
	if !strings.Contains(m.Render(), "Terminal too small") {
		t.Error("small terminal message missing")
	}
}

func TestAlertsOverlayDismiss(t *testing.T) {
	m := ready(t, 120, 40) // sample snapshot has a nearly full external volume
	if len(m.alerts.Active()) != 1 {
		t.Fatalf("expected one alert, got %d", len(m.alerts.Active()))
	}
	m = update(m, press("a"))
	if !strings.Contains(m.Render(), "Low storage on Backup") {
		t.Fatal("alert not listed")
	}
	m = update(m, press("d"))
	if len(m.alerts.Active()) != 0 {
		t.Fatal("dismiss failed")
	}
	m = update(m, press("esc"))
	if m.showAlerts {
		t.Fatal("esc should close alerts")
	}
}

func assertScreen(t *testing.T, out string, w, h int, page string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != h {
		t.Errorf("%s at %dx%d: %d lines", page, w, h, len(lines))
	}
	for i, l := range lines {
		if lw := lipgloss.Width(l); lw > w {
			t.Errorf("%s at %dx%d: line %d is %d wide", page, w, h, i, lw)
		}
	}
}

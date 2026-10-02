package pages

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
)

// Memory is the memory page.
type Memory struct{ buf []float64 }

func NewMemory() *Memory { return &Memory{} }

func (*Memory) ID() string                                          { return "memory" }
func (*Memory) Title() string                                       { return "Memory" }
func (*Memory) Capturing() bool                                     { return false }
func (*Memory) Help(*Context) []key.Binding                         { return []key.Binding{keyWindow} }
func (*Memory) HandleKey(*Context, tea.KeyPressMsg) (tea.Cmd, bool) { return nil, false }
func (*Memory) Update(*Context, tea.Msg) tea.Cmd                    { return nil }

func (p *Memory) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t, m := c.Theme, c.Snap.Memory
	if !m.Valid {
		return components.Panel(t, "Memory", "", errLine(t, "Memory statistics", m.Err), w, min(h, 5), false)
	}
	stack := w < 96
	leftW := 46
	if stack {
		leftW = w
	}
	topH := 10
	summary := components.Panel(t, "Memory", t.Muted.Render(format.Bytes(m.Total)+" installed"), p.summary(c, leftW-4), leftW, topH, false)
	top := summary
	if !stack {
		cw := w - leftW - 1
		iw, ih := components.PanelContentSize(cw, topH)
		p.buf = series(c, c.Hist.Memory, p.buf)
		chart := components.AreaChart(t, p.buf, iw, ih, 100, t.P.Secondary, pctLabel)
		top = components.Columns(1, summary, components.Panel(t, "Usage History", t.Muted.Render(windowLabel(c.Cfg.WindowSecs)), chart, cw, topH, false))
	}
	restH := h - topH
	if restH < 6 {
		return components.Fit(top, w, h)
	}
	bw := w
	pw := 0
	if w >= 110 {
		bw = w * 11 / 20
		pw = w - bw - 1
	}
	breakdown := components.Panel(t, "What's using memory", "", p.breakdown(c, bw-4), bw, restH, false)
	bottom := breakdown
	if pw > 0 {
		iw, ih := components.PanelContentSize(pw, restH)
		body := errLine(t, "Process list", c.Snap.Processes.Err)
		if c.Snap.Processes.Valid {
			body = procTable(c, TopProcesses(c.Snap.Processes.List, max(ih-1, 1), true), iw, true)
		}
		bottom = components.Columns(1, breakdown, components.Panel(t, "Top Memory", "", body, pw, restH, false))
	}
	return components.Fit(top+"\n"+bottom, w, h)
}

func (p *Memory) summary(c *Context, w int) string {
	t, m := c.Theme, c.Snap.Memory
	pc := pressureColor(t, m.Pressure)
	ps := lipgloss.NewStyle().Foreground(pc)
	big := t.BigValue.Render(format.Bytes(m.Used)) + t.Muted.Render(" used of "+format.Bytes(m.Total)) +
		t.Faint.Render(fmt.Sprintf("  (%s)", format.Percent(m.UsedPercent)))
	// The pressure gauge shows how close the kernel is to compressing and
	// swapping aggressively, derived from memorystatus_level.
	gauge := 0.0
	gaugeTxt := ""
	if m.FreePercent.OK {
		gauge = 100 - m.FreePercent.V
		gaugeTxt = fmt.Sprintf(" %3.0f%%", gauge)
	}
	lines := []string{
		big,
		"",
		components.KV(t, "Memory pressure", ps.Bold(true).Render(t.Sym.Dot+" "+m.Pressure.String()), w),
		components.Bar(t, gauge, w-len(gaugeTxt), pc) + t.Muted.Render(gaugeTxt),
		"",
	}
	swap := t.Muted.Render("unavailable")
	if m.Swap.OK {
		sw := m.Swap.V
		swap = format.Bytes(sw.Used) + t.Muted.Render(" of "+format.Bytes(sw.Total))
		if sw.Total == 0 {
			swap = t.Muted.Render("none")
		}
	}
	lines = append(lines,
		components.KV(t, "Available", format.Bytes(m.Available), w),
		components.KV(t, "Swap used", swap, w),
	)
	return strings.Join(lines, "\n")
}

type memRow struct {
	label string
	value uint64
	color color.Color
	note  string
}

func (p *Memory) breakdown(c *Context, w int) string {
	t, m := c.Theme, c.Snap.Memory
	rows := []memRow{
		{"App Memory", m.App, t.P.Accent, "Used by apps and processes"},
		{"Wired", m.Wired, t.P.Secondary, "Kernel memory; always resident"},
		{"Compressed", m.Compressed, t.P.Tertiary, "Inactive memory compressed in RAM"},
		{"Cached Files", m.Cached, t.P.Faint, "File cache, reclaimed on demand"},
		{"Free", m.Free, t.P.Track, "Unused; macOS fills RAM with cache"},
	}
	segs := []components.Segment{
		{Value: float64(m.App), Color: t.P.Accent},
		{Value: float64(m.Wired), Color: t.P.Secondary},
		{Value: float64(m.Compressed), Color: t.P.Tertiary},
		{Value: float64(min(m.Cached, m.Total-min(m.Used, m.Total))), Color: t.P.Faint},
	}
	lines := []string{components.StackedBar(t, segs, float64(m.Total), w), ""}
	labelW, valW := 13, 9
	noteW := w - labelW - valW - 4
	for _, r := range rows {
		dot := lipgloss.NewStyle().Foreground(r.color).Render(t.Sym.Dot)
		line := dot + " " + t.Value.Render(format.PadRight(r.label, labelW)) +
			fmt.Sprintf("%*s", valW, format.Bytes(r.value))
		if noteW > 12 {
			line += "  " + t.Faint.Render(format.Truncate(r.note, noteW))
		}
		lines = append(lines, line)
	}
	lines = append(lines, "",
		t.Muted.Render("Used = App + Wired + Compressed. Cached files are not wasted"),
		t.Muted.Render("RAM; memory pressure is the real health signal."),
		"",
	)
	vm := func(label string, v uint64) string {
		return t.Label.Render(label+" ") + t.Value.Render(format.Bytes(v))
	}
	sep := t.Faint.Render("  " + t.Sym.Sep + "  ")
	lines = append(lines,
		components.Fit(vm("Active", m.Active)+sep+vm("Inactive", m.Inactive), w, 1),
		components.Fit(vm("Purgeable", m.Purgeable)+sep+vm("Speculative", m.Speculative), w, 1))
	return strings.Join(lines, "\n")
}

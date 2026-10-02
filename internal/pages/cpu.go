package pages

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/config"
	"github.com/literalmock/mifo/internal/format"
)

// CPU is the detailed processor page.
type CPU struct {
	scroll   int
	coreRows int // rows visible in the last render
	buf      []float64
}

func NewCPU() *CPU { return &CPU{} }

func (*CPU) ID() string      { return "cpu" }
func (*CPU) Title() string   { return "CPU" }
func (*CPU) Capturing() bool { return false }

var (
	keyFaster = key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+/-", "sampling interval"))
	keySlower = key.NewBinding(key.WithKeys("-", "_"))
	keyScroll = key.NewBinding(key.WithKeys("j", "k", "down", "up"), key.WithHelp("j/k", "scroll cores"))
)

func (*CPU) Help(*Context) []key.Binding { return []key.Binding{keyScroll, keyFaster, keyWindow} }

// stepInterval returns a config with the sampling interval moved by dir.
func stepInterval(cfg config.Config, dir int) config.Config {
	i := slices.Index(config.Intervals, cfg.IntervalSecs)
	i = min(max(i+dir, 0), len(config.Intervals)-1)
	cfg.IntervalSecs = config.Intervals[i]
	return cfg
}

func intervalCmd(c *Context, dir int) tea.Cmd {
	cfg := stepInterval(c.Cfg, dir)
	return tea.Batch(
		func() tea.Msg { return ConfigChangedMsg{Cfg: cfg} },
		func() tea.Msg { return StatusMsg{Text: fmt.Sprintf("Sampling every %ds", cfg.IntervalSecs)} },
	)
}

func (p *CPU) HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, keyFaster):
		return intervalCmd(c, -1), true
	case key.Matches(msg, keySlower):
		return intervalCmd(c, +1), true
	case key.Matches(msg, keyDown):
		n := len(c.Snap.CPU.PerCore)
		p.scroll = min(p.scroll+1, max(n-p.coreRows, 0))
		return nil, true
	case key.Matches(msg, keyUp):
		p.scroll = max(p.scroll-1, 0)
		return nil, true
	}
	return nil, false
}

func (*CPU) Update(*Context, tea.Msg) tea.Cmd { return nil }

func (p *CPU) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t, s := c.Theme, c.Snap.CPU
	if !s.Valid {
		return components.Panel(t, "CPU", "", errLine(t, "CPU statistics", s.Err), w, min(h, 5), false)
	}
	leftW := 38
	stack := w < 96
	if stack {
		leftW = w
	}
	topH := min(18, h/2+3)
	if stack {
		topH = min(12, h/3+2)
	}
	topH = max(topH, 8)

	util := components.Panel(t, "Utilization", "", p.utilBody(c, leftW-4, topH-2), leftW, topH, false)
	win := windowLabel(c.Cfg.WindowSecs)
	var top string
	if stack {
		top = util
	} else {
		cw := w - leftW - 1
		iw, ih := components.PanelContentSize(cw, topH)
		p.buf = series(c, c.Hist.CPU, p.buf)
		chart := components.AreaChart(t, p.buf, iw, ih, 100, t.P.Accent, pctLabel)
		top = components.Columns(1, util, components.Panel(t, "History", t.Muted.Render(win), chart, cw, topH, false))
	}
	restH := h - topH
	if restH < 5 {
		return components.Fit(top, w, h)
	}
	coresW := w
	procW := 0
	if w >= 100 {
		coresW = w / 2
		procW = w - coresW - 1
	}
	info := ""
	if n := len(s.PerCore); n > 0 {
		info = t.Muted.Render(fmt.Sprintf("%d logical", n))
	}
	coreBody := p.coresBody(c, coresW-4, restH-2)
	if n := len(s.PerCore); n > p.coreRows {
		info = t.Muted.Render(fmt.Sprintf("%d-%d of %d", p.scroll+1, min(p.scroll+p.coreRows, n), n))
	}
	cores := components.Panel(t, "Per-core", info, coreBody, coresW, restH, focused)
	bottom := cores
	if procW > 0 {
		pw, ph := components.PanelContentSize(procW, restH)
		body := errLine(t, "Process list", c.Snap.Processes.Err)
		if c.Snap.Processes.Valid {
			body = procTable(c, TopProcesses(c.Snap.Processes.List, max(ph-1, 1), false), pw, false)
		}
		bottom = components.Columns(1, cores, components.Panel(t, "Top CPU", "", body, procW, restH, false))
	}
	return components.Fit(top+"\n"+bottom, w, h)
}

func (p *CPU) utilBody(c *Context, w, h int) string {
	t, s := c.Theme, c.Snap.CPU
	col := cpuColor(t, s.Total)
	big := components.BigText(fmt.Sprintf("%.0f%%", s.Total), t.Sym.Unicode)
	bs := lipgloss.NewStyle().Foreground(col)
	var lines []string
	for _, r := range big {
		lines = append(lines, bs.Render(r))
	}
	lines = append(lines, "")
	barW := max(w-20, 4)
	brk := func(label string, v float64, colr lipgloss.Style) string {
		return fmt.Sprintf("%s %s %s",
			t.Label.Render(format.PadRight(label, 7)),
			components.Bar(t, v, barW, colr.GetForeground()),
			t.Value.Render(fmt.Sprintf("%6.1f%%", v)))
	}
	lines = append(lines,
		brk("User", s.User, t.Accent),
		brk("System", s.System, t.Second),
		brk("Nice", s.Nice, t.Third),
		brk("Idle", s.Idle, t.Faint),
		"",
	)
	if s.Load.OK {
		l := s.Load.V
		lines = append(lines, components.KV(t, "Load average", fmt.Sprintf("%.2f  %.2f  %.2f", l[0], l[1], l[2]), w))
	} else {
		lines = append(lines, components.KV(t, "Load average", t.Muted.Render("unavailable"), w))
	}
	freq := t.Muted.Render("unavailable")
	if s.MaxFreq.OK {
		freq = format.Freq(s.MaxFreq.V) + t.Muted.Render(" max")
	}
	lines = append(lines, components.KV(t, "Frequency", freq, w))
	coreInfo := fmt.Sprintf("%d", len(s.PerCore))
	if c.Host.PCores > 0 {
		coreInfo = fmt.Sprintf("%d performance %s %d efficiency", c.Host.PCores, t.Sym.Sep, c.Host.ECores)
	}
	lines = append(lines, components.KV(t, "Cores", coreInfo, w))
	th := c.Snap.Sensors.Thermal
	lines = append(lines, components.KV(t, "Thermal pressure", lipgloss.NewStyle().Foreground(thermalColor(t, th)).Render(th.String()), w))
	lines = append(lines, components.KV(t, "Sampling", fmt.Sprintf("every %ds", c.Cfg.IntervalSecs)+t.Faint.Render(" (+/-)"), w))
	return strings.Join(lines, "\n")
}

func (p *CPU) coresBody(c *Context, w, h int) string {
	t, s := c.Theme, c.Snap.CPU
	n := len(s.PerCore)
	if n == 0 {
		return t.Muted.Render("Per-core data unavailable")
	}
	cols := 1
	if w >= 70 && n > h {
		cols = 2
	}
	colW := components.Split(w, cols, 3)
	rows := h
	p.coreRows = rows * cols
	p.scroll = min(p.scroll, max(n-p.coreRows, 0))
	sparkW := 0
	if colW[0] >= 44 {
		sparkW = 12
	}
	line := func(i, cw int) string {
		v := s.PerCore[i]
		kind := ""
		if i < len(s.CoreKinds) {
			kind = s.CoreKinds[i]
		}
		label := fmt.Sprintf("%s%-2d", kind, i)
		barW := cw - 5 - 7 - sparkW
		if sparkW > 0 {
			barW--
		}
		out := t.Muted.Render(format.PadRight(label, 4)) + " "
		if sparkW > 0 && i < len(c.Hist.Cores) {
			buf := series(c, c.Hist.Cores[i], nil)
			out += components.Sparkline(t, buf, sparkW, 100, t.P.Faint) + " "
		}
		out += components.Bar(t, v, max(barW, 2), cpuColor(t, v)) +
			lipgloss.NewStyle().Foreground(cpuColor(t, v)).Render(fmt.Sprintf("%6.1f%%", v))
		return out
	}
	var out []string
	for r := 0; r < rows; r++ {
		var parts []string
		for ci := 0; ci < cols; ci++ {
			i := p.scroll + ci*rows + r
			if i >= n {
				parts = append(parts, strings.Repeat(" ", colW[ci]))
				continue
			}
			parts = append(parts, components.Fit(line(i, colW[ci]), colW[ci], 1))
		}
		out = append(out, strings.Join(parts, "   "))
	}
	return strings.Join(out, "\n")
}

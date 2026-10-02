package pages

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
)

// Sensors shows temperatures, fans, power and thermal pressure.
type Sensors struct {
	scroll int
	rows   int
	buf    []float64
}

func NewSensors() *Sensors { return &Sensors{} }

func (*Sensors) ID() string      { return "sensors" }
func (*Sensors) Title() string   { return "Sensors" }
func (*Sensors) Capturing() bool { return false }
func (*Sensors) Help(*Context) []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "scroll sensors")), keyWindow}
}

func (s *Sensors) HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, keyDown):
		s.scroll = min(s.scroll+1, max(len(c.Snap.Sensors.Temps)-s.rows, 0))
		return nil, true
	case key.Matches(msg, keyUp):
		s.scroll = max(s.scroll-1, 0)
		return nil, true
	}
	return nil, false
}

func (*Sensors) Update(*Context, tea.Msg) tea.Cmd { return nil }

func (s *Sensors) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t := c.Theme
	leftW := 44
	stack := w < 96
	if stack {
		leftW = w
	}
	topH := 11
	summary := components.Panel(t, "Thermal", "", s.summary(c, leftW-4), leftW, topH, false)
	top := summary
	if !stack {
		cw := w - leftW - 1
		iw, ih := components.PanelContentSize(cw, topH)
		var body string
		if c.Snap.Sensors.CPUTemp.OK {
			s.buf = series(c, c.Hist.CPUTemp, s.buf)
			mx := components.NiceMax(max(components.MaxOf(s.buf), 50))
			body = components.AreaChart(t, s.buf, iw, ih, mx, t.P.Warn, func(v float64) string { return fmt.Sprintf("%.0f%s", v, t.Sym.Degree) })
		} else {
			body = components.Unavailable(t, "Unavailable on this Mac", "No readable SoC temperature sensors were found.", iw)
		}
		top = components.Columns(1, summary, components.Panel(t, "SoC Temperature", t.Muted.Render(windowLabel(c.Cfg.WindowSecs)), body, cw, topH, false))
	}
	restH := h - topH
	if restH < 6 {
		return components.Fit(top, w, h)
	}
	listW := w
	sideW := 0
	if w >= 96 {
		listW = w / 2
		sideW = w - listW - 1
	}
	lw, lh := components.PanelContentSize(listW, restH)
	body := s.list(c, lw, lh)
	info := fmt.Sprintf("%d readings", len(c.Snap.Sensors.Temps))
	if n := len(c.Snap.Sensors.Temps); n > lh {
		info = fmt.Sprintf("%d-%d of %d", s.scroll+1, min(s.scroll+lh, n), n)
	}
	list := components.Panel(t, "Temperature Sensors", t.Muted.Render(info), body, listW, restH, focused)
	bottom := list
	if sideW > 0 {
		fanH := 5
		provH := len(c.Snap.Sensors.Providers) + 2
		powH := max(restH-fanH-provH, 4)
		fw, _ := components.PanelContentSize(sideW, fanH)
		side := components.Panel(t, "Fans", "", s.fans(c, fw), sideW, fanH, false) + "\n" +
			components.Panel(t, "Power", t.Faint.Render("SMC"), s.power(c, fw), sideW, powH, false) + "\n" +
			components.Panel(t, "Sensor Backends", "", s.providers(c, fw), sideW, restH-fanH-powH, false)
		bottom = components.Columns(1, list, side)
	}
	return components.Fit(top+"\n"+bottom, w, h)
}

func (s *Sensors) summary(c *Context, w int) string {
	t, ss := c.Theme, c.Snap.Sensors
	tc := lipgloss.NewStyle().Foreground(thermalColor(t, ss.Thermal))
	lines := []string{
		t.Label.Render("Thermal pressure"),
		tc.Bold(true).Render(t.Sym.Dot + " " + ss.Thermal.String()),
		"",
	}
	if ss.CPUTemp.OK {
		lines = append(lines,
			components.KV(t, "SoC average", tempStr(t, ss.CPUTemp.V), w),
			components.KV(t, "SoC hottest", lipgloss.NewStyle().Foreground(t.LevelColor(ss.CPUMax.V, 80, 95)).Render(tempStr(t, ss.CPUMax.V)), w))
	} else {
		lines = append(lines, components.KV(t, "SoC temperature", t.Muted.Render("Unavailable on this Mac"), w))
	}
	if ss.GPUTemp.OK {
		lines = append(lines, components.KV(t, "GPU", tempStr(t, ss.GPUTemp.V), w))
	} else {
		lines = append(lines, components.KV(t, "GPU", t.Muted.Render("Unavailable on this Mac"), w))
	}
	note := "Thermal pressure is what macOS uses to throttle; it matters more than any single sensor."
	if c.Host.AppleSilicon && !ss.GPUTemp.OK {
		note = "Apple Silicon has no separate GPU sensor; SoC die readings cover it."
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(t.P.Faint).Width(w).Render(note))
	return strings.Join(lines, "\n")
}

func (s *Sensors) list(c *Context, w, h int) string {
	t, ss := c.Theme, c.Snap.Sensors
	if len(ss.Temps) == 0 {
		return components.Unavailable(t, "Unavailable on this Mac", "No temperature sensors could be read without elevated privileges.", w)
	}
	s.rows = h
	s.scroll = min(s.scroll, max(len(ss.Temps)-h, 0))
	barW := max(w-28-10, 4)
	var lines []string
	group := ""
	for i := s.scroll; i < len(ss.Temps) && len(lines) < h; i++ {
		tp := ss.Temps[i]
		g := ""
		if tp.Group != group {
			g = tp.Group
			group = tp.Group
		}
		col := t.LevelColor(tp.Celsius, 80, 95)
		lines = append(lines, fmt.Sprintf("%s %s %s %s",
			t.Faint.Render(format.PadRight(g, 8)),
			t.Value.Render(format.PadRight(format.Truncate(tp.Name, 18), 18)),
			components.Bar(t, tp.Celsius, barW, col),
			lipgloss.NewStyle().Foreground(col).Render(fmt.Sprintf("%8s", tempStr(t, tp.Celsius)))))
	}
	return strings.Join(lines, "\n")
}

func (s *Sensors) fans(c *Context, w int) string {
	t, ss := c.Theme, c.Snap.Sensors
	switch ss.FanState {
	case metrics.FansNone:
		return t.Value.Render("No fans") + "\n" + t.Faint.Render("This Mac is passively cooled.")
	case metrics.FansUnavailable:
		return components.Unavailable(t, "Unavailable on this Mac", "Fan speed could not be read from the SMC.", w)
	}
	var lines []string
	for _, f := range ss.Fans {
		v := fmt.Sprintf("%.0f rpm", f.RPM)
		if f.Max.OK && f.Max.V > 0 {
			pct := 100 * f.RPM / f.Max.V
			v = components.Bar(t, pct, max(w-30, 4), t.P.Secondary) + " " + v + t.Faint.Render(fmt.Sprintf(" / %.0f", f.Max.V))
		}
		lines = append(lines, components.KV(t, fmt.Sprintf("Fan %d", f.Index+1), v, w))
	}
	return strings.Join(lines, "\n")
}

func (s *Sensors) power(c *Context, w int) string {
	t, ss := c.Theme, c.Snap.Sensors
	if len(ss.Power) == 0 {
		return components.Unavailable(t, "Unavailable on this Mac", "No power rails are readable via the SMC.", w)
	}
	var lines []string
	for _, p := range ss.Power {
		lines = append(lines, components.KV(t, p.Name, format.Watts(p.Watts), w))
	}
	return strings.Join(lines, "\n")
}

func (s *Sensors) providers(c *Context, w int) string {
	t := c.Theme
	var lines []string
	for _, p := range c.Snap.Sensors.Providers {
		st := t.Accent.Render(t.Sym.Check + " ok")
		if !p.OK {
			msg := "unavailable"
			if p.Err != nil {
				msg = p.Err.Error()
			}
			st = t.Muted.Render(t.Sym.Cross + " " + format.Truncate(msg, max(w-26, 8)))
		}
		lines = append(lines, components.KV(t, p.Name, st, w))
	}
	return strings.Join(lines, "\n")
}

package pages

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
)

// Battery shows battery and power information, adapting for desktops.
type Battery struct{ buf []float64 }

func NewBattery() *Battery { return &Battery{} }

func (*Battery) ID() string                                          { return "battery" }
func (*Battery) Title() string                                       { return "Battery" }
func (*Battery) Capturing() bool                                     { return false }
func (*Battery) Help(*Context) []key.Binding                         { return []key.Binding{keyWindow} }
func (*Battery) HandleKey(*Context, tea.KeyPressMsg) (tea.Cmd, bool) { return nil, false }
func (*Battery) Update(*Context, tea.Msg) tea.Cmd                    { return nil }

func (p *Battery) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t, b := c.Theme, c.Snap.Battery
	if !b.Valid {
		return components.Panel(t, "Battery", "", errLine(t, "Battery information", b.Err), w, min(h, 5), false)
	}
	if !b.Present {
		return p.desktop(c, w, h)
	}
	leftW := 46
	stack := w < 96
	if stack {
		leftW = w
	}
	topH := 11
	charge := components.Panel(t, "Charge", "", p.charge(c, leftW-4), leftW, topH, false)
	top := charge
	if !stack {
		rw := w - leftW - 1
		top = components.Columns(1, charge, components.Panel(t, "Power", "", p.power(c, rw-4), rw, topH, false))
	}
	restH := h - topH
	if restH < 6 {
		return components.Fit(top, w, h)
	}
	healthH := min(restH, 7)
	out := top + "\n" + components.Panel(t, "Health", "", p.health(c, w-4), w, healthH, false)
	if chartH := restH - healthH; chartH >= 5 {
		iw, ih := components.PanelContentSize(w, chartH)
		p.buf = series(c, c.Hist.Battery, p.buf)
		chart := components.AreaChart(t, p.buf, iw, ih, 100, batteryColor(t, b).GetForeground(), pctLabel)
		out += "\n" + components.Panel(t, "Charge History", t.Muted.Render(windowLabel(c.Cfg.WindowSecs)), chart, w, chartH, false)
	}
	return components.Fit(out, w, h)
}

func (p *Battery) charge(c *Context, w int) string {
	t, b := c.Theme, c.Snap.Battery
	st := batteryColor(t, b)
	big := components.BigText(fmt.Sprintf("%.0f%%", b.Percent), t.Sym.Unicode)
	lines := []string{}
	for _, r := range big {
		lines = append(lines, st.Render(r))
	}
	lines = append(lines, "", components.Bar(t, b.Percent, w, st.GetForeground()), "")
	lines = append(lines, components.KV(t, "State", st.Render(batteryState(b)), w))
	src := "Battery"
	if b.ACPower {
		src = "Power adapter"
		if b.AdapterWatts.OK {
			src += fmt.Sprintf(" (%.0f W)", b.AdapterWatts.V)
		}
	}
	lines = append(lines, components.KV(t, "Power source", src, w))
	switch {
	case b.TimeToEmpty.OK:
		lines = append(lines, components.KV(t, "Time remaining", format.Duration(b.TimeToEmpty.V), w))
	case b.TimeToFull.OK:
		lines = append(lines, components.KV(t, "Time until full", format.Duration(b.TimeToFull.V), w))
	default:
		lines = append(lines, components.KV(t, "Time remaining", t.Muted.Render("macOS has no reliable estimate yet"), w))
	}
	return strings.Join(lines, "\n")
}

func (p *Battery) power(c *Context, w int) string {
	t, b := c.Theme, c.Snap.Battery
	na := t.Muted.Render("unavailable")
	opt := func(ok bool, s string) string {
		if ok {
			return s
		}
		return na
	}
	flow := na
	if b.Power.OK {
		v := b.Power.V
		switch {
		case v < -0.05:
			flow = t.Warn.Render(format.Watts(-v)) + t.Muted.Render(" out of battery")
		case v > 0.05:
			flow = t.Accent.Render(format.Watts(v)) + t.Muted.Render(" into battery")
		default:
			flow = t.Muted.Render("idle")
		}
	}
	lines := []string{
		components.KV(t, "Battery flow", flow, w),
		components.KV(t, "System load", opt(b.SystemLoad.OK, format.Watts(b.SystemLoad.V)), w),
	}
	if pw, ok := systemPower(c.Snap.Sensors); ok {
		lines = append(lines, components.KV(t, "SoC + system (SMC)", format.Watts(pw), w))
	}
	lines = append(lines,
		"",
		components.KV(t, "Voltage", opt(b.Voltage.OK, fmt.Sprintf("%.2f V", b.Voltage.V)), w),
		components.KV(t, "Current", opt(b.Amperage.OK, fmt.Sprintf("%.0f mA", b.Amperage.V)), w),
		components.KV(t, "Temperature", opt(b.Temperature.OK, tempStr(t, b.Temperature.V)), w),
		"",
		t.Faint.Render("Per-app energy impact requires powermetrics (root)"),
		t.Faint.Render("and is not collected by MIFO."),
	)
	return strings.Join(lines, "\n")
}

func (p *Battery) health(c *Context, w int) string {
	t, b := c.Theme, c.Snap.Battery
	na := t.Muted.Render("not reported")
	healthPct := na
	bar := ""
	if b.Health.OK {
		col := t.LevelColor(100-b.Health.V, 20, 40)
		healthPct = lipgloss.NewStyle().Foreground(col).Bold(true).Render(fmt.Sprintf("%.1f%%", b.Health.V))
		bar = components.Bar(t, b.Health.V, max(w/2, 10), col)
	}
	capS := na
	if b.MaxCapacity.OK && b.DesignCap.OK {
		capS = fmt.Sprintf("%d mAh", b.MaxCapacity.V) + t.Muted.Render(fmt.Sprintf(" of %d mAh design", b.DesignCap.V))
	}
	cycles := na
	if b.CycleCount.OK {
		cycles = fmt.Sprint(b.CycleCount.V)
		if b.DesignCycles.OK {
			cycles += t.Muted.Render(fmt.Sprintf(" of %d rated", b.DesignCycles.V))
		}
	}
	cond := b.Condition
	if cond == "Normal" {
		cond = t.Accent.Render(t.Sym.Check + " Normal")
	} else if cond != "" {
		cond = t.Warn.Render(cond)
	}
	half := w / 2
	left := []string{
		components.KV(t, "Maximum capacity", healthPct, half-2),
		bar,
		"",
		components.KV(t, "Condition", cond, half-2),
	}
	right := []string{
		components.KV(t, "Full charge capacity", capS, w-half-2),
		components.KV(t, "Cycle count", cycles, w-half-2),
		"",
		t.Faint.Render("Capacity is the battery's current full charge"),
		t.Faint.Render("relative to its design capacity."),
	}
	return components.Columns(4, components.Fit(strings.Join(left, "\n"), half-2, 5), strings.Join(right, "\n"))
}

func (p *Battery) desktop(c *Context, w, h int) string {
	t, s := c.Theme, c.Snap.Sensors
	lines := []string{
		t.Title.Render("This Mac has no battery"),
		t.Muted.Render("Desktop Macs run on mains power; battery metrics do not apply."),
		"",
	}
	if len(s.Power) > 0 {
		for _, p := range s.Power {
			lines = append(lines, components.KV(t, p.Name, format.Watts(p.Watts), min(w-4, 60)))
		}
	} else {
		lines = append(lines, t.Muted.Render("Power measurements are unavailable on this Mac."))
	}
	tc := lipgloss.NewStyle().Foreground(thermalColor(t, s.Thermal))
	lines = append(lines, "", components.KV(t, "Thermal pressure", tc.Render(s.Thermal.String()), min(w-4, 60)))
	return components.Panel(t, "Power", "", strings.Join(lines, "\n"), w, min(h, len(lines)+2), false)
}

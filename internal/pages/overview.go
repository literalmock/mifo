package pages

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/alerts"
	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
	"github.com/literalmock/mifo/internal/styles"
)

// Overview is the dashboard.
type Overview struct {
	buf, buf2 []float64
}

func NewOverview() *Overview { return &Overview{} }

func (*Overview) ID() string      { return "overview" }
func (*Overview) Title() string   { return "Overview" }
func (*Overview) Capturing() bool { return false }

var keyOpenProcs = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "process manager"))

func (*Overview) Help(*Context) []key.Binding {
	return []key.Binding{keyOpenProcs, keyWindow}
}

var keyWindow = key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "graph window"))

func (*Overview) HandleKey(_ *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if key.Matches(msg, keyOpenProcs) {
		return func() tea.Msg { return NavigateMsg{Page: "processes"} }, true
	}
	return nil, false
}

func (*Overview) Update(*Context, tea.Msg) tea.Cmd { return nil }

type card struct {
	id    string
	title string
	info  string
	body  func(w, h int) string
}

func (o *Overview) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t := c.Theme
	var cards []card
	add := func(id, title string, info string, body func(w, h int) string) {
		if c.Cfg.Widgets[id] {
			cards = append(cards, card{id, title, info, body})
		}
	}
	add("cpu", "CPU", "", func(w, h int) string { return o.cpuCard(c, w, h) })
	add("memory", "Memory", "", func(w, h int) string { return o.memCard(c, w, h) })
	add("disk", "Disk", "", func(w, h int) string { return o.diskCard(c, w, h) })
	add("network", "Network", "", func(w, h int) string { return o.netCard(c, w, h) })
	add("battery", batteryTitle(c), "", func(w, h int) string { return o.batteryCard(c, w, h) })
	add("thermal", "Thermal", "", func(w, h int) string { return o.thermalCard(c, w, h) })

	var rows []string
	used := 0
	const cardH = 7
	if len(cards) > 0 {
		cols := 3
		switch {
		case w >= 190 && len(cards) >= 6:
			cols = 6
		case w < 96:
			cols = 2
		}
		if w < 60 {
			cols = 1
		}
		cols = min(cols, len(cards))
		nrows := (len(cards) + cols - 1) / cols
		// Drop card rows that do not fit, keeping at least one.
		maxRows := max((h)/cardH, 1)
		nrows = min(nrows, maxRows)
		widths := components.Split(w, cols, 1)
		for r := 0; r < nrows; r++ {
			var blocks []string
			for ci := 0; ci < cols; ci++ {
				i := r*cols + ci
				if i >= len(cards) {
					blocks = append(blocks, strings.Repeat(" ", widths[ci]))
					continue
				}
				cw, ch := components.PanelContentSize(widths[ci], cardH)
				blocks = append(blocks, components.Panel(t, cards[i].title, cards[i].info, cards[i].body(cw, ch), widths[ci], cardH, false))
			}
			rows = append(rows, components.Columns(1, blocks...))
			used += cardH
		}
	}

	rem := h - used
	showCharts := c.Cfg.Widgets["charts"]
	showProcs := c.Cfg.Widgets["processes"]
	procH := 0
	if showProcs && rem >= 8 {
		procH = min(9, rem)
		if showCharts && rem-procH < 7 {
			// Prefer charts when space is short; processes have their own page.
			if rem >= 14 {
				procH = 7
			} else {
				procH = 0
			}
		}
	}
	chartH := rem - procH
	switch {
	case showCharts && chartH >= 5:
		rows = append(rows, o.charts(c, w, chartH))
	case procH > 0:
		procH = rem
	case showProcs && rem >= 2 && c.Snap.Processes.Valid:
		// Too little room for a panel: show a borderless mini table.
		rows = append(rows, procTable(c, TopProcesses(c.Snap.Processes.List, rem-1, false), w, false))
	}
	if procH > 0 {
		rows = append(rows, o.bottom(c, w, procH, focused))
	}
	return components.Fit(strings.Join(rows, "\n"), w, h)
}

func batteryTitle(c *Context) string {
	if c.Snap.Battery.Valid && !c.Snap.Battery.Present {
		return "Power"
	}
	return "Battery"
}

func bigLine(t styles.Theme, big string, bigStyle lipgloss.Style, right string, w int) string {
	l := bigStyle.Bold(true).Render(big)
	r := t.Muted.Render(right)
	gap := w - lipgloss.Width(l) - lipgloss.Width(r)
	if gap < 1 {
		return l
	}
	return l + strings.Repeat(" ", gap) + r
}

func (o *Overview) cpuCard(c *Context, w, h int) string {
	t, s := c.Theme, c.Snap.CPU
	if !s.Valid {
		return errLine(t, "CPU", s.Err)
	}
	cores := fmt.Sprintf("%d cores", len(s.PerCore))
	if c.Host.PCores > 0 {
		cores = fmt.Sprintf("%dP + %dE", c.Host.PCores, c.Host.ECores)
	}
	if s.MaxFreq.OK {
		cores += " " + t.Sym.Sep + " " + format.Freq(s.MaxFreq.V)
	}
	lines := []string{
		bigLine(t, format.Percent(s.Total), lipgloss.NewStyle().Foreground(cpuColor(t, s.Total)), cores, w),
		components.Bar(t, s.Total, w, cpuColor(t, s.Total)),
		components.KV(t, "User / System", fmt.Sprintf("%s / %s", format.Percent(s.User), format.Percent(s.System)), w),
		coreStrip(t, s, w),
	}
	o.buf = series(c, c.Hist.CPU, o.buf)
	lines = append(lines, components.Sparkline(t, o.buf, w, 100, t.P.Accent))
	return strings.Join(lines, "\n")
}

// coreStrip draws one block per core, grouped by kind.
func coreStrip(t styles.Theme, s metrics.CPUStats, w int) string {
	var sb strings.Builder
	sb.WriteString(t.Label.Render("Cores "))
	last := "?"
	for i, v := range s.PerCore {
		k := ""
		if i < len(s.CoreKinds) {
			k = s.CoreKinds[i]
		}
		if k != last && k != "" {
			if last != "?" {
				sb.WriteString(" ")
			}
			sb.WriteString(t.Faint.Render(k))
		}
		last = k
		levels := t.Sym.Spark
		idx := int(v/100*float64(len(levels)-1) + 0.5)
		idx = min(max(idx, 1), len(levels)-1)
		sb.WriteString(lipgloss.NewStyle().Foreground(cpuColor(t, v)).Render(levels[idx]))
	}
	return components.Fit(sb.String(), w, 1)
}

func memSegments(t styles.Theme, m metrics.MemoryStats) []components.Segment {
	return []components.Segment{
		{Value: float64(m.App), Color: t.P.Accent},
		{Value: float64(m.Wired), Color: t.P.Secondary},
		{Value: float64(m.Compressed), Color: t.P.Tertiary},
	}
}

func (o *Overview) memCard(c *Context, w, h int) string {
	t, m := c.Theme, c.Snap.Memory
	if !m.Valid {
		return errLine(t, "Memory", m.Err)
	}
	pressure := lipgloss.NewStyle().Foreground(pressureColor(t, m.Pressure)).Render(t.Sym.Dot + " " + m.Pressure.String())
	swap := t.Muted.Render("n/a")
	if m.Swap.OK {
		swap = format.Bytes(m.Swap.V.Used) + t.Muted.Render(" / "+format.Bytes(m.Swap.V.Total))
	}
	lines := []string{
		bigLine(t, format.Bytes(m.Used), t.Value, "of "+format.Bytes(m.Total), w),
		components.StackedBar(t, memSegments(t, m), float64(m.Total), w),
		components.KV(t, "Pressure", pressure, w),
		components.KV(t, "Swap", swap, w),
	}
	o.buf = series(c, c.Hist.Memory, o.buf)
	lines = append(lines, components.Sparkline(t, o.buf, w, 100, t.P.Secondary))
	return strings.Join(lines, "\n")
}

// primaryVolume picks the volume that holds user data.
func primaryVolume(d metrics.DiskStats) (metrics.Volume, bool) {
	for _, v := range d.Volumes {
		if v.Mount == "/System/Volumes/Data" {
			return v, true
		}
	}
	for _, v := range d.Volumes {
		if v.Mount == "/" {
			return v, true
		}
	}
	if len(d.Volumes) > 0 {
		return d.Volumes[0], true
	}
	return metrics.Volume{}, false
}

func (o *Overview) diskCard(c *Context, w, h int) string {
	t, d := c.Theme, c.Snap.Disk
	v, ok := primaryVolume(d)
	if !ok {
		return errLine(t, "Storage", d.Err)
	}
	col := t.LevelColor(v.UsedPercent, 85, 95)
	lines := []string{
		bigLine(t, format.BytesSI(v.Free), t.Value, "free of "+format.BytesSI(v.Total), w),
		components.Bar(t, v.UsedPercent, w, col),
		components.KV(t, v.Name, lipgloss.NewStyle().Foreground(col).Render(format.Percent(v.UsedPercent)+" used"), w),
	}
	ioLabel := "Read / Write"
	if w < 38 {
		ioLabel = "R / W"
	}
	if d.IOValid {
		lines = append(lines, components.KV(t, ioLabel,
			t.Accent.Render(format.Rate(d.ReadBps))+t.Muted.Render(" / ")+t.Second.Render(format.Rate(d.WriteBps)), w))
	} else {
		lines = append(lines, components.KV(t, ioLabel, t.Muted.Render("measuring…"), w))
	}
	o.buf = series(c, c.Hist.DiskR, o.buf)
	o.buf2 = series(c, c.Hist.DiskW, o.buf2)
	sum := make([]float64, len(o.buf))
	for i := range o.buf {
		sum[i] = o.buf[i]
		if i < len(o.buf2) {
			sum[i] += o.buf2[i]
		}
	}
	lines = append(lines, components.Sparkline(t, sum, w, 0, t.P.Tertiary))
	return strings.Join(lines, "\n")
}

func primaryIface(n metrics.NetworkStats) (metrics.Interface, bool) {
	for _, i := range n.Interfaces {
		if i.Primary {
			return i, true
		}
	}
	return metrics.Interface{}, false
}

func (o *Overview) netCard(c *Context, w, h int) string {
	t, n := c.Theme, c.Snap.Network
	if !n.Valid && n.Err != nil {
		return errLine(t, "Network", n.Err)
	}
	down := t.Accent.Render(t.Sym.Down+" ") + t.Value.Bold(true).Render(format.Rate(n.RxBps))
	up := t.Second.Render(t.Sym.Up+" ") + t.Value.Bold(true).Render(format.Rate(n.TxBps))
	status := t.Crit.Render(t.Sym.Dot + " Offline")
	if ifc, ok := primaryIface(n); ok {
		status = t.Accent.Render(t.Sym.Dot+" ") + t.Value.Render(ifc.Kind) + t.Muted.Render(" ("+ifc.Name+")")
	}
	lines := []string{
		components.Fit(down+"   "+up, w, 1),
		components.KV(t, "Interface", status, w),
		components.KV(t, "Received", format.BytesSI(n.RxTotal), w),
		components.KV(t, "Sent", format.BytesSI(n.TxTotal), w),
	}
	o.buf = series(c, c.Hist.NetRx, o.buf)
	lines = append(lines, components.Sparkline(t, o.buf, w, 0, t.P.Accent))
	return strings.Join(lines, "\n")
}

func batteryState(b metrics.BatteryStats) string {
	switch {
	case b.FullyCharged:
		return "Fully charged"
	case b.Charging:
		return "Charging"
	case b.ACPower:
		return "On AC, not charging"
	}
	return "On battery"
}

func batteryColor(t styles.Theme, b metrics.BatteryStats) lipgloss.Style {
	switch {
	case b.Charging || b.ACPower:
		return t.Accent
	case b.Percent <= 10:
		return t.Crit
	case b.Percent <= 20:
		return t.Warn
	}
	return t.Accent
}

func systemPower(s metrics.SensorStats) (float64, bool) {
	for _, p := range s.Power {
		if p.Name == "System total" {
			return p.Watts, true
		}
	}
	return 0, false
}

func (o *Overview) batteryCard(c *Context, w, h int) string {
	t, b := c.Theme, c.Snap.Battery
	if !b.Valid {
		return errLine(t, "Battery", b.Err)
	}
	if !b.Present {
		lines := []string{t.Value.Bold(true).Render("AC power"), t.Muted.Render("This Mac has no battery")}
		if pw, ok := systemPower(c.Snap.Sensors); ok {
			lines = append(lines, components.KV(t, "System power", format.Watts(pw), w))
		}
		return strings.Join(lines, "\n")
	}
	st := batteryColor(t, b)
	lines := []string{
		bigLine(t, fmt.Sprintf("%.0f%%", b.Percent), st, batteryState(b), w),
		components.Bar(t, b.Percent, w, st.GetForeground()),
	}
	health := t.Muted.Render("n/a")
	if b.Health.OK {
		health = fmt.Sprintf("%.0f%%", b.Health.V)
		if b.CycleCount.OK {
			health += t.Muted.Render(fmt.Sprintf(" %s %d cycles", t.Sym.Sep, b.CycleCount.V))
		}
	}
	lines = append(lines, components.KV(t, "Health", health, w))
	switch {
	case b.TimeToEmpty.OK:
		lines = append(lines, components.KV(t, "Time remaining", format.Duration(b.TimeToEmpty.V), w))
	case b.TimeToFull.OK:
		lines = append(lines, components.KV(t, "Until full", format.Duration(b.TimeToFull.V), w))
	default:
		lines = append(lines, components.KV(t, "Time remaining", t.Muted.Render("not estimated"), w))
	}
	if b.Power.OK {
		lines = append(lines, components.KV(t, "Battery power", format.Watts(b.Power.V), w))
	}
	return strings.Join(lines, "\n")
}

func (o *Overview) thermalCard(c *Context, w, h int) string {
	t, s := c.Theme, c.Snap.Sensors
	pressure := lipgloss.NewStyle().Foreground(thermalColor(t, s.Thermal)).Render(t.Sym.Dot + " " + s.Thermal.String())
	var lines []string
	if s.CPUTemp.OK {
		right := ""
		if s.CPUMax.OK {
			right = "max " + tempStr(t, s.CPUMax.V)
		}
		lines = append(lines, bigLine(t, tempStr(t, s.CPUTemp.V), lipgloss.NewStyle().Foreground(t.LevelColor(s.CPUMax.V, 80, 95)), right, w))
	} else {
		lines = append(lines, t.Muted.Render("Temperature unavailable"))
	}
	lines = append(lines, components.KV(t, "Thermal pressure", pressure, w))
	fans := ""
	switch s.FanState {
	case metrics.FansNone:
		fans = t.Muted.Render("none (fanless)")
	case metrics.FansUnavailable:
		fans = t.Muted.Render("unavailable")
	default:
		var parts []string
		for _, f := range s.Fans {
			parts = append(parts, fmt.Sprintf("%.0f", f.RPM))
		}
		fans = strings.Join(parts, " / ") + t.Muted.Render(" rpm")
	}
	lines = append(lines, components.KV(t, "Fans", fans, w))
	if pw, ok := systemPower(s); ok {
		lines = append(lines, components.KV(t, "System power", format.Watts(pw), w))
	} else {
		lines = append(lines, components.KV(t, "System power", t.Muted.Render("unavailable"), w))
	}
	o.buf = series(c, c.Hist.CPUTemp, o.buf)
	if len(o.buf) > 0 {
		lo := components.MaxOf(o.buf)
		for _, v := range o.buf {
			lo = min(lo, v)
		}
		lines = append(lines, components.SparklineRange(t, o.buf, w, lo-2, components.MaxOf(o.buf)+1, t.P.Warn))
	}
	return strings.Join(lines, "\n")
}

func (o *Overview) charts(c *Context, w, h int) string {
	t := c.Theme
	win := windowLabel(c.Cfg.WindowSecs)
	type chart struct {
		title, info string
		render      func(w, h int) string
	}
	s := c.Snap
	cpuBuf := series(c, c.Hist.CPU, nil)
	memBuf := series(c, c.Hist.Memory, nil)
	rx := series(c, c.Hist.NetRx, nil)
	tx := series(c, c.Hist.NetTx, nil)
	dr := series(c, c.Hist.DiskR, nil)
	dw := series(c, c.Hist.DiskW, nil)
	charts := []chart{
		{"CPU", t.Muted.Render(win+" "+t.Sym.Sep+" ") + t.Accent.Render(format.Percent(s.CPU.Total)), func(w, h int) string {
			return components.AreaChart(t, cpuBuf, w, h, 100, t.P.Accent, pctLabel)
		}},
		{"Memory", t.Muted.Render(win+" "+t.Sym.Sep+" ") + t.Second.Render(format.Percent(s.Memory.UsedPercent)), func(w, h int) string {
			return components.AreaChart(t, memBuf, w, h, 100, t.P.Secondary, pctLabel)
		}},
		{"Network", t.Accent.Render(t.Sym.Down+format.Rate(s.Network.RxBps)) + " " + t.Second.Render(t.Sym.Up+format.Rate(s.Network.TxBps)), func(w, h int) string {
			return components.MirrorChart(t, rx, tx, w, h, t.P.Accent, t.P.Secondary, rateLabel)
		}},
		{"Disk I/O", t.Accent.Render("R "+format.Rate(s.Disk.ReadBps)) + " " + t.Second.Render("W "+format.Rate(s.Disk.WriteBps)), func(w, h int) string {
			return components.MirrorChart(t, dr, dw, w, h, t.P.Accent, t.P.Secondary, rateLabel)
		}},
	}
	cols := 4
	if w < 150 {
		cols = 2
	}
	rows := (len(charts) + cols - 1) / cols
	if rows > 1 && h < 12 {
		// Not enough height for two rows: show the first row only.
		rows = 1
	}
	rowH := components.Split(h, rows, 0)
	widths := components.Split(w, cols, 1)
	var out []string
	for r := 0; r < rows; r++ {
		var blocks []string
		for ci := 0; ci < cols; ci++ {
			i := r*cols + ci
			if i >= len(charts) {
				break
			}
			cw, ch := components.PanelContentSize(widths[ci], rowH[r])
			blocks = append(blocks, components.Panel(t, charts[i].title, charts[i].info, charts[i].render(cw, ch), widths[ci], rowH[r], false))
		}
		out = append(out, components.Columns(1, blocks...))
	}
	return strings.Join(out, "\n")
}

// TopProcesses returns the n processes with the highest CPU usage.
func TopProcesses(list []metrics.Process, n int, byMem bool) []metrics.Process {
	cp := make([]metrics.Process, len(list))
	copy(cp, list)
	sort.SliceStable(cp, func(i, j int) bool {
		if byMem {
			return cp[i].RSS > cp[j].RSS
		}
		if cp[i].CPUPercent != cp[j].CPUPercent {
			return cp[i].CPUPercent > cp[j].CPUPercent
		}
		return cp[i].RSS > cp[j].RSS
	})
	if len(cp) > n {
		cp = cp[:n]
	}
	return cp
}

func procTable(c *Context, procs []metrics.Process, w int, byMem bool) string {
	t := c.Theme
	pidW, cpuW, memW, thrW := 7, 6, 9, 4
	showThr := w >= 50
	nameW := w - pidW - cpuW - memW - 3
	if showThr {
		nameW -= thrW + 1
	}
	nameW = max(nameW, 6)
	headers := []string{"Name", "PID", "CPU%", "Memory"}
	widths := []int{nameW, pidW, cpuW, memW}
	right := []bool{false, true, true, true}
	if showThr {
		headers = append(headers, "Thr")
		widths = append(widths, thrW)
		right = append(right, true)
	}
	var rows [][]string
	for _, p := range procs {
		cpu := "–"
		if p.CPUValid {
			cpu = fmt.Sprintf("%.1f", p.CPUPercent)
		}
		row := []string{p.Name, fmt.Sprint(p.PID), cpu, format.Bytes(p.RSS)}
		if showThr {
			thr := "–"
			if p.Threads.OK {
				thr = fmt.Sprint(p.Threads.V)
			}
			row = append(row, thr)
		}
		rows = append(rows, row)
	}
	return components.Table(t, headers, rows, widths, right, -1)
}

func (o *Overview) bottom(c *Context, w, h int, focused bool) string {
	t := c.Theme
	alertW := 0
	if w >= 100 {
		alertW = w * 2 / 5
	}
	procW := w - alertW
	if alertW > 0 {
		procW--
	}
	cw, ch := components.PanelContentSize(procW, h)
	var body string
	if c.Snap.Processes.Valid {
		body = procTable(c, TopProcesses(c.Snap.Processes.List, max(ch-1, 1), false), cw, false)
	} else {
		body = errLine(t, "Process list", c.Snap.Processes.Err)
	}
	hint := t.Faint.Render("enter ") + t.Muted.Render("open")
	procs := components.Panel(t, "Top Processes", hint, body, procW, h, focused)
	if alertW == 0 {
		return procs
	}
	aw, ah := components.PanelContentSize(alertW, h)
	return components.Columns(1, procs, components.Panel(t, "Alerts", t.Faint.Render("a ")+t.Muted.Render("all"), AlertsList(c, aw, ah), alertW, h, false))
}

// AlertsList renders the active alerts compactly.
func AlertsList(c *Context, w, h int) string {
	t := c.Theme
	if !c.Cfg.AlertsEnabled {
		return t.Muted.Render("Alerts are disabled in Settings")
	}
	act := c.Alerts.Active()
	if len(act) == 0 {
		return t.Accent.Render(t.Sym.Check+" ") + t.Muted.Render("No active alerts")
	}
	var lines []string
	for _, a := range act {
		st := t.Warn
		if a.Severity == alerts.Critical {
			st = t.Crit
		}
		lines = append(lines, st.Render(t.Sym.Dot+" ")+t.Value.Render(a.Title))
		lines = append(lines, "  "+t.Muted.Render(a.Detail))
		if len(lines) >= h {
			break
		}
	}
	return strings.Join(lines, "\n")
}

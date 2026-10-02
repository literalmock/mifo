package pages

import (
	"fmt"
	"net"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
)

// Network shows interfaces, throughput and on-demand diagnostics.
type Network struct {
	table   table.Model
	ifaces  []metrics.Interface
	sel     string
	showAll bool

	diagRunning bool
	diag        *metrics.DiagResult
	topRunning  bool
	top         *metrics.NetTopResult

	rx, tx []float64
}

func NewNetwork() *Network {
	n := &Network{table: newNavTable()}
	n.table.SetColumns(placeholderCols(6))
	return n
}

func (*Network) ID() string    { return "network" }
func (*Network) Title() string { return "Network" }

func (*Network) Capturing() bool { return false }

var (
	keyDiag   = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "connectivity test"))
	keyNetTop = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "per-app traffic"))
	keyAll    = key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "all interfaces"))
)

func (*Network) Help(*Context) []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "select")), keyDiag, keyNetTop, keyAll}
}

func (n *Network) HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, keyDiag):
		if n.diagRunning {
			return nil, true
		}
		n.diagRunning = true
		return c.Ops.Diagnostics(), true
	case key.Matches(msg, keyNetTop):
		if n.topRunning {
			return nil, true
		}
		n.topRunning = true
		return c.Ops.NetTop(), true
	case key.Matches(msg, keyAll):
		n.showAll = !n.showAll
		n.rebuild(c)
		return nil, true
	}
	if !isTableKey(msg) {
		return nil, false
	}
	var cmd tea.Cmd
	n.table, cmd = n.table.Update(msg)
	if i := n.table.Cursor(); i >= 0 && i < len(n.ifaces) {
		n.sel = n.ifaces[i].Name
	}
	return cmd, true
}

func (n *Network) Update(c *Context, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case metrics.Snapshot:
		n.rebuild(c)
	case DiagnosticsMsg:
		n.diagRunning = false
		r := msg.Result
		n.diag = &r
	case NetTopMsg:
		n.topRunning = false
		r := msg.Result
		n.top = &r
	}
	return nil
}

// interesting reports whether an interface is worth showing by default.
func interesting(i metrics.Interface) bool {
	if i.Primary {
		return true
	}
	if !i.Up || i.Kind == "Loopback" {
		return false
	}
	if i.RxTotal > 0 || i.TxTotal > 0 {
		for _, a := range i.Addrs {
			if !strings.HasPrefix(a, "fe80:") {
				return true
			}
		}
		return i.RxBps > 0 || i.TxBps > 0
	}
	return false
}

func firstIPv4(addrs []string) string {
	for _, a := range addrs {
		if ip, _, err := net.ParseCIDR(a); err == nil && ip.To4() != nil {
			return a
		}
	}
	return ""
}

func (n *Network) rebuild(c *Context) {
	n.ifaces = n.ifaces[:0]
	for _, i := range c.Snap.Network.Interfaces {
		if n.showAll || interesting(i) {
			n.ifaces = append(n.ifaces, i)
		}
	}
	rows := make([]table.Row, len(n.ifaces))
	for k, i := range n.ifaces {
		status := "Down"
		switch {
		case i.Primary:
			status = "Primary"
		case i.Up && i.Running:
			status = "Active"
		case i.Up:
			status = "Up"
		}
		ip := firstIPv4(i.Addrs)
		if ip == "" {
			ip = "–"
		}
		rows[k] = table.Row{i.Name, i.Kind, status, ip, format.Rate(i.RxBps), format.Rate(i.TxBps)}
	}
	n.table.SetRows(rows)
	for k, i := range n.ifaces {
		if i.Name == n.sel {
			n.table.SetCursor(k)
			return
		}
	}
	n.table.SetCursor(min(n.table.Cursor(), max(len(rows)-1, 0)))
	if k := n.table.Cursor(); k >= 0 && k < len(n.ifaces) {
		n.sel = n.ifaces[k].Name
	}
}

func (n *Network) selected() (metrics.Interface, bool) {
	for _, i := range n.ifaces {
		if i.Name == n.sel {
			return i, true
		}
	}
	return metrics.Interface{}, false
}

func (n *Network) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t, ns := c.Theme, c.Snap.Network
	if !ns.Valid && ns.Err != nil {
		return components.Panel(t, "Network", "", errLine(t, "Network counters", ns.Err), w, min(h, 5), false)
	}
	if len(n.ifaces) == 0 && len(ns.Interfaces) > 0 {
		n.rebuild(c)
	}
	n.table.SetStyles(tableStyles(t, focused))
	tblH := min(len(n.ifaces)+4, max(h/3, 6))
	iw, ih := components.PanelContentSize(w, tblH)
	kindW := 20
	ipW := 18
	rateW := 11
	nameW := max(iw-kindW-ipW-2*rateW-9-2*6, 6)
	n.table.SetColumns([]table.Column{
		{Title: "Interface", Width: nameW}, {Title: "Type", Width: kindW}, {Title: "Status", Width: 9},
		{Title: "IPv4", Width: ipW}, {Title: "Down", Width: rateW}, {Title: "Up", Width: rateW},
	})
	n.table.SetWidth(iw)
	n.table.SetHeight(ih)
	scope := "active"
	if n.showAll {
		scope = "all"
	}
	info := t.Accent.Render(t.Sym.Down+format.Rate(ns.RxBps)) + " " + t.Second.Render(t.Sym.Up+format.Rate(ns.TxBps)) +
		t.Faint.Render(" "+t.Sym.Sep+" "+scope)
	tbl := components.Panel(t, "Interfaces", info, n.table.View(), w, tblH, focused)

	rest := h - tblH
	diagH := 0
	if rest >= 22 {
		diagH = min(12, rest-10)
	}
	midH := rest - diagH
	if midH < 6 {
		return components.Fit(tbl, w, h)
	}
	detW := w
	chW := 0
	if w >= 96 {
		detW = w * 2 / 5
		chW = w - detW - 1
	}
	mid := components.Panel(t, "Interface Details", "", n.details(c, detW-4), detW, midH, false)
	if chW > 0 {
		mid = components.Columns(1, mid, n.chart(c, chW, midH))
	}
	out := tbl + "\n" + mid
	if diagH > 0 {
		dw := w / 2
		pw := w - dw - 1
		out += "\n" + components.Columns(1,
			components.Panel(t, "Diagnostics", t.Faint.Render("t ")+t.Muted.Render("run"), n.diagBody(c, dw-4), dw, diagH, false),
			components.Panel(t, "Per-app Traffic", t.Faint.Render("n ")+t.Muted.Render("sample"), n.topBody(c, pw-4, diagH-2), pw, diagH, false))
	}
	return components.Fit(out, w, h)
}

func (n *Network) details(c *Context, w int) string {
	t := c.Theme
	i, ok := n.selected()
	if !ok {
		return t.Muted.Render("No interface selected")
	}
	state := t.Crit.Render(t.Sym.Dot + " Down")
	switch {
	case i.Primary:
		state = t.Accent.Render(t.Sym.Dot + " Connected (default route)")
	case i.Up && i.Running:
		state = t.Accent.Render(t.Sym.Dot + " Active")
	case i.Up:
		state = t.Warn.Render(t.Sym.Dot + " Up, no link")
	}
	lines := []string{
		t.Title.Render(i.Kind) + t.Muted.Render("  "+i.Name),
		state,
		"",
	}
	var v4, v6 []string
	for _, a := range i.Addrs {
		ip, ipn, err := net.ParseCIDR(a)
		if err != nil {
			continue
		}
		if ip.To4() != nil {
			ones, _ := ipn.Mask.Size()
			mask := net.IP(ipn.Mask).String()
			v4 = append(v4, fmt.Sprintf("%s/%d", ip, ones)+t.Faint.Render(" mask "+mask))
		} else {
			v6 = append(v6, a)
		}
	}
	if len(v4) == 0 {
		lines = append(lines, components.KV(t, "IPv4", t.Muted.Render("none"), w))
	}
	for k, a := range v4 {
		label := ""
		if k == 0 {
			label = "IPv4"
		}
		lines = append(lines, components.KV(t, label, a, w))
	}
	for k, a := range v6 {
		if k >= 3 {
			lines = append(lines, components.KV(t, "", t.Faint.Render(fmt.Sprintf("+%d more", len(v6)-3)), w))
			break
		}
		label := ""
		if k == 0 {
			label = "IPv6"
		}
		lines = append(lines, components.KV(t, label, format.Truncate(a, w-6), w))
	}
	mac := i.MAC
	if mac == "" {
		mac = t.Muted.Render("none")
	}
	lines = append(lines,
		components.KV(t, "MAC", mac, w),
		components.KV(t, "MTU", fmt.Sprint(i.MTU), w),
		components.KV(t, "Received", format.BytesSI(i.RxTotal), w),
		components.KV(t, "Sent", format.BytesSI(i.TxTotal), w),
		components.KV(t, "Errors in / out", fmt.Sprintf("%d / %d", i.RxErrs, i.TxErrs), w),
	)
	return strings.Join(lines, "\n")
}

func (n *Network) chart(c *Context, w, h int) string {
	t := c.Theme
	i, ok := n.selected()
	iw, ih := components.PanelContentSize(w, h)
	if !ok {
		return components.Panel(t, "Throughput", "", "", w, h, false)
	}
	r := c.Hist.Iface[i.Name]
	if r[0] == nil {
		return components.Panel(t, "Throughput", "", t.Muted.Render("Collecting history…"), w, h, false)
	}
	n.rx = series(c, r[0], n.rx)
	n.tx = series(c, r[1], n.tx)
	head := t.Accent.Render(t.Sym.Down+" "+format.Rate(i.RxBps)) + "   " + t.Second.Render(t.Sym.Up+" "+format.Rate(i.TxBps))
	body := head + "\n" + components.MirrorChart(t, n.rx, n.tx, iw, ih-1, t.P.Accent, t.P.Secondary, rateLabel)
	return components.Panel(t, "Throughput "+t.Sym.Sep+" "+i.Name, t.Muted.Render(windowLabel(c.Cfg.WindowSecs)), body, w, h, false)
}

func (n *Network) diagBody(c *Context, w int) string {
	t := c.Theme
	if n.diagRunning {
		return t.Accent.Render(t.Sym.Dot+" Running") + t.Muted.Render(" ping 1.1.1.1 ×5 and DNS lookup…")
	}
	if n.diag == nil {
		return t.Muted.Render("Press t to measure latency, packet loss and DNS\nresolution. MIFO never probes the network on its own.")
	}
	d := n.diag
	var lines []string
	if d.PingErr != nil {
		lines = append(lines, components.KV(t, "Ping "+d.Target, t.Crit.Render("failed: "+d.PingErr.Error()), w))
	} else {
		lat := t.Muted.Render("no replies")
		if d.AvgMs.OK {
			lat = fmt.Sprintf("%.1f ms", d.AvgMs.V) + t.Faint.Render(fmt.Sprintf("  min %.1f  max %.1f", d.MinMs.V, d.MaxMs.V))
		}
		lines = append(lines, components.KV(t, "Latency ("+d.Target+")", lat, w))
		if d.Loss.OK {
			st := t.Accent
			if d.Loss.V > 0 {
				st = t.Warn
			}
			if d.Loss.V >= 50 {
				st = t.Crit
			}
			lines = append(lines, components.KV(t, "Packet loss", st.Render(fmt.Sprintf("%.0f%%", d.Loss.V))+t.Faint.Render(fmt.Sprintf("  %d/%d replies", d.Received, d.Sent)), w))
		}
	}
	if d.DNSErr != nil {
		lines = append(lines, components.KV(t, "DNS "+d.DNSHost, t.Crit.Render("failed"), w), t.Faint.Render(format.Truncate(d.DNSErr.Error(), w)))
	} else if d.DNSTime.OK {
		lines = append(lines, components.KV(t, "DNS "+d.DNSHost, fmt.Sprintf("%.0f ms", float64(d.DNSTime.V.Microseconds())/1000), w))
		if len(d.DNSAddrs) > 0 {
			lines = append(lines, components.KV(t, "", t.Faint.Render(format.Truncate(strings.Join(d.DNSAddrs, ", "), w-2)), w))
		}
	}
	lines = append(lines, "", t.Faint.Render("Measured at "+d.FinishedAt.Format("15:04:05")))
	return strings.Join(lines, "\n")
}

func (n *Network) topBody(c *Context, w, h int) string {
	t := c.Theme
	if n.topRunning {
		return t.Accent.Render(t.Sym.Dot+" Sampling") + t.Muted.Render(" for one second with nettop…")
	}
	if n.top == nil {
		return t.Muted.Render("Press n for a one-second per-app snapshot.\nAttribution is best effort: macOS credits traffic made\nby system services on an app's behalf to the service.")
	}
	if n.top.Err != nil {
		return errLine(t, "Per-app traffic", n.top.Err)
	}
	if len(n.top.Entries) == 0 {
		return t.Muted.Render("No traffic during the sample (" + n.top.At.Format("15:04:05") + ")")
	}
	rateW := 11
	nameW := max(w-2*rateW-2, 8)
	var rows [][]string
	for k, e := range n.top.Entries {
		if k >= h-2 {
			break
		}
		rows = append(rows, []string{fmt.Sprintf("%s %d", e.Name, e.PID), format.Rate(e.RxBps), format.Rate(e.TxBps)})
	}
	tbl := components.Table(t, []string{"Process", "Down", "Up"}, rows, []int{nameW, rateW, rateW}, []bool{false, true, true}, -1)
	return tbl + "\n" + t.Faint.Render("Sampled "+n.top.At.Format("15:04:05")+" "+t.Sym.Sep+" best-effort attribution")
}

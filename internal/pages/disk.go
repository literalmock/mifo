package pages

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
	"github.com/literalmock/mifo/internal/styles"
)

// Disk shows mounted volumes and throughput.
type Disk struct {
	table   table.Model
	vols    []metrics.Volume
	sel     string // selected mount point
	r, wbuf []float64
}

func NewDisk() *Disk {
	d := &Disk{table: newNavTable()}
	d.table.SetColumns(placeholderCols(7))
	return d
}

func (*Disk) ID() string      { return "disk" }
func (*Disk) Title() string   { return "Disk" }
func (*Disk) Capturing() bool { return false }
func (*Disk) Help(*Context) []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "select volume")), keyWindow}
}

func (d *Disk) HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if !isTableKey(msg) {
		return nil, false
	}
	var cmd tea.Cmd
	d.table, cmd = d.table.Update(msg)
	if i := d.table.Cursor(); i >= 0 && i < len(d.vols) {
		d.sel = d.vols[i].Mount
	}
	return cmd, true
}

func (d *Disk) Update(c *Context, msg tea.Msg) tea.Cmd {
	if _, ok := msg.(metrics.Snapshot); ok {
		d.rebuild(c)
	}
	return nil
}

func textBar(t bool, pct float64, w int) string {
	full, empty := "━", "─"
	if !t {
		full, empty = "#", "-"
	}
	n := int(pct/100*float64(w) + 0.5)
	n = min(max(n, 0), w)
	return strings.Repeat(full, n) + strings.Repeat(empty, w-n)
}

func volumeLocation(v metrics.Volume) string {
	switch {
	case !v.Internal.OK:
		return "Unknown"
	case v.Internal.V:
		return "Internal"
	}
	return "External"
}

func (d *Disk) rebuild(c *Context) {
	d.vols = c.Snap.Disk.Volumes
	rows := make([]table.Row, len(d.vols))
	for i, v := range d.vols {
		rows[i] = table.Row{
			v.Name, v.Mount, v.FSType, volumeLocation(v),
			format.BytesSI(v.Total), format.BytesSI(v.Free),
			textBar(c.Theme.Sym.Unicode, v.UsedPercent, 10) + fmt.Sprintf(" %3.0f%%", v.UsedPercent),
		}
	}
	d.table.SetRows(rows)
	for i, v := range d.vols {
		if v.Mount == d.sel {
			d.table.SetCursor(i)
			return
		}
	}
	d.table.SetCursor(min(d.table.Cursor(), max(len(rows)-1, 0)))
	if i := d.table.Cursor(); i >= 0 && i < len(d.vols) {
		d.sel = d.vols[i].Mount
	}
}

func (d *Disk) View(c *Context, w, h int, focused bool) string {
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	t, ds := c.Theme, c.Snap.Disk
	d.rebuild(c) // cheap; keeps glyphs in sync with the theme
	d.table.SetStyles(tableStyles(t, focused))
	tblH := min(len(d.vols)+4, max(h/2-1, 6))
	iw, ih := components.PanelContentSize(w, tblH)
	nameW := max(iw-(18+6+8+8+8+15)-2*7, 8)
	mountW := 18
	d.table.SetColumns([]table.Column{
		{Title: "Volume", Width: nameW}, {Title: "Mount point", Width: mountW}, {Title: "FS", Width: 6},
		{Title: "Location", Width: 8}, {Title: "Size", Width: 8}, {Title: "Free", Width: 8}, {Title: "Used", Width: 15},
	})
	d.table.SetWidth(iw)
	d.table.SetHeight(ih)
	var vbody string
	if len(d.vols) == 0 {
		vbody = errLine(t, "Volume list", ds.Err)
	} else {
		vbody = d.table.View()
	}
	vols := components.Panel(t, "Volumes", t.Muted.Render(fmt.Sprintf("%d mounted", len(d.vols))), vbody, w, tblH, focused)

	restH := h - tblH
	if restH < 6 {
		return components.Fit(vols, w, h)
	}
	detW := w
	ioW := 0
	if w >= 96 {
		detW = w * 2 / 5
		ioW = w - detW - 1
	}
	det := components.Panel(t, "Volume Details", "", d.details(c, detW-4), detW, restH, false)
	bottom := det
	if ioW > 0 {
		bottom = components.Columns(1, det, d.ioPanel(c, ioW, restH))
	}
	return components.Fit(vols+"\n"+bottom, w, h)
}

func (d *Disk) details(c *Context, w int) string {
	t := c.Theme
	var v metrics.Volume
	found := false
	for _, x := range d.vols {
		if x.Mount == d.sel {
			v, found = x, true
		}
	}
	if !found {
		return t.Muted.Render("No volume selected")
	}
	col := t.LevelColor(v.UsedPercent, 85, 95)
	yn := func(o metrics.Opt[bool], yes, no string) string {
		if !o.OK {
			return t.Muted.Render("unknown")
		}
		if o.V {
			return yes
		}
		return no
	}
	smart := t.Muted.Render("not reported")
	switch v.SMART {
	case "Verified":
		smart = t.Accent.Render(t.Sym.Check + " Verified")
	case "":
	case "Not Supported":
		smart = t.Muted.Render("not supported")
	default:
		smart = t.Crit.Render(v.SMART)
	}
	access := "Read / write"
	if v.ReadOnly {
		access = "Read-only"
	}
	lines := []string{
		t.Title.Render(v.Name) + t.Muted.Render("  "+v.Mount),
		"",
		t.BigValue.Render(format.BytesSI(v.Free)) + t.Muted.Render(" free of "+format.BytesSI(v.Total)),
		components.Bar(t, v.UsedPercent, w, col),
		components.KV(t, "Used", lipgloss.NewStyle().Foreground(col).Render(fmt.Sprintf("%s (%s)", format.BytesSI(v.Used), format.Percent(v.UsedPercent))), w),
		"",
		components.KV(t, "File system", v.FSType, w),
		components.KV(t, "Device", v.Device, w),
		components.KV(t, "Location", volumeLocation(v), w),
		components.KV(t, "Media", yn(v.SolidState, "Solid state", "Rotational"), w),
		components.KV(t, "Connection", orUnknown(t, v.Protocol), w),
		components.KV(t, "Access", access, w),
		components.KV(t, "SMART health", smart, w),
	}
	if v.Mount == "/" {
		lines = append(lines, "", t.Faint.Render("The sealed system volume shares space with Data"), t.Faint.Render("in the same APFS container."))
	}
	return strings.Join(lines, "\n")
}

func orUnknown(t styles.Theme, s string) string {
	if s == "" {
		return t.Muted.Render("unknown")
	}
	return s
}

func (d *Disk) ioPanel(c *Context, w, h int) string {
	t, ds := c.Theme, c.Snap.Disk
	iw, ih := components.PanelContentSize(w, h)
	if !ds.IOValid {
		msg := t.Muted.Render("Measuring throughput…")
		if ds.IOErr != nil {
			msg = errLine(t, "Disk I/O counters", ds.IOErr)
		}
		return components.Panel(t, "Disk Activity", "", msg, w, h, false)
	}
	stats := []string{
		t.Accent.Render(t.Sym.Dot+" Read  ") + t.Value.Bold(true).Render(format.Rate(ds.ReadBps)) + t.Muted.Render(fmt.Sprintf("  %.0f IOPS", ds.ReadIOPS)),
		t.Second.Render(t.Sym.Dot+" Write ") + t.Value.Bold(true).Render(format.Rate(ds.WriteBps)) + t.Muted.Render(fmt.Sprintf("  %.0f IOPS", ds.WriteIOPS)),
		t.Faint.Render(fmt.Sprintf("Since boot: %s read %s %s written", format.BytesSI(ds.ReadTotal), t.Sym.Sep, format.BytesSI(ds.WriteTot))),
	}
	chartH := ih - len(stats) - 1
	body := strings.Join(stats, "\n")
	if chartH >= 2 {
		d.r = series(c, c.Hist.DiskR, d.r)
		d.wbuf = series(c, c.Hist.DiskW, d.wbuf)
		body += "\n\n" + components.MirrorChart(t, d.r, d.wbuf, iw, chartH, t.P.Accent, t.P.Secondary, rateLabel)
	}
	return components.Panel(t, "Disk Activity", t.Muted.Render(windowLabel(c.Cfg.WindowSecs)), body, w, h, false)
}

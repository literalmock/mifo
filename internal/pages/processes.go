package pages

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/literalmock/mifo/internal/components"
	"github.com/literalmock/mifo/internal/format"
	"github.com/literalmock/mifo/internal/metrics"
)

// SortKey selects the process ordering.
type SortKey int

const (
	SortCPU SortKey = iota
	SortMemory
	SortPID
	SortName
	SortThreads
)

func (s SortKey) String() string {
	return [...]string{"CPU", "Memory", "PID", "Name", "Threads"}[s]
}

// SortProcesses orders list in place. Ties break by PID for stability.
func SortProcesses(list []metrics.Process, by SortKey, desc bool) {
	less := func(a, b metrics.Process) int {
		switch by {
		case SortCPU:
			return cmpF(a.CPUPercent, b.CPUPercent)
		case SortMemory:
			return cmpU(a.RSS, b.RSS)
		case SortName:
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case SortThreads:
			return cmpI(a.Threads.V, b.Threads.V)
		}
		return cmpI(a.PID, b.PID)
	}
	sort.SliceStable(list, func(i, j int) bool {
		c := less(list[i], list[j])
		if c == 0 {
			return list[i].PID < list[j].PID
		}
		if desc {
			return c > 0
		}
		return c < 0
	})
}

func cmpF(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
func cmpU(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
func cmpI(a, b int) int { return a - b }

// FilterProcesses returns processes whose name, path, user or PID matches
// the query (case-insensitive). uid >= 0 restricts to that user.
func FilterProcesses(list []metrics.Process, query string, uid int) []metrics.Process {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]metrics.Process, 0, len(list))
	for _, p := range list {
		if uid >= 0 && p.UID != uid {
			continue
		}
		if q != "" &&
			!strings.Contains(strings.ToLower(p.Name), q) &&
			!strings.Contains(strings.ToLower(p.Path), q) &&
			!strings.Contains(strings.ToLower(p.User), q) &&
			!strings.HasPrefix(strconv.Itoa(p.PID), q) {
			continue
		}
		out = append(out, p)
	}
	return out
}

type confirmState struct {
	proc  metrics.Process
	force bool
}

// Processes is the process manager.
type Processes struct {
	table     table.Model
	search    textinput.Model
	searching bool
	sortBy    SortKey
	desc      bool
	mineOnly  bool
	uid       int

	rows   []metrics.Process
	sel    metrics.ProcKey
	hasSel bool

	showDetail    bool
	detail        metrics.ProcessDetail
	detailLoading bool
	detailWant    metrics.ProcKey

	confirm *confirmState
	result  string
	resErr  bool
	resAt   time.Time

	themeName string
}

func NewProcesses() *Processes {
	km := table.DefaultKeyMap()
	km.LineUp = key.NewBinding(key.WithKeys("up", "k"))
	km.LineDown = key.NewBinding(key.WithKeys("down", "j"))
	km.PageUp = key.NewBinding(key.WithKeys("pgup"))
	km.PageDown = key.NewBinding(key.WithKeys("pgdown", "space"))
	km.HalfPageUp = key.NewBinding(key.WithKeys("ctrl+u"))
	km.HalfPageDown = key.NewBinding(key.WithKeys("ctrl+d"))
	km.GotoTop = key.NewBinding(key.WithKeys("home", "g"))
	km.GotoBottom = key.NewBinding(key.WithKeys("end", "G"))
	tb := table.New(table.WithKeyMap(km), table.WithFocused(true))
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "filter by name, path, user or PID"
	ti.CharLimit = 64
	p := &Processes{table: tb, search: ti, desc: true, uid: -1}
	p.table.SetColumns(p.cols(100, "↓"))
	return p
}

func (*Processes) ID() string    { return "processes" }
func (*Processes) Title() string { return "Processes" }

func (p *Processes) Capturing() bool { return p.searching || p.confirm != nil }

var (
	keySortCPU  = key.NewBinding(key.WithKeys("c"), key.WithHelp("c/m/i/n/t", "sort"))
	keySortMem  = key.NewBinding(key.WithKeys("m"))
	keySortPID  = key.NewBinding(key.WithKeys("i"))
	keySortName = key.NewBinding(key.WithKeys("n"))
	keySortThr  = key.NewBinding(key.WithKeys("t"))
	keyMine     = key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "mine only"))
	keyTerm     = key.NewBinding(key.WithKeys("x", "delete"), key.WithHelp("x", "terminate"))
	keyKill     = key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "force quit"))
	keyYes      = key.NewBinding(key.WithKeys("y", "Y", "enter"), key.WithHelp("y", "confirm"))
	keyNo       = key.NewBinding(key.WithKeys("n", "N", "esc"), key.WithHelp("n/esc", "cancel"))
)

func (p *Processes) Help(*Context) []key.Binding {
	switch {
	case p.confirm != nil:
		return []key.Binding{keyYes, keyNo}
	case p.searching:
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "apply")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear")),
		}
	}
	return []key.Binding{keyDown, keySlash, keySortCPU, keyEnter, keyTerm, keyKill, keyMine}
}

func (p *Processes) selected() (metrics.Process, bool) {
	i := p.table.Cursor()
	if i < 0 || i >= len(p.rows) {
		return metrics.Process{}, false
	}
	return p.rows[i], true
}

func (p *Processes) setSort(k SortKey) {
	if p.sortBy == k {
		p.desc = !p.desc
	} else {
		p.sortBy = k
		p.desc = k != SortName && k != SortPID
	}
}

func (p *Processes) HandleKey(c *Context, msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if p.confirm != nil {
		switch {
		case key.Matches(msg, keyYes):
			cs := p.confirm
			p.confirm = nil
			return c.Ops.Signal(cs.proc.Key(), cs.force), true
		case key.Matches(msg, keyNo):
			p.confirm = nil
		}
		return nil, true
	}
	if p.searching {
		switch msg.String() {
		case "esc":
			p.searching = false
			p.search.SetValue("")
			p.search.Blur()
			p.rebuild(c)
			return nil, true
		case "enter", "down", "up":
			p.searching = false
			p.search.Blur()
			return nil, true
		}
		var cmd tea.Cmd
		p.search, cmd = p.search.Update(msg)
		p.rebuild(c)
		p.table.GotoTop()
		p.syncSel()
		return cmd, true
	}
	switch {
	case key.Matches(msg, keySlash):
		p.searching = true
		return p.search.Focus(), true
	case key.Matches(msg, keySortCPU):
		p.setSort(SortCPU)
	case key.Matches(msg, keySortMem):
		p.setSort(SortMemory)
	case key.Matches(msg, keySortPID):
		p.setSort(SortPID)
	case key.Matches(msg, keySortName):
		p.setSort(SortName)
	case key.Matches(msg, keySortThr):
		p.setSort(SortThreads)
	case key.Matches(msg, keyMine):
		p.mineOnly = !p.mineOnly
	case key.Matches(msg, keyEnter):
		p.showDetail = !p.showDetail
		return p.maybeFetch(c), true
	case key.Matches(msg, keyTerm), key.Matches(msg, keyKill):
		if proc, ok := p.selected(); ok {
			p.confirm = &confirmState{proc: proc, force: key.Matches(msg, keyKill)}
		}
		return nil, true
	case msg.String() == "esc":
		if p.showDetail {
			p.showDetail = false
			return nil, true
		}
		if p.search.Value() != "" {
			p.search.SetValue("")
			p.rebuild(c)
			return nil, true
		}
		return nil, false
	default:
		before := p.table.Cursor()
		var cmd tea.Cmd
		p.table, cmd = p.table.Update(msg)
		if p.table.Cursor() != before {
			p.syncSel()
			return tea.Batch(cmd, p.maybeFetch(c)), true
		}
		return cmd, isTableKey(msg)
	}
	p.rebuild(c)
	return nil, true
}

func isTableKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "up", "down", "j", "k", "pgup", "pgdown", "space", "home", "end", "g", "G", "ctrl+u", "ctrl+d":
		return true
	}
	return false
}

func (p *Processes) syncSel() {
	if proc, ok := p.selected(); ok {
		p.sel, p.hasSel = proc.Key(), true
	}
}

func (p *Processes) maybeFetch(c *Context) tea.Cmd {
	if !p.showDetail {
		return nil
	}
	proc, ok := p.selected()
	if !ok || (p.detail.Key == proc.Key() && p.detail.Err == nil && !p.detail.Fetched.IsZero()) {
		return nil
	}
	if p.detailLoading && p.detailWant == proc.Key() {
		return nil
	}
	p.detailLoading, p.detailWant = true, proc.Key()
	return c.Ops.ProcessDetail(proc.Key())
}

func (p *Processes) Update(c *Context, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case metrics.Snapshot:
		p.rebuild(c)
	case ProcessDetailMsg:
		if msg.Detail.Key == p.detailWant {
			p.detailLoading = false
		}
		p.detail = msg.Detail
	case SignalResultMsg:
		name := fmt.Sprintf("PID %d", msg.Key.PID)
		for _, r := range c.Snap.Processes.List {
			if r.Key() == msg.Key {
				name = fmt.Sprintf("%s (PID %d)", r.Name, r.PID)
			}
		}
		verb := "Sent SIGTERM to "
		if msg.Force {
			verb = "Force quit (SIGKILL) "
		}
		p.resAt = c.Now
		if msg.Err != nil {
			p.result, p.resErr = msg.Err.Error(), true
			if errors.Is(msg.Err, metrics.ErrPermission) {
				p.result = "macOS denied the request for " + name + ". It belongs to another user or is protected by System Integrity Protection; MIFO never escalates privileges."
			}
			return func() tea.Msg { return StatusMsg{Text: "Could not signal " + name, Error: true} }
		}
		p.result, p.resErr = verb+name, false
		return func() tea.Msg { return StatusMsg{Text: verb + name} }
	default:
		if p.searching {
			var cmd tea.Cmd
			p.search, cmd = p.search.Update(msg)
			return cmd
		}
	}
	return nil
}

// rebuild re-filters and re-sorts the process list, keeping the selection on
// the same process (by PID + start time) across refreshes.
func (p *Processes) rebuild(c *Context) {
	if !c.HasSnap {
		return
	}
	uid := -1
	if p.mineOnly {
		uid = selfUID()
	}
	rows := FilterProcesses(c.Snap.Processes.List, p.search.Value(), uid)
	SortProcesses(rows, p.sortBy, p.desc)
	p.rows = rows
	trs := make([]table.Row, len(rows))
	for i, r := range rows {
		cpu := "–"
		if r.CPUValid {
			cpu = fmt.Sprintf("%.1f", r.CPUPercent)
		}
		thr := "–"
		if r.Threads.OK {
			thr = strconv.Itoa(r.Threads.V)
		}
		trs[i] = table.Row{r.Name, fmt.Sprintf("%7d", r.PID), fmt.Sprintf("%7s", cpu), fmt.Sprintf("%10s", format.Bytes(r.RSS)), fmt.Sprintf("%5s", thr), r.User, r.Status()}
	}
	p.table.SetRows(trs)
	if p.hasSel {
		for i, r := range rows {
			if r.Key() == p.sel {
				p.table.SetCursor(i)
				return
			}
		}
	}
	p.table.SetCursor(min(p.table.Cursor(), max(len(rows)-1, 0)))
	p.syncSel()
}

func (p *Processes) applyStyles(c *Context) {
	if p.themeName == c.Theme.P.Name+fmt.Sprint(c.Theme.Sym.Unicode) {
		return
	}
	p.themeName = c.Theme.P.Name + fmt.Sprint(c.Theme.Sym.Unicode)
	t := c.Theme
	st := table.DefaultStyles()
	st.Header = lipgloss.NewStyle().Foreground(t.P.Muted).Bold(true).Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(t.P.Border)
	if !t.Sym.Unicode {
		st.Header = st.Header.BorderStyle(lipgloss.ASCIIBorder())
	}
	// Cells carry no colours so the selection background spans the row.
	st.Cell = lipgloss.NewStyle().Padding(0, 1)
	st.Selected = lipgloss.NewStyle().Background(t.P.AccentBg).Foreground(t.P.Text).Bold(true)
	p.table.SetStyles(st)
	ts := textinput.DefaultStyles(t.P.Dark)
	ts.Focused.Prompt = t.Accent
	ts.Blurred.Prompt = t.Faint
	ts.Focused.Text = t.Value
	ts.Blurred.Text = t.Muted
	ts.Focused.Placeholder = t.Faint
	ts.Blurred.Placeholder = t.Faint
	p.search.SetStyles(ts)
}

func (p *Processes) View(c *Context, w, h int, focused bool) string {
	t := c.Theme
	if !c.HasSnap {
		return waiting(c, w, h)
	}
	p.applyStyles(c)
	if !c.Snap.Processes.Valid {
		return components.Panel(t, "Processes", "", errLine(t, "Process list", c.Snap.Processes.Err), w, min(h, 5), false)
	}
	if len(p.rows) == 0 && p.search.Value() == "" && !p.mineOnly {
		p.rebuild(c)
	}

	detailW, detailH := 0, 0
	if p.showDetail {
		if w >= 120 {
			detailW = 46
		} else {
			detailH = min(13, h/2)
		}
	}
	listW := w - detailW
	if detailW > 0 {
		listW--
	}
	listH := h - detailH

	// Search / status line.
	arrow := t.Sym.Down
	if !p.desc {
		arrow = t.Sym.Up
	}
	var bar string
	if p.searching || p.search.Value() != "" {
		p.search.SetWidth(max(listW-40, 10))
		bar = p.search.View()
	} else {
		bar = t.Faint.Render("/ search")
	}
	info := t.Muted.Render(fmt.Sprintf("%d of %d", len(p.rows), len(c.Snap.Processes.List))) +
		t.Faint.Render(" "+t.Sym.Sep+" sort ") + t.Accent.Render(p.sortBy.String()+" "+arrow)
	if p.mineOnly {
		info += t.Faint.Render(" "+t.Sym.Sep+" ") + t.Second.Render("mine")
	}
	iw, ih := components.PanelContentSize(listW, listH)
	topLine := spreadLine(bar, info, iw)

	// Result line (permission errors etc.) is shown for 15 seconds.
	var resLine string
	if p.result != "" && c.Now.Sub(p.resAt) < 15*time.Second {
		st := t.Accent
		if p.resErr {
			st = t.Warn
		}
		resLine = st.Render(format.Truncate(p.result, iw))
	}
	tableH := ih - 1
	if resLine != "" {
		tableH--
	}
	p.table.SetColumns(p.cols(iw, arrow))
	p.table.SetWidth(iw)
	p.table.SetHeight(max(tableH, 3))
	body := topLine + "\n" + p.table.View()
	if resLine != "" {
		body += "\n" + resLine
	}
	list := components.Panel(t, "Process Manager", t.Faint.Render(fmt.Sprintf("%d threads", c.Snap.Processes.Threads)), body, listW, listH, focused)

	out := list
	if detailW > 0 {
		out = components.Columns(1, list, components.Panel(t, "Details", "", p.detailBody(c, detailW-4), detailW, h, false))
	} else if detailH > 0 {
		out = list + "\n" + components.Panel(t, "Details", "", p.detailBody(c, w-4), w, detailH, false)
	}
	out = components.Fit(out, w, h)
	if p.confirm != nil {
		out = centerOverlay(out, p.confirmView(c, min(w-4, 64)), w, h)
	}
	return out
}

func (p *Processes) cols(w int, arrow string) []table.Column {
	// Each column has 2 cells of padding.
	pid, cpu, mem, thr, user, status := 7, 7, 10, 7, 12, 9
	fixed := pid + cpu + mem + 2*4
	showThr, showUser, showStatus := w >= 70, w >= 84, w >= 98
	if showThr {
		fixed += thr + 2
	}
	if showUser {
		fixed += user + 2
	}
	if showStatus {
		fixed += status + 2
	}
	name := max(w-fixed, 10)
	title := func(s string, k SortKey, w int) string {
		if p.sortBy == k {
			s = arrow + " " + s
		}
		if w > 0 {
			return fmt.Sprintf("%*s", w, s) // right-align numeric headers
		}
		return s
	}
	cols := []table.Column{
		{Title: title("Name", SortName, 0), Width: name},
		{Title: title("PID", SortPID, pid), Width: pid},
		{Title: title("CPU%", SortCPU, cpu), Width: cpu},
		{Title: title("Memory", SortMemory, mem), Width: mem},
		{Title: title("Thr", SortThreads, 5), Width: 0},
		{Title: "User", Width: 0},
		{Title: "Status", Width: 0},
	}
	if showThr {
		cols[4].Width = thr
	}
	if showUser {
		cols[5].Width = user
	}
	if showStatus {
		cols[6].Width = status
	}
	return cols
}

func (p *Processes) detailBody(c *Context, w int) string {
	t := c.Theme
	proc, ok := p.selected()
	if !ok {
		return t.Muted.Render("No process selected")
	}
	parent := fmt.Sprint(proc.PPID)
	for _, q := range c.Snap.Processes.List {
		if q.PID == proc.PPID {
			parent += t.Muted.Render(" " + q.Name)
			break
		}
	}
	cpu := t.Muted.Render("measuring")
	if proc.CPUValid {
		cpu = fmt.Sprintf("%.1f%%", proc.CPUPercent)
	}
	thr := t.Muted.Render("unavailable")
	if proc.Threads.OK {
		thr = fmt.Sprint(proc.Threads.V)
	}
	started := t.Muted.Render("unknown")
	if !proc.Start.IsZero() {
		started = proc.Start.Format("Jan 2 15:04") + t.Muted.Render(" ("+format.Uptime(c.Now.Sub(proc.Start))+")")
	}
	lines := []string{
		t.Title.Render(format.Truncate(proc.Name, w)),
		"",
		components.KV(t, "PID", fmt.Sprint(proc.PID), w),
		components.KV(t, "Parent PID", parent, w),
		components.KV(t, "User", fmt.Sprintf("%s (%d)", proc.User, proc.UID), w),
		components.KV(t, "Status", proc.Status()+t.Faint.Render(" "+proc.State), w),
		components.KV(t, "CPU", cpu, w),
		components.KV(t, "CPU time", format.CPUTime(proc.CPUTime), w),
		components.KV(t, "Memory (RSS)", format.Bytes(proc.RSS), w),
		components.KV(t, "Virtual size", format.Bytes(proc.VSZ), w),
		components.KV(t, "Threads", thr, w),
		components.KV(t, "Started", started, w),
		"",
		t.Label.Render("Executable"),
	}
	lines = append(lines, wrapPlain(t.Value, proc.Path, w)...)
	lines = append(lines, "", t.Label.Render("Command line"))
	switch {
	case p.detailLoading && p.detailWant == proc.Key():
		lines = append(lines, t.Faint.Render("loading…"))
	case p.detail.Key == proc.Key() && p.detail.Err != nil:
		lines = append(lines, t.Warn.Render(p.detail.Err.Error()))
	case p.detail.Key == proc.Key():
		lines = append(lines, wrapPlain(t.Muted, p.detail.Args, w)...)
	default:
		lines = append(lines, t.Faint.Render("—"))
	}
	return strings.Join(lines, "\n")
}

func wrapPlain(st lipgloss.Style, s string, w int) []string {
	if w <= 0 {
		return nil
	}
	var out []string
	r := []rune(s)
	for len(r) > 0 && len(out) < 6 {
		n := min(w, len(r))
		out = append(out, st.Render(string(r[:n])))
		r = r[n:]
	}
	if len(r) > 0 && len(out) > 0 {
		out[len(out)-1] = st.Render(format.Truncate(string([]rune(s)[(len(out)-1)*w:]), w))
	}
	return out
}

func (p *Processes) confirmView(c *Context, w int) string {
	t := c.Theme
	cs := p.confirm
	title := "Terminate process?"
	desc := "Sends SIGTERM so the process can save its work and exit cleanly."
	action := "Terminate"
	accent := t.P.Warn
	if cs.force {
		title = "Force quit process?"
		desc = "Sends SIGKILL. The process stops immediately and any unsaved work is lost."
		action = "Force Quit"
		accent = t.P.Crit
	}
	as := lipgloss.NewStyle().Foreground(accent).Bold(true)
	body := as.Render(title) + "\n\n" +
		t.Value.Render(cs.proc.Name) + t.Muted.Render(fmt.Sprintf("  PID %d %s %s", cs.proc.PID, t.Sym.Sep, cs.proc.User)) + "\n" +
		t.Faint.Render(format.TruncateLeft(cs.proc.Path, w-8)) + "\n\n" +
		lipgloss.NewStyle().Foreground(t.P.Muted).Width(w-6).Render(desc) + "\n\n" +
		as.Render("[y] "+action) + "    " + t.Muted.Render("[n] Cancel")
	return lipgloss.NewStyle().Border(t.Sym.Border).BorderForeground(accent).
		Background(t.P.Surface).Padding(1, 2).Width(w).Render(body)
}

// centerOverlay composites fg centred over base.
func centerOverlay(base, fg string, w, h int) string {
	fw, fh := lipgloss.Size(fg)
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(fg).X(max((w-fw)/2, 0)).Y(max((h-fh)/2, 0)).Z(1),
	)
	return comp.Render()
}

func spreadLine(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return components.Fit(left, w, 1)
	}
	return left + strings.Repeat(" ", gap) + right
}
